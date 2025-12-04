package config

import (
	"fmt"
	"os"
	"regexp"

	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/parser"
)

// semverPattern validates strict semantic versioning format: MAJOR.MINOR.PATCH (e.g., 1.2.3)
// Does not allow prerelease identifiers or build metadata
// @todo may need to revisit if tags used by teams don't match semver
var semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

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

// LoadVersion loads the version from INPUT_VERSION
// Returns an error if a version is provided and it is not in valid X.Y.Z format
func LoadVersion() (string, error) {
	version := os.Getenv("INPUT_VERSION")

	// Validate strict semver format (X.Y.Z only)
	if version != "" && !semverPattern.MatchString(version) {
		return "", fmt.Errorf("invalid version format: %s (must be X.Y.Z format, e.g., 1.2.3)", version)
	}

	return version, nil
}
