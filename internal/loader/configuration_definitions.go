package loader

import (
	"agent-metadata-action/internal/config"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent-metadata-action/internal/models"

	"gopkg.in/yaml.v3"
)

const AGENT_CONTROL_FILE = "agent-schema-for-agent-control.yml" // @todo move out of this file
const AGENT_CONTROL_PLATFORM = "ALL"                            // @todo move out of this file

// ReadConfigurationDefinitions reads and parses the configurationDefinitions file
func ReadConfigurationDefinitions(workspacePath string) ([]models.ConfigurationDefinition, error) {
	fullPath := filepath.Join(workspacePath, config.GetConfigurationDefinitionsFilepath())

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", fullPath, err)
	}

	var configFile models.ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if len(configFile.Configs) == 0 {
		return nil, fmt.Errorf("configurationDefinitions cannot be empty")
	}

	// Load and encode schema files (schema is optional for now but will be required in the future)
	for i := range configFile.Configs {
		// Skip if no schema path is provided
		if configFile.Configs[i]["schema"] == nil || configFile.Configs[i]["schema"] == "" {
			fmt.Printf("::debug::no schema provided - skipping\n")
			continue
		}
		schemaPath := configFile.Configs[i]["schema"].(string)

		// @todo at some point, we may want to do this concurrently if there are any agents with a large number of files
		encoded, err := loadAndEncodeSchema(workspacePath, schemaPath)
		if err != nil {
			fmt.Printf("::warn::failed to load schema at schema path %s: %v -- continuing without it\n", schemaPath, err)
			continue
		}
		configFile.Configs[i]["schema"] = encoded
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

	// Schema paths are relative to the expected root directory
	fullPath := filepath.Join(workspacePath, config.GetRootFolderForAgentRepo(), schemaPath)

	// Additional security check: ensure the resolved path is within the expected root directory
	expectedRootDir := filepath.Join(workspacePath, config.GetRootFolderForAgentRepo())
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve schema path: %w", err)
	}

	resolvedRootDirectory, err := filepath.Abs(expectedRootDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve expected root directory:%s %w", expectedRootDir, err)
	}

	if !strings.HasPrefix(resolvedPath, resolvedRootDirectory+string(filepath.Separator)) && resolvedPath != resolvedRootDirectory {
		return "", fmt.Errorf("invalid schema path: must be within expected root directory: %s\n", expectedRootDir)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file at %s: %w", fullPath, err)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("schema file at %s is empty", fullPath)
	}

	if !json.Valid(data) {
		return "", fmt.Errorf("schema file at %s is not valid JSON", fullPath)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}

// @todo break this out into a different file
// LoadAndEncodeAgentControl reads and encodes the agent control content
// Returns a single entry with platform AGENT_CONTROL_PLATFORM
func LoadAndEncodeAgentControl(workspacePath string) ([]models.AgentControlDefinition, error) {
	agentControlPath := filepath.Join(workspacePath, config.GetAgentControlFolderForAgentRepo(), AGENT_CONTROL_FILE)

	data, err := os.ReadFile(agentControlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent control file at %s: %w", agentControlPath, err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("agent control file at %s is empty", agentControlPath)
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	return []models.AgentControlDefinition{
		{
			Platform: AGENT_CONTROL_PLATFORM,
			Content:  encoded,
		},
	}, nil
}
