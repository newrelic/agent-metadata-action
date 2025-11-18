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
slug: test-config
name: Test Configuration
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
	assert.Equal(t, "test-config", config.Slug)
	assert.Equal(t, "Test Configuration", config.Name)
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
			name: "missing slug",
			yamlData: `
name: Test Configuration
version: 1.0.0
platform: kubernetes
description: A test configuration
type: test-type
format: json
schema: ./schema.json
`,
			expectedError: "slug is required",
		},
		{
			name: "missing name",
			yamlData: `
slug: test-config
version: 1.0.0
platform: kubernetes
description: A test configuration
type: test-type
format: json
schema: ./schema.json
`,
			expectedError: "name is required",
		},
		{
			name: "missing version",
			yamlData: `
slug: test-config
name: Test Configuration
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
slug: test-config
name: Test Configuration
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
slug: test-config
name: Test Configuration
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
slug: test-config
name: Test Configuration
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
slug: test-config
name: Test Configuration
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
slug: test-config
name: Test Configuration
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
	// When name is provided, error messages should include the config name
	yamlData := `
slug: test-config
name: MyConfig
version: 1.0.0
description: A test configuration
type: test-type
format: json
schema: ./schema.json
`
	var config ConfigurationDefinition
	err := yaml.Unmarshal([]byte(yamlData), &config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform is required for config 'MyConfig'")
}

func TestMetadata_UnmarshalYAML_Success(t *testing.T) {
	yamlData := `
version: 1.2.3
features:
  - Feature 1
  - Feature 2
bugs:
  - Bug fix 1
security:
  - CVE-2024-1234
`
	var metadata Metadata
	err := yaml.Unmarshal([]byte(yamlData), &metadata)

	require.NoError(t, err)
	assert.Equal(t, "1.2.3", metadata.Version)
	assert.Equal(t, []string{"Feature 1", "Feature 2"}, metadata.Features)
	assert.Equal(t, []string{"Bug fix 1"}, metadata.Bugs)
	assert.Equal(t, []string{"CVE-2024-1234"}, metadata.Security)
}

func TestMetadata_UnmarshalYAML_MinimalFields(t *testing.T) {
	yamlData := `
version: 1.0.0
`
	var metadata Metadata
	err := yaml.Unmarshal([]byte(yamlData), &metadata)

	require.NoError(t, err)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Nil(t, metadata.Features)
	assert.Nil(t, metadata.Bugs)
	assert.Nil(t, metadata.Security)
}

func TestMetadata_UnmarshalYAML_MissingVersion(t *testing.T) {
	yamlData := `
features:
  - Feature 1
bugs:
  - Bug fix 1
`
	var metadata Metadata
	err := yaml.Unmarshal([]byte(yamlData), &metadata)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestAgentControl_UnmarshalJSON_Success(t *testing.T) {
	jsonData := `{
		"platform": "kubernetes",
		"content": "base64encodedcontent"
	}`

	var agentControl AgentControl
	err := json.Unmarshal([]byte(jsonData), &agentControl)

	require.NoError(t, err)
	assert.Equal(t, "kubernetes", agentControl.Platform)
	assert.Equal(t, "base64encodedcontent", agentControl.Content)
}

func TestAgentControl_UnmarshalJSON_MissingPlatform(t *testing.T) {
	jsonData := `{
		"content": "base64encodedcontent"
	}`

	var agentControl AgentControl
	err := json.Unmarshal([]byte(jsonData), &agentControl)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform is required for agentControl")
}

func TestAgentControl_UnmarshalJSON_MissingContent(t *testing.T) {
	jsonData := `{
		"platform": "kubernetes"
	}`

	var agentControl AgentControl
	err := json.Unmarshal([]byte(jsonData), &agentControl)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content is required for agentControl")
}

func TestAgentControl_UnmarshalJSON_InvalidJSON(t *testing.T) {
	jsonData := `{invalid json`

	var agentControl AgentControl
	err := json.Unmarshal([]byte(jsonData), &agentControl)

	assert.Error(t, err)
}

func TestConfigFile_UnmarshalYAML_Success(t *testing.T) {
	yamlData := `
configurationDefinitions:
  - slug: config-1
    name: Config 1
    version: 1.0.0
    platform: kubernetes
    description: First config
    type: type-1
    format: json
    schema: ./schema1.json
  - slug: config-2
    name: Config 2
    version: 2.0.0
    platform: host
    description: Second config
    type: type-2
    format: yaml
    schema: ./schema2.json
`

	var configFile ConfigFile
	err := yaml.Unmarshal([]byte(yamlData), &configFile)

	require.NoError(t, err)
	assert.Len(t, configFile.Configs, 2)
	assert.Equal(t, "config-1", configFile.Configs[0].Slug)
	assert.Equal(t, "Config 1", configFile.Configs[0].Name)
	assert.Equal(t, "config-2", configFile.Configs[1].Slug)
	assert.Equal(t, "Config 2", configFile.Configs[1].Name)
}

func TestConfigFile_UnmarshalYAML_EmptyConfigs(t *testing.T) {
	yamlData := `
configurationDefinitions: []
`

	var configFile ConfigFile
	err := yaml.Unmarshal([]byte(yamlData), &configFile)

	require.NoError(t, err)
	assert.Len(t, configFile.Configs, 0)
}

func TestConfigFile_UnmarshalYAML_ValidationFailure(t *testing.T) {
	yamlData := `
configurationDefinitions:
  - slug: config-1
    name: Config 1
    version: 1.0.0
    platform: kubernetes
    description: First config
    type: type-1
    format: json
    # missing schema field
`

	var configFile ConfigFile
	err := yaml.Unmarshal([]byte(yamlData), &configFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema is required")
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
				Slug:        "test-config",
				Name:        "Test",
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
	assert.Contains(t, string(jsonData), "test-config")
	assert.Contains(t, string(jsonData), "1.2.3")
	assert.Contains(t, string(jsonData), "base64content")

	// Test unmarshaling back
	var unmarshaled AgentMetadata
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, "test-config", unmarshaled.ConfigurationDefinitions[0].Slug)
	assert.Equal(t, "1.2.3", unmarshaled.Metadata.Version)
	assert.Equal(t, "all", unmarshaled.AgentControl[0].Platform)
}
