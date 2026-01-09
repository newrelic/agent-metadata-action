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
	tests := []struct {
		name          string
		files         map[string]string // filename -> content
		expectedCount int
	}{
		{
			name: "single yml file",
			files: map[string]string{
				"control.yml": `schema:
  type: object
  properties:
    setting1:
      type: string`,
			},
			expectedCount: 1,
		},
		{
			name: "single yaml file",
			files: map[string]string{
				"control.yaml": `schema:
  type: object`,
			},
			expectedCount: 1,
		},
		{
			name: "multiple files",
			files: map[string]string{
				"control1.yml": `schema:
  type: object
  properties:
    field1:
      type: string`,
				"control2.yaml": `schema:
  type: object
  properties:
    field2:
      type: number`,
				"control3.yml": `schema:
  type: object
  properties:
    field3:
      type: boolean`,
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
			require.NoError(t, os.MkdirAll(agentControlDir, 0755))

			// Create all test files
			for filename, content := range tt.files {
				filePath := filepath.Join(agentControlDir, filename)
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
			}

			// method under test
			agentControl, err := ReadAgentControlDefinitions(tmpDir)

			require.NoError(t, err)
			assert.Len(t, agentControl, tt.expectedCount)

			// Verify all entries have correct platform and content
			for _, control := range agentControl {
				assert.Equal(t, AgentControlPlatform, control.Platform)
				assert.NotEmpty(t, control.Content)

				// Verify content can be decoded
				decoded, err := base64.StdEncoding.DecodeString(control.Content)
				require.NoError(t, err)
				assert.NotEmpty(t, string(decoded))
			}

			// For single file tests, verify exact content
			if tt.expectedCount == 1 {
				var expectedContent string
				for _, content := range tt.files {
					expectedContent = content
					break
				}
				expectedEncoded := base64.StdEncoding.EncodeToString([]byte(expectedContent))
				assert.Equal(t, expectedEncoded, agentControl[0].Content)
			}
		})
	}
}

func TestReadAgentControlDefinitions_ErrorCases(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, tmpDir string)
		expectedCount  int
		expectWarning  bool
		warningMessage string
	}{
		{
			name: "directory not found",
			setupFunc: func(t *testing.T, tmpDir string) {
				// Don't create the directory
			},
			expectedCount: 0,
			expectWarning: false,
		},
		{
			name: "empty file",
			setupFunc: func(t *testing.T, tmpDir string) {
				agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(agentControlDir, 0755))
				agentControlFile := filepath.Join(agentControlDir, "empty.yml")
				require.NoError(t, os.WriteFile(agentControlFile, []byte(""), 0644))
			},
			expectedCount:  0,
			expectWarning:  true,
			warningMessage: "failed to load agent control file",
		},
		{
			name: "skip non-YAML files",
			setupFunc: func(t *testing.T, tmpDir string) {
				agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(agentControlDir, 0755))

				// Create non-YAML files that should be skipped
				require.NoError(t, os.WriteFile(filepath.Join(agentControlDir, "readme.txt"), []byte("text"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(agentControlDir, "config.json"), []byte("{}"), 0644))

				// Create one valid YAML file
				require.NoError(t, os.WriteFile(filepath.Join(agentControlDir, "valid.yml"), []byte("test: data"), 0644))
			},
			expectedCount: 1,
			expectWarning: false,
		},
		{
			name: "unreadable file",
			setupFunc: func(t *testing.T, tmpDir string) {
				agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(agentControlDir, 0755))

				// Create a file with no read permissions
				unreadableFile := filepath.Join(agentControlDir, "unreadable.yml")
				require.NoError(t, os.WriteFile(unreadableFile, []byte("test: data"), 0644))
				require.NoError(t, os.Chmod(unreadableFile, 0000))
			},
			expectedCount:  0,
			expectWarning:  true,
			warningMessage: "failed to load agent control file",
		},
		{
			name: "partial success with some failing files",
			setupFunc: func(t *testing.T, tmpDir string) {
				agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(agentControlDir, 0755))

				// Create one valid file
				require.NoError(t, os.WriteFile(filepath.Join(agentControlDir, "valid.yml"), []byte("test: data"), 0644))

				// Create one empty file (should warn and skip)
				require.NoError(t, os.WriteFile(filepath.Join(agentControlDir, "empty.yml"), []byte(""), 0644))
			},
			expectedCount:  1,
			expectWarning:  true,
			warningMessage: "failed to load agent control file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFunc(t, tmpDir)

			getStdout, _ := testutil.CaptureOutput(t)

			// method under test
			agentControl, err := ReadAgentControlDefinitions(tmpDir)

			outputStr := getStdout()

			require.NoError(t, err)
			assert.NotNil(t, agentControl)
			assert.Len(t, agentControl, tt.expectedCount)

			if tt.expectWarning {
				assert.Contains(t, outputStr, tt.warningMessage)
			}
		})
	}
}

func TestLoadAndEncodeAgentControl(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T, tmpDir string, filename string)
		filename    string
		wantErr     bool
		errContains string
	}{
		{
			name: "success",
			setupFunc: func(t *testing.T, tmpDir string, filename string) {
				agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(agentControlDir, 0755))
				filePath := filepath.Join(agentControlDir, filename)
				require.NoError(t, os.WriteFile(filePath, []byte("test: data"), 0644))
			},
			filename: "test.yml",
			wantErr:  false,
		},
		{
			name: "file does not exist",
			setupFunc: func(t *testing.T, tmpDir string, filename string) {
				// Don't create the file
				agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(agentControlDir, 0755))
			},
			filename:    "nonexistent.yml",
			wantErr:     true,
			errContains: "failed to read agent control file",
		},
		{
			name: "empty file",
			setupFunc: func(t *testing.T, tmpDir string, filename string) {
				agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(agentControlDir, 0755))
				filePath := filepath.Join(agentControlDir, filename)
				require.NoError(t, os.WriteFile(filePath, []byte(""), 0644))
			},
			filename:    "empty.yml",
			wantErr:     true,
			errContains: "is empty",
		},
		{
			name: "unreadable file",
			setupFunc: func(t *testing.T, tmpDir string, filename string) {
				agentControlDir := filepath.Join(tmpDir, config.GetAgentControlFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(agentControlDir, 0755))
				filePath := filepath.Join(agentControlDir, filename)
				require.NoError(t, os.WriteFile(filePath, []byte("test: data"), 0644))
				require.NoError(t, os.Chmod(filePath, 0000))
			},
			filename:    "unreadable.yml",
			wantErr:     true,
			errContains: "failed to read agent control file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFunc(t, tmpDir, tt.filename)

			// method under test
			result, err := loadAndEncodeAgentControl(tmpDir, tt.filename)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, AgentControlPlatform, result.Platform)
				assert.NotEmpty(t, result.Content)

				// Verify content is base64 encoded
				decoded, err := base64.StdEncoding.DecodeString(result.Content)
				require.NoError(t, err)
				assert.NotEmpty(t, string(decoded))
			}
		})
	}
}
