package loader

import (
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/models"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

const AgentControlPlatform = "ALL"

// ReadAgentControlDefinitions reads YAML files from agentControl folder
func ReadAgentControlDefinitions(workspacePath string) ([]models.AgentControlDefinition, error) {
	fullPath := filepath.Join(workspacePath, config.GetAgentControlFolderForAgentRepo())

	// Get all YAML files (.yml and .yaml)
	allFiles, err := filepath.Glob(filepath.Join(fullPath, "*.y*ml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob YAML files at %s: %w", fullPath, err)
	}

	var agentControlDefinitions = make([]models.AgentControlDefinition, 0, len(allFiles))
	for _, filePath := range allFiles {
		fileName := filepath.Base(filePath)
		encoded, err := loadAndEncodeAgentControl(workspacePath, fileName)
		if err != nil {
			fmt.Printf("::warn::failed to load agent control file %s: %v -- continuing without it\n", fileName, err)
			continue
		}

		agentControlDefinitions = append(agentControlDefinitions, encoded)
	}
	return agentControlDefinitions, nil
}

// LoadAndEncodeAgentControl reads and encodes the agent control content
// Returns a single entry with platform AgentControlPlatform
func loadAndEncodeAgentControl(workspacePath string, filename string) (models.AgentControlDefinition, error) {
	agentControlPath := filepath.Join(workspacePath, config.GetAgentControlFolderForAgentRepo(), filename)

	data, err := os.ReadFile(agentControlPath)
	if err != nil {
		return models.AgentControlDefinition{}, fmt.Errorf("failed to read agent control file at %s: %w", agentControlPath, err)
	}

	if len(data) == 0 {
		return models.AgentControlDefinition{}, fmt.Errorf("agent control file at %s is empty", agentControlPath)
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	return models.AgentControlDefinition{
		Platform: AgentControlPlatform,
		Content:  encoded,
	}, nil
}
