package config

import (
	"fmt"
	"os"

	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/parser"
)

// LoadMetadataForAgents loads metadata with only version populated
func LoadMetadataForAgents(version string) models.Metadata {
	return models.Metadata{
		Version: version,
	}
}

// LoadMetadataForDocs loads metadata from changed MDX files in a PR
func LoadMetadataForDocs() ([]models.Metadata, error) {

	var metadataArray []models.Metadata

	// Get changed MDX files (for PR context)
	changedFilepaths, err := github.GetChangedMDXFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "::debug::Could not get changed files: %v\n", err)
	} else if len(changedFilepaths) > 0 {
		for _, filepath := range changedFilepaths {
			frontMatter, err := parser.ParseMDXFile(filepath)
			if err != nil {
				// @todo should we try to load what we can or fail if we hit one bad file?
				return nil, fmt.Errorf("failed to parse MDX file %s: %w", filepath, err)
			}

			// Validate version is not blank
			if frontMatter.Version == "" {
				return nil, fmt.Errorf("version is required in metadata for file %s", filepath)
			}

			metadataArray = append(metadataArray, models.Metadata{
				Version:                   frontMatter.Version,
				Features:                  frontMatter.Features,
				Bugs:                      frontMatter.Bugs,
				Security:                  frontMatter.Security,
				Deprecations:              frontMatter.Deprecations,
				SupportedOperatingSystems: frontMatter.SupportedOperatingSystems,
				EOL:                       frontMatter.EOL,
			})
		}

		fmt.Fprintf(os.Stderr, "::notice::Loaded metadata from %d changed MDX files\n", len(changedFilepaths))
	}
	return metadataArray, nil
}
