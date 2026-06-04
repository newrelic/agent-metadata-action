package loader

import (
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ReadConfigurationDefinitions reads and parses the configurationDefinitions file
func ReadConfigurationDefinitions(ctx context.Context, workspacePath string) ([]models.ConfigurationDefinition, error) {
	fullPath := filepath.Join(workspacePath, config.GetConfigurationDefinitionsFilepath())

	definitions, err := readDefinitionsFile(fullPath)
	if err != nil {
		return nil, err
	}

	for i := range definitions {
		// Skip if no schema path is provided
		if definitions[i]["schema"] == nil || definitions[i]["schema"] == "" {
			logging.Debug(ctx, "no schema provided - skipping")
			continue
		}
		schemaPath, ok := definitions[i]["schema"].(string)
		if !ok {
			// Drop the field so the server doesn't reject the whole request over a malformed type.
			logging.Warn(ctx, "schema field is not a string - dropping it")
			delete(definitions[i], "schema")
			continue
		}

		// @todo at some point, we may want to do this concurrently if there are any agents with a large number of files
		encoded, err := loadAndEncodeFile(workspacePath, schemaPath, "schema")
		if err != nil {
			// Drop the field rather than leaving the path string in place — the server would
			// otherwise try to base64-decode the path and reject the whole bundled request.
			logging.Warnf(ctx, "failed to load schema at schema path %s: %v -- dropping schema field", schemaPath, err)
			delete(definitions[i], "schema")
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
func ReadAgentControlDefinitions(ctx context.Context, workspacePath string) ([]models.AgentControlDefinition, error) {
	fullPath := filepath.Join(workspacePath, config.GetAgentControlDefinitionsFilepath())

	definitions, err := readDefinitionsFile(fullPath)
	if err != nil {
		return nil, err
	}

	// Load and encode content files
	for i := range definitions {
		// Skip if no content path is provided
		if definitions[i]["content"] == nil || definitions[i]["content"] == "" {
			logging.Debug(ctx, "no content provided - skipping")
			continue
		}
		contentPath, ok := definitions[i]["content"].(string)
		if !ok {
			// Drop the field so the server doesn't reject the whole request over a malformed type.
			logging.Warn(ctx, "content field is not a string - dropping it")
			delete(definitions[i], "content")
			continue
		}

		// @todo at some point, we may want to do this concurrently if there are any agents with a large number of files
		encoded, err := loadAndEncodeFile(workspacePath, contentPath, "content")
		if err != nil {
			// Drop the field rather than leaving the path string in place — the server would
			// otherwise try to base64-decode the path and reject the whole bundled request.
			logging.Warnf(ctx, "failed to load content at path %s: %v -- dropping content field", contentPath, err)
			delete(definitions[i], "content")
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

	// Content paths are relative to the .fleetControl directory; the resolved path
	// must stay within the workspace so we can't read arbitrary files on the runner.
	fullPath := filepath.Join(workspacePath, config.GetRootFolderForAgentRepo(), contentPath)

	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s path: %w", filePathField, err)
	}

	resolvedWorkspace, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path %s: %w", workspacePath, err)
	}

	if !strings.HasPrefix(resolvedPath, resolvedWorkspace+string(filepath.Separator)) && resolvedPath != resolvedWorkspace {
		return "", fmt.Errorf("invalid %s path: must be within workspace: %s", filePathField, resolvedWorkspace)
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
