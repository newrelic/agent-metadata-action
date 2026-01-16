package oci

import (
	"agent-metadata-action/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateLayerAnnotations(t *testing.T) {
	artifact := &models.ArtifactDefinition{
		Name:   "test-artifact",
		Path:   "./dist/agent-linux.tar.gz",
		OS:     "linux",
		Arch:   "amd64",
		Format: "tar+gzip",
	}
	version := "1.2.3"

	annotations := CreateLayerAnnotations(artifact, version)

	assert.Equal(t, "agent-linux.tar.gz", annotations["org.opencontainers.image.title"])
	assert.Equal(t, "1.2.3", annotations["org.opencontainers.image.version"])
	assert.Equal(t, "binary", annotations["com.newrelic.artifact.type"])
}

func TestCreateManifestAnnotations(t *testing.T) {
	annotations := CreateManifestAnnotations()

	// Per specification, only the creation timestamp is included at manifest level
	assert.Len(t, annotations, 1, "Manifest should only have one annotation")
	assert.Contains(t, annotations, "org.opencontainers.image.created")
	assert.NotEmpty(t, annotations["org.opencontainers.image.created"])
}
