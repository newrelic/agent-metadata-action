package oci

import (
	"agent-metadata-action/internal/models"
	"time"
)

func CreateLayerAnnotations(artifact *models.ArtifactDefinition, version string) map[string]string {
	return map[string]string{
		"org.opencontainers.image.title":   artifact.GetFilename(),
		"org.opencontainers.image.version": version,
		"com.newrelic.artifact.type":       "binary",
	}
}

func CreateManifestAnnotations() map[string]string {
	return map[string]string{
		"org.opencontainers.image.created": time.Now().UTC().Format(time.RFC3339),
	}
}
