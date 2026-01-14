package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMetadataForAgents(t *testing.T) {
	t.Setenv("INPUT_AGENT_TYPE", "myagenttype")
	t.Setenv("INPUT_VERSION", "1.2.3")

	metadata := LoadMetadataForAgents("1.2.3")
	assert.Equal(t, "1.2.3", metadata["version"])
	assert.Empty(t, metadata["features"])
	assert.Empty(t, metadata["bugs"])
	assert.Empty(t, metadata["security"])
	assert.Empty(t, metadata["deprecations"])
	assert.Empty(t, metadata["supportedOperatingSystems"])
	assert.Empty(t, metadata["eol"])
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
	assert.Equal(t, "NRJavaAgent", metadata[0].AgentType)
	assert.Equal(t, "1.5.0", metadata[0].AgentMetadataFromDocs["version"])
	assert.Equal(t, []interface{}{"Added new monitoring capability", "Improved performance"}, metadata[0].AgentMetadataFromDocs["features"])
	assert.Equal(t, []interface{}{"Fixed memory leak"}, metadata[0].AgentMetadataFromDocs["bugs"])
	assert.Equal(t, []interface{}{"Patched CVE-2024-1234"}, metadata[0].AgentMetadataFromDocs["security"])
	assert.Equal(t, []interface{}{"Removed legacy API"}, metadata[0].AgentMetadataFromDocs["deprecations"])
	assert.Equal(t, []interface{}{"Windows", "Linux", "macOS"}, metadata[0].AgentMetadataFromDocs["supportedOperatingSystems"])
	assert.Equal(t, "2025-12-31", metadata[0].AgentMetadataFromDocs["eol"])

	// Verify second file's metadata
	assert.Equal(t, "NRNodeAgent", metadata[1].AgentType)
	assert.Equal(t, "1.6.0", metadata[1].AgentMetadataFromDocs["version"])
	assert.Equal(t, []interface{}{"New dashboard feature"}, metadata[1].AgentMetadataFromDocs["features"])
	assert.Equal(t, []interface{}{"Fixed crash on startup"}, metadata[1].AgentMetadataFromDocs["bugs"])
}

func TestLoadMetadataForDocs_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (tmpWorkspace string, mdxFiles []string)
		expectError   bool
		expectedInErr string
		expectedInLog string
	}{
		{
			name: "github GetChangedMDXFiles returns error",
			setupFunc: func(t *testing.T) (string, []string) {
				// Mock to return error
				originalFunc := github.GetChangedMDXFilesFunc
				github.GetChangedMDXFilesFunc = func() ([]string, error) {
					return nil, fmt.Errorf("git error")
				}
				t.Cleanup(func() {
					github.GetChangedMDXFilesFunc = originalFunc
				})
				return "", nil
			},
			expectError:   true,
			expectedInErr: "could not get changed files",
		},
		{
			name: "blank version",
			setupFunc: func(t *testing.T) (string, []string) {
				tmpWorkspace := t.TempDir()
				releaseNotesDir := filepath.Join(tmpWorkspace, "src/content/docs/release-notes/agent-release-notes")
				require.NoError(t, os.MkdirAll(releaseNotesDir, 0755))

				mdxContent := `---
subject: Java agent
releaseDate: '2024-01-15'
version: ""
features:
  - New feature
---

# Test Release Notes
`
				mdxFile := filepath.Join(releaseNotesDir, "test-agent.mdx")
				require.NoError(t, os.WriteFile(mdxFile, []byte(mdxContent), 0644))
				return tmpWorkspace, []string{mdxFile}
			},
			expectError:   true,
			expectedInErr: "unable to load metadata for any",
			expectedInLog: "Version is required",
		},
		{
			name: "missing subject",
			setupFunc: func(t *testing.T) (string, []string) {
				tmpWorkspace := t.TempDir()
				releaseNotesDir := filepath.Join(tmpWorkspace, "src/content/docs/release-notes/agent-release-notes")
				require.NoError(t, os.MkdirAll(releaseNotesDir, 0755))

				mdxContent := `---
releaseDate: '2024-01-15'
version: 1.2.3
features:
  - New feature
---

# Test Release Notes
`
				mdxFile := filepath.Join(releaseNotesDir, "test-agent.mdx")
				require.NoError(t, os.WriteFile(mdxFile, []byte(mdxContent), 0644))
				return tmpWorkspace, []string{mdxFile}
			},
			expectError:   true,
			expectedInErr: "unable to load metadata for any",
			expectedInLog: "Subject (to derive agent type) is required",
		},
		{
			name: "empty subject",
			setupFunc: func(t *testing.T) (string, []string) {
				tmpWorkspace := t.TempDir()
				releaseNotesDir := filepath.Join(tmpWorkspace, "src/content/docs/release-notes/agent-release-notes")
				require.NoError(t, os.MkdirAll(releaseNotesDir, 0755))

				mdxContent := `---
subject: ""
releaseDate: '2024-01-15'
version: 1.2.3
features:
  - New feature
---

# Test Release Notes
`
				mdxFile := filepath.Join(releaseNotesDir, "test-agent.mdx")
				require.NoError(t, os.WriteFile(mdxFile, []byte(mdxContent), 0644))
				return tmpWorkspace, []string{mdxFile}
			},
			expectError:   true,
			expectedInErr: "unable to load metadata for any",
			expectedInLog: "Subject (to derive agent type) is required",
		},
		{
			name: "malformed MDX file",
			setupFunc: func(t *testing.T) (string, []string) {
				tmpWorkspace := t.TempDir()
				releaseNotesDir := filepath.Join(tmpWorkspace, "src/content/docs/release-notes/agent-release-notes")
				require.NoError(t, os.MkdirAll(releaseNotesDir, 0755))

				// Invalid YAML frontmatter
				mdxContent := `---
subject: Java agent
invalid yaml: [unclosed
version: 1.2.3
---

# Test Release Notes
`
				mdxFile := filepath.Join(releaseNotesDir, "test-agent.mdx")
				require.NoError(t, os.WriteFile(mdxFile, []byte(mdxContent), 0644))
				return tmpWorkspace, []string{mdxFile}
			},
			expectError:   true,
			expectedInErr: "unable to load metadata for any",
			expectedInLog: "Failed to parse MDX file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getStdout, _ := testutil.CaptureOutput(t)

			tmpWorkspace, mdxFiles := tt.setupFunc(t)

			// Mock GetChangedMDXFiles if files provided
			if mdxFiles != nil {
				originalFunc := github.GetChangedMDXFilesFunc
				github.GetChangedMDXFilesFunc = func() ([]string, error) {
					return mdxFiles, nil
				}
				defer func() {
					github.GetChangedMDXFilesFunc = originalFunc
				}()
			}

			if tmpWorkspace != "" {
				t.Setenv("GITHUB_WORKSPACE", tmpWorkspace)
			}

			// method under test
			metadata, err := LoadMetadataForDocs()

			stdout := getStdout()

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, metadata)
				if tt.expectedInErr != "" {
					assert.Contains(t, err.Error(), tt.expectedInErr)
				}
				if tt.expectedInLog != "" {
					assert.Contains(t, stdout, tt.expectedInLog)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadMetadataForDocs_NoChangedFiles(t *testing.T) {
	// Mock GetChangedMDXFiles to return empty list (not error)
	originalFunc := github.GetChangedMDXFilesFunc
	github.GetChangedMDXFilesFunc = func() ([]string, error) {
		return []string{}, nil
	}
	defer func() {
		github.GetChangedMDXFilesFunc = originalFunc
	}()

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	metadata, err := LoadMetadataForDocs()

	stdout := getStdout()

	require.NoError(t, err)
	assert.Nil(t, metadata)
	assert.Contains(t, stdout, "no changed files detected")
}
