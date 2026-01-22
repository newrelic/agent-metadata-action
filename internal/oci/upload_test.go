package oci

import (
	"agent-metadata-action/internal/models"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockClient is a mock implementation of Client for testing
type mockClient struct {
	uploadFunc func(ctx context.Context, artifact *models.ArtifactDefinition, artifactPath, agentType, version string) (string, int64, error)
}

func (m *mockClient) UploadArtifact(ctx context.Context, artifact *models.ArtifactDefinition, artifactPath, agentType, version string) (string, int64, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, artifact, artifactPath, agentType, version)
	}
	return "", 0, errors.New("mock not configured")
}

func TestUploadArtifacts_Success_SingleArtifact(t *testing.T) {
	ctx := context.Background()
	workspace := "/workspace"
	agentType := "NRDotNetAgent"
	version := "1.0.0"

	config := &models.OCIConfig{
		Artifacts: []models.ArtifactDefinition{
			{
				Name:   "test-artifact",
				Path:   "./dist/agent.tar.gz",
				OS:     "linux",
				Arch:   "amd64",
				Format: "tar+gzip",
			},
		},
	}

	mock := &mockClient{
		uploadFunc: func(ctx context.Context, artifact *models.ArtifactDefinition, artifactPath, agentType, version string) (string, int64, error) {
			assert.Equal(t, "/workspace/dist/agent.tar.gz", artifactPath)
			assert.Equal(t, "NRDotNetAgent", agentType)
			assert.Equal(t, "1.0.0", version)
			return "sha256:abc123", int64(1024), nil
		},
	}

	results := UploadArtifacts(ctx, mock, config, workspace, agentType, version)

	assert.Len(t, results, 1)
	assert.Equal(t, "test-artifact", results[0].Name)
	assert.Equal(t, "./dist/agent.tar.gz", results[0].Path)
	assert.Equal(t, "linux", results[0].OS)
	assert.Equal(t, "amd64", results[0].Arch)
	assert.Equal(t, "tar+gzip", results[0].Format)
	assert.True(t, results[0].Uploaded)
	assert.Equal(t, "sha256:abc123", results[0].Digest)
	assert.Equal(t, int64(1024), results[0].Size)
	assert.Empty(t, results[0].Error)
}

func TestUploadArtifacts_UploadError(t *testing.T) {
	ctx := context.Background()
	workspace := "/workspace"
	agentType := "NRDotNetAgent"
	version := "1.0.0"

	config := &models.OCIConfig{
		Artifacts: []models.ArtifactDefinition{
			{
				Name:   "failing-artifact",
				Path:   "./dist/agent.tar.gz",
				OS:     "linux",
				Arch:   "amd64",
				Format: "tar+gzip",
			},
		},
	}

	expectedError := "failed to push artifact to registry"
	mock := &mockClient{
		uploadFunc: func(ctx context.Context, artifact *models.ArtifactDefinition, artifactPath, agentType, version string) (string, int64, error) {
			return "", 0, errors.New(expectedError)
		},
	}

	results := UploadArtifacts(ctx, mock, config, workspace, agentType, version)

	assert.Len(t, results, 1)
	assert.Equal(t, "failing-artifact", results[0].Name)
	assert.False(t, results[0].Uploaded)
	assert.Empty(t, results[0].Digest)
	assert.Equal(t, int64(0), results[0].Size)
	assert.Equal(t, expectedError, results[0].Error)
}

func TestUploadArtifacts_EmptyArtifactsList(t *testing.T) {
	ctx := context.Background()
	workspace := "/workspace"
	agentType := "NRDotNetAgent"
	version := "1.0.0"

	config := &models.OCIConfig{
		Artifacts: []models.ArtifactDefinition{},
	}

	mock := &mockClient{
		uploadFunc: func(ctx context.Context, artifact *models.ArtifactDefinition, artifactPath, agentType, version string) (string, int64, error) {
			t.Fatal("UploadArtifact should not be called with empty artifacts list")
			return "", 0, nil
		},
	}

	results := UploadArtifacts(ctx, mock, config, workspace, agentType, version)

	assert.Empty(t, results)
}
