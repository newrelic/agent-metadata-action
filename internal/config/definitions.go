package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent-metadata-action/internal/models"

	"gopkg.in/yaml.v3"
)

const FLEET_CONTROL_DIR = ".fleetControl"
const CONFIG_FILE_PATH = "configurationDefinitions.yml"
const AGENT_CONTROL_DIR = "agentControl"
const AGENT_CONTROL_FILE = "agent-schema-for-agent-control.yml"

// ReadConfigurationDefinitions reads and parses the configurationDefinitions file
func ReadConfigurationDefinitions(workspacePath string) ([]models.ConfigurationDefinition, error) {
	fullPath := filepath.Join(workspacePath, FLEET_CONTROL_DIR, CONFIG_FILE_PATH)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", fullPath, err)
	}

	var configFile models.ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate that configurationDefinitions is not empty
	if len(configFile.Configs) == 0 {
		return nil, fmt.Errorf("configurationDefinitions cannot be empty")
	}

	// Load and encode schema files (schema is required by validation)
	for i := range configFile.Configs {
		// @todo at some point, we may want to do this concurrently if there are any agents with a large number of files
		encoded, err := loadAndEncodeSchema(workspacePath, configFile.Configs[i].Schema)
		if err != nil {
			return nil, fmt.Errorf("failed to load schema for config %s and version %s: %w", configFile.Configs[i].Type, configFile.Configs[i].Version, err)
		}
		configFile.Configs[i].Schema = encoded
	}

	return configFile.Configs, nil
}

// loadAndEncodeSchema reads a schema file and returns its base64-encoded content
func loadAndEncodeSchema(workspacePath, schemaPath string) (string, error) {
	if schemaPath == "" {
		return "", nil
	}

	// Validate schema path to prevent directory traversal attacks
	if strings.Contains(schemaPath, "..") {
		return "", fmt.Errorf("invalid schema path: contains directory traversal")
	}

	// Schema paths are relative to the .fleetControl directory
	fullPath := filepath.Join(workspacePath, FLEET_CONTROL_DIR, schemaPath)

	// Additional security check: ensure the resolved path is within .fleetControl
	fleetControlDir := filepath.Join(workspacePath, FLEET_CONTROL_DIR)
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve schema path: %w", err)
	}

	resolvedFleetControl, err := filepath.Abs(fleetControlDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve fleet control directory: %w", err)
	}

	if !strings.HasPrefix(resolvedPath, resolvedFleetControl+string(filepath.Separator)) && resolvedPath != resolvedFleetControl {
		return "", fmt.Errorf("invalid schema path: must be within .fleetControl directory")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file at %s: %w", fullPath, err)
	}

	// Fail if schema file is empty
	if len(data) == 0 {
		return "", fmt.Errorf("schema file at %s is empty", fullPath)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}

// LoadAndEncodeAgentControl reads and encodes the agent control content
// Returns a single entry with platform "all"
func LoadAndEncodeAgentControl(workspacePath string) ([]models.AgentControl, error) {
	agentControlPath := filepath.Join(workspacePath, FLEET_CONTROL_DIR, AGENT_CONTROL_DIR, AGENT_CONTROL_FILE)

	data, err := os.ReadFile(agentControlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent control file at %s: %w", agentControlPath, err)
	}

	// Fail if agent control file is empty
	if len(data) == 0 {
		return nil, fmt.Errorf("agent control file at %s is empty", agentControlPath)
	}

	// Encode content as base64
	encoded := base64.StdEncoding.EncodeToString(data)

	return []models.AgentControl{
		{
			Platform: "all",
			Content:  encoded,
		},
	}, nil
}
