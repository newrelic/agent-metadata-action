package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestLoadMetadata_VersionOnly(t *testing.T) {
	t.Setenv("INPUT_VERSION", "2.0.0")

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

	_, err := LoadMetadata()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to determine version")
}

func TestLoadMetadata_WithMDXFiles_Success(t *testing.T) {
	// Skip if not in a git repository
	if _, err := exec.Command("git", "rev-parse", "HEAD").Output(); err != nil {
		t.Skip("Skipping test: not in a git repository")
	}

	// Get the main branch SHA and current HEAD SHA
	baseCmd := exec.Command("git", "rev-parse", "main")
	baseOut, err := baseCmd.Output()
	if err != nil {
		t.Skip("Skipping test: main branch doesn't exist")
	}
	baseSHA := strings.TrimSpace(string(baseOut))

	headCmd := exec.Command("git", "rev-parse", "HEAD")
	headOut, err := headCmd.Output()
	if err != nil {
		t.Skip("Skipping test: not in a git repository")
	}
	headSHA := strings.TrimSpace(string(headOut))

	// Check if we're on a branch with committed MDX changes
	diffCmd := exec.Command("git", "diff", "--name-only", fmt.Sprintf("%s...%s", baseSHA, headSHA))
	diffOut, err := diffCmd.Output()
	if err != nil {
		t.Skip("Skipping test: git diff failed")
	}

	hasMDXChanges := false
	for _, line := range strings.Split(string(diffOut), "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), ".mdx") &&
			strings.Contains(line, "src/content/docs/release-notes") {
			hasMDXChanges = true
			break
		}
	}
	if !hasMDXChanges {
		t.Skip("Skipping test: no committed MDX changes found (commit integration test files to test)")
	}

	// Create mock PR event with real git SHAs
	event := struct {
		PullRequest struct {
			Base struct {
				SHA string `json:"sha"`
			} `json:"base"`
			Head struct {
				SHA string `json:"sha"`
			} `json:"head"`
		} `json:"pull_request"`
	}{}
	event.PullRequest.Base.SHA = baseSHA
	event.PullRequest.Head.SHA = headSHA

	eventData, err := json.Marshal(event)
	require.NoError(t, err)

	tmpEventFile := filepath.Join(t.TempDir(), "event.json")
	err = os.WriteFile(tmpEventFile, eventData, 0644)
	require.NoError(t, err)

	// Get current working directory as workspace
	workspace, err := os.Getwd()
	require.NoError(t, err)
	// Navigate up to the project root
	workspace = filepath.Join(workspace, "../..")

	// Set environment variables
	t.Setenv("INPUT_VERSION", "1.5.0")
	t.Setenv("GITHUB_EVENT_PATH", tmpEventFile)
	t.Setenv("GITHUB_WORKSPACE", workspace)

	// Load metadata
	metadata, err := LoadMetadata()
	require.NoError(t, err)

	// Verify version is set
	assert.Equal(t, "1.5.0", metadata.Version)

	// The actual metadata content depends on what's in the committed files
	// We just verify that if there are changed MDX files, we get some metadata
	t.Logf("Loaded metadata: features=%d, bugs=%d, security=%d",
		len(metadata.Features), len(metadata.Bugs), len(metadata.Security))
}

func TestLoadMetadata_NoMDXFiles_ReturnsEmptyMetadata(t *testing.T) {
	t.Setenv("INPUT_VERSION", "3.0.0")

	// Don't set GITHUB_EVENT_PATH - simulates no PR context
	os.Unsetenv("GITHUB_EVENT_PATH")

	metadata, err := LoadMetadata()
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", metadata.Version)
	assert.Nil(t, metadata.Features)
	assert.Nil(t, metadata.Bugs)
	assert.Nil(t, metadata.Security)
	assert.Nil(t, metadata.Deprecations)
	assert.Nil(t, metadata.SupportedOperatingSystems)
	assert.Empty(t, metadata.EOL)
}
