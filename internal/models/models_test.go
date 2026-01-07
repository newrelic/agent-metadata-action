package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConfigurationDefinition_UnmarshalYAML_Success(t *testing.T) {
	yamlData := `
version: 1.0.0
platform: kubernetes
description: A test configuration
type: test-type
format: json
schema: ./schema.json
`
	var config ConfigurationDefinition
	err := yaml.Unmarshal([]byte(yamlData), &config)

	require.NoError(t, err)
	assert.Equal(t, "1.0.0", config["version"])
	assert.Equal(t, "kubernetes", config["platform"])
	assert.Equal(t, "A test configuration", config["description"])
	assert.Equal(t, "test-type", config["type"])
	assert.Equal(t, "json", config["format"])
	assert.Equal(t, "./schema.json", config["schema"])
}

func TestConfigurationDefinition_UnmarshalYAML_AdditionalFields(t *testing.T) {
	// Test that additional fields are captured and passed through
	yamlData := `
version: 1.0.0
platform: kubernetes
description: A test configuration
type: test-type
format: json
schema: ./schema.json
newField: newValue
anotherField: 12345
`
	var config ConfigurationDefinition
	err := yaml.Unmarshal([]byte(yamlData), &config)

	require.NoError(t, err)
	assert.Equal(t, "1.0.0", config["version"])
	assert.Equal(t, "kubernetes", config["platform"])
	assert.Equal(t, "A test configuration", config["description"])
	assert.Equal(t, "test-type", config["type"])
	assert.Equal(t, "json", config["format"])
	assert.Equal(t, "./schema.json", config["schema"])
	assert.Equal(t, "newValue", config["newField"])
	assert.Equal(t, 12345, config["anotherField"])
}

func TestConfigurationDefinition_UnmarshalYAML_MissingFields(t *testing.T) {
	// Test that missing fields don't cause errors (no validation)
	yamlData := `
version: 1.0.0
platform: kubernetes
`
	var config ConfigurationDefinition
	err := yaml.Unmarshal([]byte(yamlData), &config)

	require.NoError(t, err)
	assert.Equal(t, "1.0.0", config["version"])
	assert.Equal(t, "kubernetes", config["platform"])
	assert.Nil(t, config["description"])
	assert.Nil(t, config["type"])
}

func TestAgentMetadata_JSONMarshaling(t *testing.T) {
	// Test that AgentMetadata can be marshaled to JSON
	agentMetadata := AgentMetadata{
		ConfigurationDefinitions: []ConfigurationDefinition{
			{
				"version":     "1.0.0",
				"platform":    "k8s",
				"description": "Test config",
				"type":        "test",
				"format":      "json",
				"schema":      "encoded",
			},
		},
		Metadata: Metadata{
			"version":  "1.2.3",
			"features": []string{"feature1"},
			"bugs":     []string{"bug1"},
			"security": []string{"CVE-1"},
		},
		AgentControl: []AgentControl{
			{
				Platform: "all",
				Content:  "base64content",
			},
		},
	}

	jsonData, err := json.Marshal(agentMetadata)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "1.0.0")
	assert.Contains(t, string(jsonData), "1.2.3")
	assert.Contains(t, string(jsonData), "base64content")
}
