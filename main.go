package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type AgentMetadata struct {
	Schema        string     `json:"schema"`
	Configuration ConfigJson `json:"configuration"`
	Metadata      string     `json:"metadata"`
}

type ConfigYaml struct {
	Name        string `yaml:"name"`
	Slug        string `yaml:"slug"`
	Platform    string `yaml:"platform"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Version     string `yaml:"version"`
	Schema      string `yaml:"schema"`
}

type ConfigJson struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Platform    string `json:"platform"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Version     string `json:"version"`
}

type ConfigFile struct {
	Configs []ConfigYaml `yaml:"configs"`
}

type GitHubConfig struct {
	AgentRepo   string
	GitHubToken string
	Branch      string // Add branch field
}

type GitHubClient struct {
	client *http.Client
	token  string
}

type GitHubContent struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func main() {
	config, err := loadGitHubConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "::error::Error loading config: %v\n", err)
		os.Exit(1)
	}

	configs, err := readConfigsFileToJson(
		config.AgentRepo,
		config.GitHubToken,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "::error::Error reading configs: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("::notice::Successfully fetched and validated agent metadata")
	printConfigs(configs)
}

func printConfigs(configs []ConfigJson) {
	for i, config := range configs {
		fmt.Printf("Config %d:\n", i+1)
		fmt.Printf("  Name: %s\n", config.Name)
		fmt.Printf("  Slug: %s\n", config.Slug)
		fmt.Printf("  Platform: %s\n", config.Platform)
		fmt.Printf("  Description: %s\n", config.Description)
		fmt.Printf("  Type: %s\n", config.Type)
		fmt.Printf("  Version: %s\n", config.Version)
		fmt.Println()
	}
}

func loadGitHubConfig() (*GitHubConfig, error) {
	agentRepo := os.Getenv("AGENT_REPO")
	if agentRepo == "" {
		return nil, fmt.Errorf("AGENT_REPO environment variable not set")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, fmt.Errorf(
			"GITHUB_TOKEN environment variable not set",
		)
	}

	// Branch is optional, defaults to empty (uses default branch)
	branch := os.Getenv("BRANCH")

	return &GitHubConfig{
		AgentRepo:   agentRepo,
		GitHubToken: githubToken,
		Branch:      branch,
	}, nil
}

func readConfigsFileToJson(
	agentRepo, githubToken string,
) ([]ConfigJson, error) {
	config, err := loadGitHubConfig()
	if err != nil {
		return nil, err
	}

	data, err := fetchFromGitHub(agentRepo, githubToken, config.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub: %w", err)
	}

	var configFile ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return convertToConfigJson(configFile.Configs), nil
}

func convertToConfigJson(yamlConfigs []ConfigYaml) []ConfigJson {
	configs := make([]ConfigJson, 0, len(yamlConfigs))
	for _, c := range yamlConfigs {
		configs = append(configs, ConfigJson{
			Name:        c.Name,
			Slug:        c.Slug,
			Platform:    c.Platform,
			Description: c.Description,
			Type:        c.Type,
			Version:     c.Version,
		})
	}
	return configs
}

// @todo update to fetch from tagged release
func fetchFromGitHub(repo, token, branch string) ([]byte, error) {
	client := NewGitHubClient(token)
	return client.FetchFile(repo, ".fleetControl/configs.yml", branch)
}

func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,
	}
}

func (gc *GitHubClient) FetchFile(
	repo, path, branch string,
) ([]byte, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/contents/%s",
		repo,
		path,
	)

	// Add branch query parameter if specified
	if branch != "" {
		url = fmt.Sprintf("%s?ref=%s", url, branch)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := gc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"GitHub API returned status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var content GitHubContent
	if err := json.Unmarshal(body, &content); err != nil {
		return nil, fmt.Errorf(
			"failed to parse GitHub response: %w",
			err,
		)
	}

	if content.Encoding != "base64" {
		return nil, fmt.Errorf(
			"unexpected encoding: %s",
			content.Encoding,
		)
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to decode base64 content: %w",
			err,
		)
	}

	return decoded, nil
}
