package config

import (
	"fmt"
	"os"

	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/models"

	"gopkg.in/yaml.v3"
)

const CONFIG_FILE_PATH = ".fleetControl/configs.yml"

// Config represents the GitHub configuration
type Config struct {
	AgentRepo   string
	GitHubToken string
	Branch      string
}

// LoadEnv loads environment variables
func LoadEnv() (*Config, error) {
	agentRepo := os.Getenv("AGENT_REPO")
	if agentRepo == "" {
		return nil, fmt.Errorf("AGENT_REPO environment variable not set")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable not set")
	}

	// Branch is optional, defaults to empty (uses default branch)
	branch := os.Getenv("BRANCH")

	return &Config{
		AgentRepo:   agentRepo,
		GitHubToken: githubToken,
		Branch:      branch,
	}, nil
}

// ReadConfigs reads and parses the configs file from GitHub
func ReadConfigs(cfg *Config) ([]models.ConfigJson, error) {
	client := github.GetClient(cfg.GitHubToken)

	data, err := client.FetchFile(cfg.AgentRepo, CONFIG_FILE_PATH, cfg.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub: %w", err)
	}

	var configFile models.ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return models.ConvertToConfigJson(configFile.Configs), nil
}
