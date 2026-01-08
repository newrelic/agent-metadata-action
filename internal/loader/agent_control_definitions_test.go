package loader

import (
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/testutil"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAgentControlDefinitions_Success(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
	err := os.MkdirAll(agentControlDir, 0755)
	require.NoError(t, err)

	// Create test agent control file
	agentControlContent := `schema:
  type: object
  properties:
    setting1:
      type: string
    setting2:
      type: boolean`
	agentControlFile := filepath.Join(agentControlDir, "agent-schema-for-agent-control.yml")
	err = os.WriteFile(agentControlFile, []byte(agentControlContent), 0644)
	require.NoError(t, err)

	// Test reading the agent control
	agentControl, err := ReadAgentControlDefinitions(tmpDir)
	require.NoError(t, err)
	assert.Len(t, agentControl, 1)
	assert.Equal(t, AgentControlPlatform, agentControl[0].Platform)
	assert.NotEmpty(t, agentControl[0].Content)

	// Verify content was base64 encoded
	expectedEncoded := base64.StdEncoding.EncodeToString([]byte(agentControlContent))
	assert.Equal(t, expectedEncoded, agentControl[0].Content)

	// Verify we can decode it back
	decoded, err := base64.StdEncoding.DecodeString(agentControl[0].Content)
	require.NoError(t, err)
	assert.Equal(t, agentControlContent, string(decoded))
}

func TestReadAgentControlDefinitions_DirectoryNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	agentControl, err := ReadAgentControlDefinitions(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, agentControl)
	assert.Empty(t, agentControl)
}

func TestReadAgentControlDefinitions_EmptyFile(t *testing.T) {
	// Create temporary directory structure with empty agent control file
	tmpDir := t.TempDir()
	agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
	err := os.MkdirAll(agentControlDir, 0755)
	require.NoError(t, err)

	// Create empty agent control file
	agentControlFile := filepath.Join(agentControlDir, "agent-schema-for-agent-control.yml")
	err = os.WriteFile(agentControlFile, []byte(""), 0644)
	require.NoError(t, err)

	getStdout, _ := testutil.CaptureOutput(t)

	// Test reading the agent control - should not fail if file can't be loaded
	agentControl, err := ReadAgentControlDefinitions(tmpDir)

	outputStr := getStdout()

	require.NoError(t, err)
	assert.NotNil(t, agentControl)
	assert.Contains(t, outputStr, "failed to load agent control file")
}

func TestReadAgentControlDefinitions_SkipNonYAMLFiles(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
	err := os.MkdirAll(agentControlDir, 0755)
	require.NoError(t, err)

	// Create a subdirectory that should be skipped by glob
	subDir := filepath.Join(agentControlDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create non-YAML files that should be skipped by glob
	txtFile := filepath.Join(agentControlDir, "readme.txt")
	err = os.WriteFile(txtFile, []byte("text file"), 0644)
	require.NoError(t, err)

	jsonFile := filepath.Join(agentControlDir, "config.json")
	err = os.WriteFile(jsonFile, []byte(`{"key": "value"}`), 0644)
	require.NoError(t, err)

	// Create a valid agent control file
	agentControlContent := `schema:
  type: object`
	agentControlFile := filepath.Join(agentControlDir, "agent-control.yml")
	err = os.WriteFile(agentControlFile, []byte(agentControlContent), 0644)
	require.NoError(t, err)

	// Test reading - should skip non-YAML files and only read YAML file
	agentControl, err := ReadAgentControlDefinitions(tmpDir)
	require.NoError(t, err)
	assert.Len(t, agentControl, 1, "Should only have 1 entry, non-YAML files should be skipped")
	assert.Equal(t, AgentControlPlatform, agentControl[0].Platform)
	assert.NotEmpty(t, agentControl[0].Content)
}

func TestReadAgentControlDefinitions_MultipleFiles(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
	err := os.MkdirAll(agentControlDir, 0755)
	require.NoError(t, err)

	// Create multiple agent control files
	file1Content := `schema:
  type: object
  properties:
    field1:
      type: string`
	file1 := filepath.Join(agentControlDir, "control1.yml")
	err = os.WriteFile(file1, []byte(file1Content), 0644)
	require.NoError(t, err)

	file2Content := `schema:
  type: object
  properties:
    field2:
      type: number`
	file2 := filepath.Join(agentControlDir, "control2.yaml")
	err = os.WriteFile(file2, []byte(file2Content), 0644)
	require.NoError(t, err)

	file3Content := `schema:
  type: object
  properties:
    field3:
      type: boolean`
	file3 := filepath.Join(agentControlDir, "control3.yml")
	err = os.WriteFile(file3, []byte(file3Content), 0644)
	require.NoError(t, err)

	// Test reading the agent control files
	agentControl, err := ReadAgentControlDefinitions(tmpDir)
	require.NoError(t, err)
	assert.Len(t, agentControl, 3)

	// Verify all entries have correct platform and content
	for _, control := range agentControl {
		assert.Equal(t, AgentControlPlatform, control.Platform)
		assert.NotEmpty(t, control.Content)
	}

	// Verify at least one can be decoded
	decoded, err := base64.StdEncoding.DecodeString(agentControl[0].Content)
	require.NoError(t, err)
	assert.NotEmpty(t, string(decoded))
}
