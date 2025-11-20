package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadVersion_ValidFormats(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "standard version",
			version: "1.2.3",
		},
		{
			name:    "simple version",
			version: "1.0.0",
		},
		{
			name:    "large numbers",
			version: "100.200.300",
		},
		{
			name:    "zero version",
			version: "0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("INPUT_VERSION", tt.version)

			version, err := LoadVersion()
			require.NoError(t, err)
			assert.Equal(t, tt.version, version)
		})
	}
}

func TestLoadVersion_InvalidFormats(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "with v prefix",
			version: "v1.2.3",
		},
		{
			name:    "with prerelease",
			version: "1.2.3-alpha",
		},
		{
			name:    "with build metadata",
			version: "1.2.3+build",
		},
		{
			name:    "two components",
			version: "1.2",
		},
		{
			name:    "four components",
			version: "1.2.3.4",
		},
		{
			name:    "leading zero",
			version: "01.2.3",
		},
		{
			name:    "non-numeric",
			version: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("INPUT_VERSION", tt.version)

			_, err := LoadVersion()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid version format")
		})
	}
}

func TestLoadVersion_NotSet_Error(t *testing.T) {
	t.Setenv("INPUT_VERSION", "")

	_, err := LoadVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to determine version")
	assert.Contains(t, err.Error(), "INPUT_VERSION not set")
}

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single item",
			input:    "feature1",
			expected: []string{"feature1"},
		},
		{
			name:     "multiple items",
			input:    "feature1,feature2,feature3",
			expected: []string{"feature1", "feature2", "feature3"},
		},
		{
			name:     "with spaces",
			input:    "feature1, feature2 , feature3",
			expected: []string{"feature1", "feature2", "feature3"},
		},
		{
			name:     "with empty elements",
			input:    "feature1,,feature2,",
			expected: []string{"feature1", "feature2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadMetadata_AllInputs(t *testing.T) {
	t.Setenv("INPUT_VERSION", "1.2.3")
	t.Setenv("INPUT_FEATURES", "feature1,feature2")
	t.Setenv("INPUT_BUGS", "bug1,bug2,bug3")
	t.Setenv("INPUT_SECURITY", "CVE-2024-1234")
	t.Setenv("INPUT_DEPRECATIONS", "deprecated1,deprecated2")
	t.Setenv("INPUT_SUPPORTEDOPERATINGSYSTEMS", "linux,windows,darwin")
	t.Setenv("INPUT_EOL", "2025-12-31")

	metadata, err := LoadMetadata()
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", metadata.Version)
	assert.Equal(t, []string{"feature1", "feature2"}, metadata.Features)
	assert.Equal(t, []string{"bug1", "bug2", "bug3"}, metadata.Bugs)
	assert.Equal(t, []string{"CVE-2024-1234"}, metadata.Security)
	assert.Equal(t, []string{"deprecated1", "deprecated2"}, metadata.Deprecations)
	assert.Equal(t, []string{"linux", "windows", "darwin"}, metadata.SupportedOperatingSystems)
	assert.Equal(t, "2025-12-31", metadata.EOL)
}

func TestLoadMetadata_VersionOnly(t *testing.T) {
	t.Setenv("INPUT_VERSION", "2.0.0")
	t.Setenv("INPUT_FEATURES", "")
	t.Setenv("INPUT_BUGS", "")
	t.Setenv("INPUT_SECURITY", "")
	t.Setenv("INPUT_DEPRECATIONS", "")
	t.Setenv("INPUT_SUPPORTEDOPERATINGSYSTEMS", "")
	t.Setenv("INPUT_EOL", "")

	metadata, err := LoadMetadata()
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", metadata.Version)
	assert.Empty(t, metadata.Features)
	assert.Empty(t, metadata.Bugs)
	assert.Empty(t, metadata.Security)
	assert.Empty(t, metadata.Deprecations)
	assert.Empty(t, metadata.SupportedOperatingSystems)
	assert.Empty(t, metadata.EOL)
}

func TestLoadMetadata_NoVersion_Error(t *testing.T) {
	t.Setenv("INPUT_VERSION", "")
	t.Setenv("INPUT_FEATURES", "feature1")

	_, err := LoadMetadata()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to determine version")
}
