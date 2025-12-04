package config

import (
	"os"
	"path/filepath"
	"testing"

	"agent-metadata-action/internal/github"

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

func TestLoadMetadataForAgents(t *testing.T) {
	t.Setenv("INPUT_AGENT_TYPE", "myagenttype")
	t.Setenv("INPUT_VERSION", "1.2.3")

	metadata := LoadMetadataForAgents("1.2.3")
	assert.Equal(t, "1.2.3", metadata.Version)
	assert.Empty(t, metadata.Features)
	assert.Empty(t, metadata.Bugs)
	assert.Empty(t, metadata.Security)
	assert.Empty(t, metadata.Deprecations)
	assert.Empty(t, metadata.SupportedOperatingSystems)
	assert.Empty(t, metadata.EOL)
}

func TestLoadMetadata_WithMDXFiles_Success(t *testing.T) {
	// Create a temporary workspace
	tmpWorkspace := t.TempDir()

	// Create the release notes directory structure
	releaseNotesDir := filepath.Join(tmpWorkspace, "src/content/docs/release-notes/agent-release-notes")
	err := os.MkdirAll(releaseNotesDir, 0755)
	require.NoError(t, err)

	// Create test MDX files with known content
	mdxContent1 := `---
subject: Test Agent
releaseDate: '2024-01-15'
version: 1.5.0
features:
  - Added new monitoring capability
  - Improved performance
bugs:
  - Fixed memory leak
security:
  - Patched CVE-2024-1234
deprecations:
  - Removed legacy API
supportedOperatingSystems:
  - Windows
  - Linux
  - macOS
eol: '2025-12-31'
---

# Test Release Notes

This is a test release.
`

	mdxContent2 := `---
subject: Another Agent
releaseDate: '2024-01-16'
version: 1.6.0
features:
  - New dashboard feature
bugs:
  - Fixed crash on startup
---

# Another Release
`

	mdxFile1 := filepath.Join(releaseNotesDir, "test-agent-1.5.0.mdx")
	mdxFile2 := filepath.Join(releaseNotesDir, "test-agent-1.6.0.mdx")

	err = os.WriteFile(mdxFile1, []byte(mdxContent1), 0644)
	require.NoError(t, err)

	err = os.WriteFile(mdxFile2, []byte(mdxContent2), 0644)
	require.NoError(t, err)

	// Mock GetChangedMDXFiles to return our test files
	originalFunc := github.GetChangedMDXFilesFunc
	github.GetChangedMDXFilesFunc = func() ([]string, error) {
		return []string{mdxFile1, mdxFile2}, nil
	}
	defer func() {
		github.GetChangedMDXFilesFunc = originalFunc
	}()

	// Set environment variables
	t.Setenv("INPUT_VERSION", "1.5.0")
	t.Setenv("GITHUB_WORKSPACE", tmpWorkspace)

	// Load metadata
	metadata, err := LoadMetadataForDocs()
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Len(t, metadata, 2, "Should load 2 MDX files")

	// Verify first file's metadata
	assert.Equal(t, "1.5.0", metadata[0].Version)
	assert.Equal(t, []string{"Added new monitoring capability", "Improved performance"}, metadata[0].Features)
	assert.Equal(t, []string{"Fixed memory leak"}, metadata[0].Bugs)
	assert.Equal(t, []string{"Patched CVE-2024-1234"}, metadata[0].Security)
	assert.Equal(t, []string{"Removed legacy API"}, metadata[0].Deprecations)
	assert.Equal(t, []string{"Windows", "Linux", "macOS"}, metadata[0].SupportedOperatingSystems)
	assert.Equal(t, "2025-12-31", metadata[0].EOL)

	// Verify second file's metadata
	assert.Equal(t, "1.6.0", metadata[1].Version)
	assert.Equal(t, []string{"New dashboard feature"}, metadata[1].Features)
	assert.Equal(t, []string{"Fixed crash on startup"}, metadata[1].Bugs)
}

func TestLoadMetadata_NoMDXFiles_ReturnsEmptyMetadata(t *testing.T) {
	// Don't set GITHUB_EVENT_PATH - simulates no PR context
	err := os.Unsetenv("GITHUB_EVENT_PATH")
	if err != nil {
		return
	}

	metadata, err := LoadMetadataForDocs()
	require.NoError(t, err)
	assert.Nil(t, metadata)
}
