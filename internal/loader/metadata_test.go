package loader

import (
	"os"
	"path/filepath"
	"testing"

	"agent-metadata-action/internal/github"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMetadataForAgents(t *testing.T) {
	t.Setenv("INPUT_AGENT_TYPE", "myagenttype")
	t.Setenv("INPUT_VERSION", "1.2.3")

	metadata := LoadMetadataForAgents("1.2.3")
	assert.Equal(t, "1.2.3", metadata["version"])
	assert.Nil(t, metadata["features"])
	assert.Nil(t, metadata["bugs"])
	assert.Nil(t, metadata["security"])
	assert.Nil(t, metadata["deprecations"])
	assert.Nil(t, metadata["supportedOperatingSystems"])
	assert.Nil(t, metadata["eol"])
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
subject: Java agent
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
subject: Node.js agent
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
	assert.Equal(t, "java-agent", metadata[0].AgentType)
	assert.Equal(t, "1.5.0", metadata[0].AgentMetadataFromDocs["version"])
	assert.Equal(t, []interface{}{"Added new monitoring capability", "Improved performance"}, metadata[0].AgentMetadataFromDocs["features"])
	assert.Equal(t, []interface{}{"Fixed memory leak"}, metadata[0].AgentMetadataFromDocs["bugs"])
	assert.Equal(t, []interface{}{"Patched CVE-2024-1234"}, metadata[0].AgentMetadataFromDocs["security"])
	assert.Equal(t, []interface{}{"Removed legacy API"}, metadata[0].AgentMetadataFromDocs["deprecations"])
	assert.Equal(t, []interface{}{"Windows", "Linux", "macOS"}, metadata[0].AgentMetadataFromDocs["supportedOperatingSystems"])
	assert.Equal(t, "2025-12-31", metadata[0].AgentMetadataFromDocs["eol"])

	// Verify second file's metadata
	assert.Equal(t, "nodejs-agent", metadata[1].AgentType)
	assert.Equal(t, "1.6.0", metadata[1].AgentMetadataFromDocs["version"])
	assert.Equal(t, []interface{}{"New dashboard feature"}, metadata[1].AgentMetadataFromDocs["features"])
	assert.Equal(t, []interface{}{"Fixed crash on startup"}, metadata[1].AgentMetadataFromDocs["bugs"])
}

func TestLoadMetadata_NoMDXFiles_ReturnsEmptyMetadata(t *testing.T) {
	// Don't set GITHUB_EVENT_PATH - simulates no PR context
	err := os.Unsetenv("GITHUB_EVENT_PATH")
	if err != nil {
		return
	}

	metadata, err := LoadMetadataForDocs()
	require.Error(t, err)
	assert.Nil(t, metadata)
}
