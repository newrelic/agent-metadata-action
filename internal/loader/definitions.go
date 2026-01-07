package loader

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent-metadata-action/internal/models"

	"gopkg.in/yaml.v3"
)

const FleetControlDir = ".fleetControl"
const ConfigFilePath = "configurationDefinitions.yml"
const AgentControlDir = "agentControl"
const AgentControlFile = "agent-schema-for-agent-control.yml"
const AgentControlPlatform = "ALL"

// ReadConfigurationDefinitions reads and parses the configurationDefinitions file
func ReadConfigurationDefinitions(workspacePath string) ([]models.ConfigurationDefinition, error) {
	fullPath := filepath.Join(workspacePath, FleetControlDir, ConfigFilePath)

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

	// Load and encode schema files, if provided
	for i := range configFile.Configs {
		// Get schema path from map
		schemaPath, _ := configFile.Configs[i]["schema"].(string)

		// Skip if no schema path is provided
		if schemaPath == "" {
			fmt.Println("::warn::no schema path provided")
			continue
		}

		// @todo at some point, we may want to do this concurrently if there are any agents with a large number of files
		encoded, err := loadAndEncodeSchema(workspacePath, schemaPath)
		if err != nil {
			fmt.Printf("::warn::failed to load schema at schema path %s: %v -- continuing without it\n", schemaPath, err)
			continue
		}

		// Update schema with encoded content
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

	// Schema paths are relative to the .fleetControl directory
	fullPath := filepath.Join(workspacePath, FleetControlDir, schemaPath)

	// Additional security check: ensure the resolved path is within .fleetControl
	fleetCtrlDir := filepath.Join(workspacePath, FleetControlDir)
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve schema path: %w", err)
	}

	resolvedFleetControl, err := filepath.Abs(fleetCtrlDir)
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

	if len(data) == 0 {
		return "", fmt.Errorf("schema file at %s is empty", fullPath)
	}

	if !json.Valid(data) {
		return "", fmt.Errorf("schema file at %s is not valid JSON", fullPath)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}

// LoadAndEncodeAgentControl reads and encodes the agent control content
// Returns a single entry with platform AgentControlPlatform
func LoadAndEncodeAgentControl(workspacePath string) ([]models.AgentControl, error) {
	return LoadAndEncodeAgentControlFunc(workspacePath)
}

// LoadAndEncodeAgentControlFunc is a variable that holds the function to load and encode agent control
// This allows tests to override the implementation
var LoadAndEncodeAgentControlFunc = loadAndEncodeAgentControlImpl

// loadAndEncodeAgentControlImpl is the actual implementation
func loadAndEncodeAgentControlImpl(workspacePath string) ([]models.AgentControl, error) {
	agentControlPath := filepath.Join(workspacePath, FleetControlDir, AgentControlDir, AgentControlFile)

	data, err := os.ReadFile(agentControlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent control file at  %s: %w", agentControlPath, err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("agent control file at %s is empty\n", agentControlPath)
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	return []models.AgentControl{
		{
			Platform: AgentControlPlatform,
			Content:  encoded,
		},
	}, nil
}
