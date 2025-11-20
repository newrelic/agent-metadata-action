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
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, "kubernetes", config.Platform)
	assert.Equal(t, "A test configuration", config.Description)
	assert.Equal(t, "test-type", config.Type)
	assert.Equal(t, "json", config.Format)
	assert.Equal(t, "./schema.json", config.Schema)
}

func TestConfigurationDefinition_UnmarshalYAML_MissingFields(t *testing.T) {
	tests := []struct {
		name          string
		yamlData      string
		expectedError string
	}{
		{
			name: "missing version",
			yamlData: `
platform: kubernetes
description: A test configuration
type: test-type
format: json
schema: ./schema.json
`,
			expectedError: "version is required",
		},
		{
			name: "missing platform",
			yamlData: `
version: 1.0.0
description: A test configuration
type: test-type
format: json
schema: ./schema.json
`,
			expectedError: "platform is required",
		},
		{
			name: "missing description",
			yamlData: `
version: 1.0.0
platform: kubernetes
type: test-type
format: json
schema: ./schema.json
`,
			expectedError: "description is required",
		},
		{
			name: "missing type",
			yamlData: `
version: 1.0.0
platform: kubernetes
description: A test configuration
format: json
schema: ./schema.json
`,
			expectedError: "type is required",
		},
		{
			name: "missing format",
			yamlData: `
version: 1.0.0
platform: kubernetes
description: A test configuration
type: test-type
schema: ./schema.json
`,
			expectedError: "format is required",
		},
		{
			name: "missing schema",
			yamlData: `
version: 1.0.0
platform: kubernetes
description: A test configuration
type: test-type
format: json
`,
			expectedError: "schema is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config ConfigurationDefinition
			err := yaml.Unmarshal([]byte(tt.yamlData), &config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestConfigurationDefinition_UnmarshalYAML_ErrorContext(t *testing.T) {
	// When type and version are provided, error messages should include context
	yamlData := `
version: 1.0.0
description: A test configuration
type: mytype
format: json
schema: ./schema.json
`
	var config ConfigurationDefinition
	err := yaml.Unmarshal([]byte(yamlData), &config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform is required for config with type 'mytype' and version '1.0.0'")
}

func TestRequireField_ValidValue(t *testing.T) {
	err := requireField("some-value", "fieldName", "")
	assert.NoError(t, err)
}

func TestRequireField_EmptyValueWithoutContext(t *testing.T) {
	err := requireField("", "fieldName", "")
	assert.Error(t, err)
	assert.Equal(t, "fieldName is required", err.Error())
}

func TestRequireField_EmptyValueWithContext(t *testing.T) {
	err := requireField("", "fieldName", "myContext")
	assert.Error(t, err)
	assert.Equal(t, "fieldName is required for myContext", err.Error())
}

func TestAgentMetadata_JSONMarshaling(t *testing.T) {
	// Test that AgentMetadata can be marshaled to JSON
	agentMetadata := AgentMetadata{
		ConfigurationDefinitions: []ConfigurationDefinition{
			{
				Version:     "1.0.0",
				Platform:    "k8s",
				Description: "Test config",
				Type:        "test",
				Format:      "json",
				Schema:      "encoded",
			},
		},
		Metadata: Metadata{
			Version:  "1.2.3",
			Features: []string{"feature1"},
			Bugs:     []string{"bug1"},
			Security: []string{"CVE-1"},
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
