package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config holds the configuration loaded from environment
type Config struct {
	GithubToken  string
	BuildPath    string
	BuildCommand string
}

// GitHubReleaser manages GitHub releases
type GitHubReleaser struct {
	token     string
	headers   map[string]string
	repoName  string
	ownerName string
}

// NewGitHubReleaser creates a new GitHubReleaser instance
func NewGitHubReleaser(config Config) (*GitHubReleaser, error) {
	if config.GithubToken == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	// Get repo info from git config
	repoURL, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git remote URL: %w", err)
	}

	urlParts := strings.Split(strings.TrimSpace(string(repoURL)), "/")
	repoName := strings.TrimSuffix(urlParts[len(urlParts)-1], ".git")
	ownerParts := strings.Split(urlParts[len(urlParts)-2], ":")
	ownerName := ownerParts[len(ownerParts)-1]

	return &GitHubReleaser{
		token: config.GithubToken,
		headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", config.GithubToken),
			"Accept":        "application/vnd.github.v3+json",
		},
		repoName:  repoName,
		ownerName: ownerName,
	}, nil
}

// makeRequest makes an HTTP request to the GitHub API
func (g *GitHubReleaser) makeRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Set default headers
	for k, v := range g.headers {
		req.Header.Set(k, v)
	}

	// Set additional headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	return client.Do(req)
}

// RunBuild executes the build command
func (g *GitHubReleaser) RunBuild(buildCmd string) error {
	fmt.Println("Building project...")
	cmdParts := strings.Fields(buildCmd)
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CreateZip creates a ZIP file from the build directory
func (g *GitHubReleaser) CreateZip(buildPath, outputFile string) error {
	fmt.Printf("Creating ZIP archive from %s...\n", buildPath)

	if _, err := os.Stat(buildPath); os.IsNotExist(err) {
		return fmt.Errorf("build directory %s not found", buildPath)
	}

	zipFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	return filepath.Walk(buildPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(buildPath, path)
		if err != nil {
			return err
		}

		file, err := archive.Create(relPath)
		if err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		_, err = io.Copy(file, src)
		return err
	})
}

// GenerateChangelog generates a changelog from git commits
func (g *GitHubReleaser) GenerateChangelog() (string, error) {
	// Try to get the last tag
	lastTag, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output()

	var commits []byte
	if err == nil {
		// Get commits since last tag
		commits, err = exec.Command("git", "log",
			fmt.Sprintf("%s..HEAD", strings.TrimSpace(string(lastTag))),
			"--pretty=format:- %s").Output()
	} else {
		// If no tags exist, get all commits
		commits, err = exec.Command("git", "log", "--pretty=format:- %s").Output()
	}

	if err != nil {
		return "", err
	}

	return string(commits), nil
}

// CreateRelease creates a GitHub release and uploads the ZIP file
func (g *GitHubReleaser) CreateRelease(version, zipFile string) error {
	fmt.Printf("Creating GitHub release %s...\n", version)

	changelog, err := g.GenerateChangelog()
	if err != nil {
		return fmt.Errorf("failed to generate changelog: %w", err)
	}

	// Create release
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases",
		g.ownerName, g.repoName)

	releaseData := map[string]interface{}{
		"tag_name":   version,
		"name":       fmt.Sprintf("Release %s", version),
		"body":       changelog,
		"draft":      false,
		"prerelease": false,
	}

	jsonData, err := json.Marshal(releaseData)
	if err != nil {
		return err
	}

	headers := map[string]string{"Content-Type": "application/json"}
	resp, err := g.makeRequest("POST", releaseURL, bytes.NewBuffer(jsonData), headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create release: %s", body)
	}

	var release struct {
		UploadURL string `json:"upload_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}

	// Upload asset
	fmt.Println("Uploading release asset...")
	uploadURL := strings.Split(release.UploadURL, "{")[0]
	uploadURL = fmt.Sprintf("%s?name=%s", uploadURL, filepath.Base(zipFile))

	file, err := os.Open(zipFile)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(zipFile))
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	writer.Close()

	headers = map[string]string{"Content-Type": writer.FormDataContentType()}
	resp, err = g.makeRequest("POST", uploadURL, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload asset: %s", body)
	}

	return nil
}

// LoadConfig loads configuration from environment file
func LoadConfig(envFile string) (Config, error) {
	config := Config{}

	data, err := os.ReadFile(envFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return config, err
		}
		fmt.Printf("Warning: %s not found\n", envFile)
		// Don't return here - continue to check environment variables
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		switch key {
		case "GITHUB_TOKEN":
			config.GithubToken = value
		case "BUILD_PATH":
			config.BuildPath = value
		case "BUILD_COMMAND":
			config.BuildCommand = value
		}
	}

	// Check environment variables if not found in file
	if config.GithubToken == "" {
		config.GithubToken = os.Getenv("GITHUB_TOKEN")
	}
	if config.BuildPath == "" {
		config.BuildPath = os.Getenv("BUILD_PATH")
	}
	if config.BuildCommand == "" {
		config.BuildCommand = os.Getenv("BUILD_COMMAND")
	}

	// Validate required fields
	var missingFields []string
	if config.GithubToken == "" {
		missingFields = append(missingFields, "GITHUB_TOKEN")
	}
	if config.BuildPath == "" {
		missingFields = append(missingFields, "BUILD_PATH")
	}
	if config.BuildCommand == "" {
		missingFields = append(missingFields, "BUILD_COMMAND")
	}

	if len(missingFields) > 0 {
		return config, fmt.Errorf("missing required configuration: %s", strings.Join(missingFields, ", "))
	}

	return config, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <version>")
		fmt.Println("Example: go run main.go v1.0.0")
		fmt.Println("\nNote: Create a .release.env file with your configuration:")
		fmt.Println("GITHUB_TOKEN=your-token-here")
		fmt.Println("BUILD_PATH=dist")
		fmt.Println("BUILD_COMMAND=npm run build")
		os.Exit(1)
	}

	version := os.Args[1]
	if !strings.HasPrefix(version, "v") {
		fmt.Println("Version must start with 'v' (e.g., v1.0.0)")
		os.Exit(1)
	}

	config, err := LoadConfig(".release.env")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	releaser, err := NewGitHubReleaser(config)
	if err != nil {
		fmt.Printf("Error creating releaser: %v\n", err)
		os.Exit(1)
	}

	zipFile := "release.zip"
	defer os.Remove(zipFile) // Cleanup

	// Run build
	if err := releaser.RunBuild(config.BuildCommand); err != nil {
		fmt.Printf("Build failed: %v\n", err)
		os.Exit(1)
	}

	// Create ZIP
	if err := releaser.CreateZip(config.BuildPath, zipFile); err != nil {
		fmt.Printf("Failed to create ZIP: %v\n", err)
		os.Exit(1)
	}

	// Create release
	if err := releaser.CreateRelease(version, zipFile); err != nil {
		fmt.Printf("Failed to create release: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created release %s\n", version)
}
