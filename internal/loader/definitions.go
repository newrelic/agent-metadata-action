package loader

import (
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/models"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ReadConfigurationDefinitions reads and parses the configurationDefinitions file
func ReadConfigurationDefinitions(workspacePath string) ([]models.ConfigurationDefinition, error) {
	fullPath := filepath.Join(workspacePath, config.GetConfigurationDefinitionsFilepath())

	definitions, err := readDefinitionsFile(fullPath)
	if err != nil {
		return nil, err
	}

	for i := range definitions {
		// Skip if no schema path is provided
		if definitions[i]["schema"] == nil || definitions[i]["schema"] == "" {
			fmt.Printf("::debug::no schema provided - skipping\n")
			continue
		}
		schemaPath, ok := definitions[i]["schema"].(string)
		if !ok {
			fmt.Printf("::warn::schema field is not a string - skipping\n")
			continue
		}

		// @todo at some point, we may want to do this concurrently if there are any agents with a large number of files
		encoded, err := loadAndEncodeFile(workspacePath, schemaPath, "schema")
		if err != nil {
			fmt.Printf("::warn::failed to load schema at schema path %s: %v -- continuing without it\n", schemaPath, err)
			continue
		}
		definitions[i]["schema"] = encoded
	}

	// Convert to []models.ConfigurationDefinition
	result := make([]models.ConfigurationDefinition, len(definitions))
	for i, def := range definitions {
		result[i] = models.ConfigurationDefinition(def)
	}

	return result, nil
}

// ReadAgentControlDefinitions reads and parses the agentControlDefinitions file
func ReadAgentControlDefinitions(workspacePath string) ([]models.AgentControlDefinition, error) {
	fullPath := filepath.Join(workspacePath, config.GetAgentControlDefinitionsFilepath())

	definitions, err := readDefinitionsFile(fullPath)
	if err != nil {
		return nil, err
	}

	// Load and encode content files
	for i := range definitions {
		// Skip if no content path is provided
		if definitions[i]["content"] == nil || definitions[i]["content"] == "" {
			fmt.Printf("::debug::no content provided - skipping\n")
			continue
		}
		contentPath, ok := definitions[i]["content"].(string)
		if !ok {
			fmt.Printf("::warn::content field is not a string - skipping\n")
			continue
		}

		// @todo at some point, we may want to do this concurrently if there are any agents with a large number of files
		encoded, err := loadAndEncodeFile(workspacePath, contentPath, "content")
		if err != nil {
			fmt.Printf("::warn::failed to load content at path %s: %v -- continuing without it\n", contentPath, err)
			continue
		}
		definitions[i]["content"] = encoded
	}

	// Convert to []models.AgentControlDefinition
	result := make([]models.AgentControlDefinition, len(definitions))
	for i, def := range definitions {
		result[i] = models.AgentControlDefinition(def)
	}

	return result, nil
}

// readDefinitionsFile reads a YAML file and extracts the first array it finds at the top level.
// This is a generic function that works for both configurationDefinitions and agentControlDefinitions files.
// It returns the array of definitions as []map[string]interface{}.
func readDefinitionsFile(fullPath string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file at %s: %w", fullPath, err)
	}

	// Unmarshal into a generic map to find the top-level array
	var fileContent map[string]interface{}
	if err := yaml.Unmarshal(data, &fileContent); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Find the first array in the top-level keys
	for key, value := range fileContent {
		if arr, ok := value.([]interface{}); ok {
			// Convert []interface{} to []map[string]interface{}
			definitions := make([]map[string]interface{}, 0, len(arr))
			for i, item := range arr {
				if def, ok := item.(map[string]interface{}); ok {
					definitions = append(definitions, def)
				} else {
					return nil, fmt.Errorf("item %d in %s is not a map", i, key)
				}
			}

			if len(definitions) == 0 {
				return nil, fmt.Errorf("%s cannot be empty", key)
			}

			return definitions, nil
		}
	}

	return nil, fmt.Errorf("no array found in YAML file")
}

// loadAndEncodeFile reads a file (schema, agent control, etc.) and returns its base64-encoded content.
// contentFieldName is the field in the definition map (e.g., "schema", "content") where the file path is found
func loadAndEncodeFile(workspacePath string, contentPath string, filePathField string) (string, error) {
	if contentPath == "" {
		return "", nil
	}

	// Validate content path to prevent directory traversal attacks
	if strings.Contains(contentPath, "..") {
		return "", fmt.Errorf("invalid %s path: contains directory traversal", filePathField)
	}

	// Content paths are relative to the expected root directory
	fullPath := filepath.Join(workspacePath, config.GetRootFolderForAgentRepo(), contentPath)

	// Additional security check: ensure the resolved path is within the expected root directory
	expectedRootDir := filepath.Join(workspacePath, config.GetRootFolderForAgentRepo())
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s path: %w", filePathField, err)
	}

	resolvedRootDirectory, err := filepath.Abs(expectedRootDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve expected root directory:%s %w", expectedRootDir, err)
	}

	if !strings.HasPrefix(resolvedPath, resolvedRootDirectory+string(filepath.Separator)) && resolvedPath != resolvedRootDirectory {
		return "", fmt.Errorf("invalid %s path: must be within expected root directory: %s\n", filePathField, expectedRootDir)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s file at %s: %w", filePathField, fullPath, err)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("%s file at %s is empty", filePathField, fullPath)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}
