package loader

import (
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/validator"
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

		resolvedPath, err := resolveContentPath(workspacePath, schemaPath)
		if err != nil {
			logging.Warnf(ctx, "failed to resolve schema path %s: %v -- dropping schema field", schemaPath, err)
			delete(definitions[i], "schema")
			continue
		}

		// @todo at some point, we may want to do this concurrently if there are any agents with a large number of files
		encoded, err := loadAndEncodeFile(resolvedPath)
		if err != nil {
			// Drop the field rather than leaving the path string in place — the server would
			// otherwise try to base64-decode the path and reject the whole bundled request.
			logging.Warnf(ctx, "failed to read schema file at %s: %v -- dropping schema field", resolvedPath, err)
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

		resolvedPath, err := resolveContentPath(workspacePath, contentPath)
		if err != nil {
			logging.Warnf(ctx, "failed to resolve content path %s: %v -- dropping content field", contentPath, err)
			delete(definitions[i], "content")
			continue
		}

		if config.GetValidateAgentType() {
			if err := validator.ValidateAgentTypeDefinitionFunc(ctx, resolvedPath); err != nil {
				return nil, fmt.Errorf("agent type definition validation failed for %s: %w", contentPath, err)
			}
			logging.Noticef(ctx, "Agent type definition at %s passed validation", contentPath)
		}

		// @todo at some point, we may want to do this concurrently if there are any agents with a large number of files
		encoded, err := loadAndEncodeFile(resolvedPath)
		if err != nil {
			// Drop the field rather than leaving the path string in place — the server would
			// otherwise try to base64-decode the path and reject the whole bundled request.
			logging.Warnf(ctx, "failed to read content file at %s: %v -- dropping content field", resolvedPath, err)
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

// ReadAgentDefinition reads the optional agentDefinition.yml file.
// Returns nil, nil if the file does not exist (the file is optional).
func ReadAgentDefinition(ctx context.Context, workspacePath string) (*models.AgentDefinition, error) {
	fullPath := filepath.Join(workspacePath, config.GetAgentDefinitionFilepath())

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			logging.Debug(ctx, "agentDefinition.yml not found - skipping")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read agentDefinition.yml: %w", err)
	}

	var def models.AgentDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse agentDefinition.yml: %w", err)
	}
	return &def, nil
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

// resolveContentPath resolves contentPath (relative to the .fleetControl directory) to an
// absolute path and validates that it stays within workspacePath, preventing directory
// traversal outside the workspace.
func resolveContentPath(workspacePath string, contentPath string) (string, error) {
	fullPath := filepath.Join(workspacePath, config.GetRootFolderForAgentRepo(), contentPath)

	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	resolvedWorkspace, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path %s: %w", workspacePath, err)
	}

	if !strings.HasPrefix(resolvedPath, resolvedWorkspace+string(filepath.Separator)) && resolvedPath != resolvedWorkspace {
		return "", fmt.Errorf("must be within workspace: %s", resolvedWorkspace)
	}

	return resolvedPath, nil
}

// loadAndEncodeFile reads the file at resolvedPath and returns its base64-encoded content.
func loadAndEncodeFile(resolvedPath string) (string, error) {
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file at %s: %w", resolvedPath, err)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("file at %s is empty", resolvedPath)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}
