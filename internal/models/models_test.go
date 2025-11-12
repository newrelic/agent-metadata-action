package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToConfigJson(t *testing.T) {
	configs := []ConfigurationDefinition{
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

	assert.NotNil(t, configs)
	assert.Equal(t, 2, len(configs))

	// Verify first config
	assert.Equal(t, "Test Config", configs[0].Name)
	assert.Equal(t, "test-config", configs[0].Slug)
	assert.Equal(t, "linux", configs[0].Platform)
	assert.Equal(t, "A test configuration", configs[0].Description)
	assert.Equal(t, "agent", configs[0].Type)
	assert.Equal(t, "1.0.0", configs[0].Version)

	// Verify second config
	assert.Equal(t, "Another Config", configs[1].Name)
	assert.Equal(t, "another-config", configs[1].Slug)
	assert.Equal(t, "windows", configs[1].Platform)
	assert.Equal(t, "Another test configuration", configs[1].Description)
	assert.Equal(t, "integration", configs[1].Type)
	assert.Equal(t, "2.0.0", configs[1].Version)
}
