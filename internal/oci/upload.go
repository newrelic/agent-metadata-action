package oci

import (
	"agent-metadata-action/internal/models"
	"context"
)

type ArtifactUploader interface {
	UploadArtifact(ctx context.Context, artifact *models.ArtifactDefinition, artifactPath, version string) (digest string, size int64, err error)
}

func UploadArtifacts(ctx context.Context, client ArtifactUploader, config *models.OCIConfig, workspacePath, version string) []models.ArtifactUploadResult {
	results := make([]models.ArtifactUploadResult, 0, len(config.Artifacts))

	for _, artifact := range config.Artifacts {
		result := models.ArtifactUploadResult{
			Name:     artifact.Name,
			Path:     artifact.Path,
			OS:       artifact.OS,
			Arch:     artifact.Arch,
			Format:   artifact.Format,
			Uploaded: false,
		}

		fullPath, err := ResolveArtifactPath(workspacePath, artifact.Path)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		digest, size, err := client.UploadArtifact(ctx, &artifact, fullPath, version)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Digest = digest
			result.Size = size
			result.Uploaded = true
		}

		results = append(results, result)
	}

	return results
}

func HasFailures(results []models.ArtifactUploadResult) bool {
	for _, r := range results {
		if !r.Uploaded {
			return true
		}
	}
	return false
}
