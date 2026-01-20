package oci

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Success(t *testing.T) {
	// Set up environment variables
	os.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
	os.Setenv("INPUT_OCI_USERNAME", "testuser")
	os.Setenv("INPUT_OCI_PASSWORD", "testpass")
	os.Setenv("INPUT_BINARIES", `[
		{
			"name": "test-binary",
			"path": "/path/to/binary",
			"os": "linux",
			"arch": "amd64",
			"format": "tar"
		}
	]`)
	defer cleanupEnv()

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "docker.io/newrelic/agents", config.Registry)
	assert.Equal(t, "testuser", config.Username)
	assert.Equal(t, "testpass", config.Password)
	assert.Len(t, config.Artifacts, 1)
	assert.Equal(t, "test-binary", config.Artifacts[0].Name)
	assert.Equal(t, "/path/to/binary", config.Artifacts[0].Path)
	assert.Equal(t, "linux", config.Artifacts[0].OS)
	assert.Equal(t, "amd64", config.Artifacts[0].Arch)
	assert.Equal(t, "tar", config.Artifacts[0].Format)
}

func TestLoadConfig_EmptyBinaries(t *testing.T) {
	// When registry is empty, validation should pass even without binaries
	os.Setenv("INPUT_OCI_REGISTRY", "")
	os.Setenv("INPUT_OCI_USERNAME", "")
	os.Setenv("INPUT_OCI_PASSWORD", "")
	os.Setenv("INPUT_BINARIES", "")
	defer cleanupEnv()

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "", config.Registry)
	assert.Equal(t, "", config.Username)
	assert.Equal(t, "", config.Password)
	assert.Len(t, config.Artifacts, 0)
	assert.False(t, config.IsEnabled())
}


func TestLoadConfig_MultipleArtifacts(t *testing.T) {
	os.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
	os.Setenv("INPUT_OCI_USERNAME", "testuser")
	os.Setenv("INPUT_OCI_PASSWORD", "testpass")
	os.Setenv("INPUT_BINARIES", `[
		{
			"name": "linux-amd64",
			"path": "/path/to/linux-amd64",
			"os": "linux",
			"arch": "amd64",
			"format": "tar"
		},
		{
			"name": "darwin-arm64",
			"path": "/path/to/darwin-arm64",
			"os": "darwin",
			"arch": "arm64",
			"format": "tar+gzip"
		},
		{
			"name": "windows-amd64",
			"path": "/path/to/windows-amd64",
			"os": "windows",
			"arch": "amd64",
			"format": "zip"
		}
	]`)
	defer cleanupEnv()

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Len(t, config.Artifacts, 3)
	assert.Equal(t, "linux-amd64", config.Artifacts[0].Name)
	assert.Equal(t, "darwin-arm64", config.Artifacts[1].Name)
	assert.Equal(t, "windows-amd64", config.Artifacts[2].Name)
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	os.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
	os.Setenv("INPUT_OCI_USERNAME", "testuser")
	os.Setenv("INPUT_OCI_PASSWORD", "testpass")
	os.Setenv("INPUT_BINARIES", `invalid json`)
	defer cleanupEnv()

	_, err := LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse binaries JSON")
}

func TestLoadConfig_ValidationFailure_NoArtifacts(t *testing.T) {
	// When registry is set but no artifacts are provided, validation should fail
	os.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
	os.Setenv("INPUT_OCI_USERNAME", "testuser")
	os.Setenv("INPUT_OCI_PASSWORD", "testpass")
	os.Setenv("INPUT_BINARIES", "")
	defer cleanupEnv()

	config, err := LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "binaries input is required when oci-registry is set")
	assert.True(t, config.IsEnabled())
}

func TestLoadConfig_ValidationFailure_EmptyArtifactArray(t *testing.T) {
	// When registry is set but empty artifact array is provided, validation should fail
	os.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
	os.Setenv("INPUT_OCI_USERNAME", "testuser")
	os.Setenv("INPUT_OCI_PASSWORD", "testpass")
	os.Setenv("INPUT_BINARIES", "[]")
	defer cleanupEnv()

	_, err := LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "binaries input is required when oci-registry is set")
}

func TestLoadConfig_ValidationFailure_InvalidArtifact(t *testing.T) {
	tests := []struct {
		name        string
		binariesJSON string
		expectedErr string
	}{
		{
			name: "missing name",
			binariesJSON: `[{
				"path": "/path/to/binary",
				"os": "linux",
				"arch": "amd64",
				"format": "tar"
			}]`,
			expectedErr: "name is required",
		},
		{
			name: "missing path",
			binariesJSON: `[{
				"name": "test-binary",
				"os": "linux",
				"arch": "amd64",
				"format": "tar"
			}]`,
			expectedErr: "path is required",
		},
		{
			name: "missing os",
			binariesJSON: `[{
				"name": "test-binary",
				"path": "/path/to/binary",
				"arch": "amd64",
				"format": "tar"
			}]`,
			expectedErr: "os is required",
		},
		{
			name: "missing arch",
			binariesJSON: `[{
				"name": "test-binary",
				"path": "/path/to/binary",
				"os": "linux",
				"format": "tar"
			}]`,
			expectedErr: "arch is required",
		},
		{
			name: "missing format",
			binariesJSON: `[{
				"name": "test-binary",
				"path": "/path/to/binary",
				"os": "linux",
				"arch": "amd64"
			}]`,
			expectedErr: "format is required",
		},
		{
			name: "invalid format",
			binariesJSON: `[{
				"name": "test-binary",
				"path": "/path/to/binary",
				"os": "linux",
				"arch": "amd64",
				"format": "invalid"
			}]`,
			expectedErr: "invalid format",
		},
		{
			name: "invalid artifact name",
			binariesJSON: `[{
				"name": "test binary!",
				"path": "/path/to/binary",
				"os": "linux",
				"arch": "amd64",
				"format": "tar"
			}]`,
			expectedErr: "invalid artifact name",
		},
		{
			name: "duplicate names",
			binariesJSON: `[
				{
					"name": "test-binary",
					"path": "/path/to/binary1",
					"os": "linux",
					"arch": "amd64",
					"format": "tar"
				},
				{
					"name": "test-binary",
					"path": "/path/to/binary2",
					"os": "darwin",
					"arch": "arm64",
					"format": "tar"
				}
			]`,
			expectedErr: "duplicate artifact name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
			os.Setenv("INPUT_OCI_USERNAME", "testuser")
			os.Setenv("INPUT_OCI_PASSWORD", "testpass")
			os.Setenv("INPUT_BINARIES", tt.binariesJSON)
			defer cleanupEnv()

			config, err := LoadConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.True(t, config.IsEnabled())
		})
	}
}

func TestLoadConfig_NoCredentials(t *testing.T) {
	// For local registries, credentials are optional
	os.Setenv("INPUT_OCI_REGISTRY", "localhost:5000")
	os.Setenv("INPUT_OCI_USERNAME", "")
	os.Setenv("INPUT_OCI_PASSWORD", "")
	os.Setenv("INPUT_BINARIES", `[
		{
			"name": "test-binary",
			"path": "/path/to/binary",
			"os": "linux",
			"arch": "amd64",
			"format": "tar"
		}
	]`)
	defer cleanupEnv()

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "localhost:5000", config.Registry)
	assert.Equal(t, "", config.Username)
	assert.Equal(t, "", config.Password)
	assert.Len(t, config.Artifacts, 1)
}

// cleanupEnv clears all OCI-related environment variables
func cleanupEnv() {
	os.Unsetenv("INPUT_OCI_REGISTRY")
	os.Unsetenv("INPUT_OCI_USERNAME")
	os.Unsetenv("INPUT_OCI_PASSWORD")
	os.Unsetenv("INPUT_BINARIES")
}
