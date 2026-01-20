package oci

import (
	"agent-metadata-action/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasFailures(t *testing.T) {
	tests := []struct {
		name     string
		results  []models.ArtifactUploadResult
		expected bool
	}{
		{
			name:     "empty results",
			results:  []models.ArtifactUploadResult{},
			expected: false,
		},
		{
			name: "all successful",
			results: []models.ArtifactUploadResult{
				{Name: "artifact1", Uploaded: true},
				{Name: "artifact2", Uploaded: true},
			},
			expected: false,
		},
		{
			name: "single success",
			results: []models.ArtifactUploadResult{
				{Name: "artifact1", Uploaded: true},
			},
			expected: false,
		},
		{
			name: "single failure",
			results: []models.ArtifactUploadResult{
				{Name: "artifact1", Uploaded: false, Error: "upload failed"},
			},
			expected: true,
		},
		{
			name: "mixed success and failure",
			results: []models.ArtifactUploadResult{
				{Name: "artifact1", Uploaded: true},
				{Name: "artifact2", Uploaded: false, Error: "upload failed"},
			},
			expected: true,
		},
		{
			name: "all failures",
			results: []models.ArtifactUploadResult{
				{Name: "artifact1", Uploaded: false, Error: "error 1"},
				{Name: "artifact2", Uploaded: false, Error: "error 2"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasFailures(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleUploads_DisabledConfig(t *testing.T) {
	config := &models.OCIConfig{
		Registry: "", // Empty registry = disabled
	}

	err := HandleUploads(config, "/workspace", "dotnet-agent", "1.0.0")
	assert.NoError(t, err, "Should not error when OCI upload is disabled")
}

func TestHandleUploads_ValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()

	config := &models.OCIConfig{
		Registry: "ghcr.io/test/agents",
		Username: "user",
		Password: "pass",
		Artifacts: []models.ArtifactDefinition{
			{
				Name:   "test-artifact",
				Path:   "nonexistent.tar.gz", // File doesn't exist
				OS:     "linux",
				Arch:   "amd64",
				Format: "tar+gzip",
			},
		},
	}

	err := HandleUploads(config, tmpDir, "dotnet-agent", "1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binary validation failed")
}
