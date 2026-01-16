package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArtifactDefinition_Validate(t *testing.T) {
	tests := []struct {
		name        string
		artifact    ArtifactDefinition
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid artifact with tar+gzip",
			artifact: ArtifactDefinition{
				Name:   "linux-amd64",
				Path:   "./dist/agent.tar.gz",
				OS:     "linux",
				Arch:   "amd64",
				Format: "tar+gzip",
			},
			expectError: false,
		},
		{
			name: "valid artifact with zip",
			artifact: ArtifactDefinition{
				Name:   "windows-amd64",
				Path:   "./dist/agent.zip",
				OS:     "windows",
				Arch:   "amd64",
				Format: "zip",
			},
			expectError: false,
		},
		{
			name: "valid artifact with os=any and arch=any",
			artifact: ArtifactDefinition{
				Name:   "java-agent",
				Path:   "./dist/agent.jar.tar.gz",
				OS:     "any",
				Arch:   "any",
				Format: "tar+gzip",
			},
			expectError: false,
		},
		{
			name: "missing name",
			artifact: ArtifactDefinition{
				Path:   "./dist/agent.tar.gz",
				OS:     "linux",
				Arch:   "amd64",
				Format: "tar+gzip",
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "invalid name with spaces",
			artifact: ArtifactDefinition{
				Name:   "linux amd64",
				Path:   "./dist/agent.tar.gz",
				OS:     "linux",
				Arch:   "amd64",
				Format: "tar+gzip",
			},
			expectError: true,
			errorMsg:    "invalid artifact name",
		},
		{
			name: "missing path",
			artifact: ArtifactDefinition{
				Name:   "linux-amd64",
				OS:     "linux",
				Arch:   "amd64",
				Format: "tar+gzip",
			},
			expectError: true,
			errorMsg:    "path is required",
		},
		{
			name: "missing os",
			artifact: ArtifactDefinition{
				Name:   "linux-amd64",
				Path:   "./dist/agent.tar.gz",
				Arch:   "amd64",
				Format: "tar+gzip",
			},
			expectError: true,
			errorMsg:    "os is required",
		},
		{
			name: "missing arch",
			artifact: ArtifactDefinition{
				Name:   "linux-amd64",
				Path:   "./dist/agent.tar.gz",
				OS:     "linux",
				Format: "tar+gzip",
			},
			expectError: true,
			errorMsg:    "arch is required",
		},
		{
			name: "missing format",
			artifact: ArtifactDefinition{
				Name: "linux-amd64",
				Path: "./dist/agent.tar.gz",
				OS:   "linux",
				Arch: "amd64",
			},
			expectError: true,
			errorMsg:    "format is required",
		},
		{
			name: "invalid format",
			artifact: ArtifactDefinition{
				Name:   "linux-amd64",
				Path:   "./dist/agent.rar",
				OS:     "linux",
				Arch:   "amd64",
				Format: "rar",
			},
			expectError: true,
			errorMsg:    "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.artifact.Validate()
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

func TestOCIConfig_ValidateUniqueNames(t *testing.T) {
	tests := []struct {
		name        string
		config      OCIConfig
		expectError bool
	}{
		{
			name: "unique names",
			config: OCIConfig{
				Artifacts: []ArtifactDefinition{
					{Name: "artifact1"},
					{Name: "artifact2"},
					{Name: "artifact3"},
				},
			},
			expectError: false,
		},
		{
			name: "duplicate names",
			config: OCIConfig{
				Artifacts: []ArtifactDefinition{
					{Name: "artifact1"},
					{Name: "artifact2"},
					{Name: "artifact1"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateUniqueNames()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "duplicate artifact name")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestArtifactDefinition_GetMediaType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"tar", "application/vnd.newrelic.agent.v1+tar"},
		{"tar+gzip", "application/vnd.newrelic.agent.v1+tar+gzip"},
		{"zip", "application/vnd.newrelic.agent.v1+zip"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			artifact := ArtifactDefinition{Format: tt.format}
			assert.Equal(t, tt.expected, artifact.GetMediaType())
		})
	}
}

func TestArtifactDefinition_GetArtifactType(t *testing.T) {
	artifact := ArtifactDefinition{Format: "tar+gzip"}
	assert.Equal(t, "application/vnd.newrelic.agent.v1+tar+gzip", artifact.GetArtifactType())
}

func TestArtifactDefinition_GetPlatformString(t *testing.T) {
	tests := []struct {
		os       string
		arch     string
		expected string
	}{
		{"linux", "amd64", "linux/amd64"},
		{"windows", "amd64", "windows/amd64"},
		{"any", "any", "any/any"},
	}

	for _, tt := range tests {
		t.Run(tt.os+"/"+tt.arch, func(t *testing.T) {
			artifact := ArtifactDefinition{OS: tt.os, Arch: tt.arch}
			assert.Equal(t, tt.expected, artifact.GetPlatformString())
		})
	}
}

func TestArtifactDefinition_GetFilename(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"./dist/agent.tar.gz", "agent.tar.gz"},
		{"/absolute/path/to/file.zip", "file.zip"},
		{"simple.tar.gz", "simple.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			artifact := ArtifactDefinition{Path: tt.path}
			assert.Equal(t, tt.expected, artifact.GetFilename())
		})
	}
}
