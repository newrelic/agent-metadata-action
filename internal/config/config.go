package config

import (
	"fmt"
	"os"
	"path/filepath"

	"agent-metadata-action/internal/models"

	"gopkg.in/yaml.v3"
)

const CONFIG_FILE_PATH = ".fleetControl/configurationDefinitions.yml"

// LoadEnv loads the workspace path from environment variables
func LoadEnv() (string, error) {
	workspace := os.Getenv("GITHUB_WORKSPACE")
	if workspace == "" {
		return "", fmt.Errorf("GITHUB_WORKSPACE environment variable not set")
	}
	return workspace, nil
}

// ReadConfigurationDefinitions reads and parses the configurationDefinitions file
func ReadConfigurationDefinitions(workspacePath string) ([]models.ConfigurationDefinition, error) {
	fullPath := filepath.Join(workspacePath, CONFIG_FILE_PATH)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", fullPath, err)
	}

	var configFile models.ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return configFile.Configs, nil
}
