package oci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateBinaryPath(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.tar.gz")
	err := os.WriteFile(testFile, []byte("test data"), 0644)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		workspace   string
		setupFunc   func(t *testing.T) string
		binaryPath  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid relative path",
			workspace:   tmpDir,
			binaryPath:  "test.tar.gz",
			expectError: false,
		},
		{
			name:        "valid absolute path",
			workspace:   tmpDir,
			binaryPath:  testFile,
			expectError: false,
		},
		{
			name:        "path with directory traversal",
			workspace:   tmpDir,
			binaryPath:  "../test.tar.gz",
			expectError: true,
			errorMsg:    "directory traversal",
		},
		{
			name:        "path with multiple directory traversal segments",
			workspace:   tmpDir,
			binaryPath:  "../../etc/passwd",
			expectError: true,
			errorMsg:    "directory traversal",
		},
		{
			name:        "file not found",
			workspace:   tmpDir,
			binaryPath:  "nonexistent.tar.gz",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:      "valid absolute path outside workspace",
			workspace: tmpDir,
			setupFunc: func(t *testing.T) string {
				// Create test file in system temp (outside workspace)
				outsideFile := filepath.Join(os.TempDir(), "outside-workspace.tar.gz")
				err := os.WriteFile(outsideFile, []byte("test data"), 0644)
				assert.NoError(t, err)
				t.Cleanup(func() { os.Remove(outsideFile) })
				return outsideFile
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binaryPath := tt.binaryPath
			if tt.setupFunc != nil {
				binaryPath = tt.setupFunc(t)
			}

			err := ValidateBinaryPath(tt.workspace, binaryPath)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBinaryPath_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty file
	emptyFile := filepath.Join(tmpDir, "empty.tar.gz")
	err := os.WriteFile(emptyFile, []byte{}, 0644)
	assert.NoError(t, err)

	err = ValidateBinaryPath(tmpDir, "empty.tar.gz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateBinaryPath_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	subdir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subdir, 0755)
	assert.NoError(t, err)

	err = ValidateBinaryPath(tmpDir, "subdir")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "directory, not a file")
}

func TestResolveArtifactPath(t *testing.T) {
	workspace := "/workspace"

	tests := []struct {
		name         string
		artifactPath string
		expected     string
	}{
		{
			name:         "relative path",
			artifactPath: "./dist/agent.tar.gz",
			expected:     "/workspace/dist/agent.tar.gz",
		},
		{
			name:         "absolute path",
			artifactPath: "/absolute/path/agent.tar.gz",
			expected:     "/absolute/path/agent.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveArtifactPath(workspace, tt.artifactPath)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
