package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToConfigJson(t *testing.T) {
	yamlConfigs := []ConfigYaml{
		{
			Name:        "Test Config",
			Slug:        "test-config",
			Platform:    "linux",
			Description: "A test configuration",
			Type:        "agent",
			Version:     "1.0.0",
			Schema:      "v1",
		},
		{
			Name:        "Another Config",
			Slug:        "another-config",
			Platform:    "windows",
			Description: "Another test configuration",
			Type:        "integration",
			Version:     "2.0.0",
			Schema:      "v2",
		},
	}

	jsonConfigs := ConvertToConfigJson(yamlConfigs)

	assert.NotNil(t, jsonConfigs)
	assert.Equal(t, 2, len(jsonConfigs))

	// Verify first config
	assert.Equal(t, "Test Config", jsonConfigs[0].Name)
	assert.Equal(t, "test-config", jsonConfigs[0].Slug)
	assert.Equal(t, "linux", jsonConfigs[0].Platform)
	assert.Equal(t, "A test configuration", jsonConfigs[0].Description)
	assert.Equal(t, "agent", jsonConfigs[0].Type)
	assert.Equal(t, "1.0.0", jsonConfigs[0].Version)

	// Verify second config
	assert.Equal(t, "Another Config", jsonConfigs[1].Name)
	assert.Equal(t, "another-config", jsonConfigs[1].Slug)
	assert.Equal(t, "windows", jsonConfigs[1].Platform)
	assert.Equal(t, "Another test configuration", jsonConfigs[1].Description)
	assert.Equal(t, "integration", jsonConfigs[1].Type)
	assert.Equal(t, "2.0.0", jsonConfigs[1].Version)
}

func TestConvertToConfigJsonEmptyArray(t *testing.T) {
	yamlConfigs := []ConfigYaml{}
	jsonConfigs := ConvertToConfigJson(yamlConfigs)

	assert.NotNil(t, jsonConfigs)
	assert.Equal(t, 0, len(jsonConfigs))
}
