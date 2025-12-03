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
var semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

// LoadMetadata loads metadata from changed MDX files in a PR
func LoadMetadata() (models.Metadata, error) {
	version, err := LoadVersion()
	if err != nil {
		return models.Metadata{}, err
	}

	var features, bugs, security, deprecations, supportedOperatingSystems []string
	var eol string

	// Get changed MDX files (for PR context)
	changedFiles, err := github.GetChangedMDXFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "::debug::Could not get changed files: %v\n", err)
	} else if len(changedFiles) > 0 {
		// Parse MDX files and extract metadata
		workspace := GetWorkspace()
		features, bugs, security, deprecations, supportedOperatingSystems, eol, err = parser.ParseMDXFiles(changedFiles, workspace)
		if err != nil {
			return models.Metadata{}, fmt.Errorf("failed to parse MDX files: %w", err)
		}
		fmt.Fprintf(os.Stderr, "::notice::Loaded metadata from %d changed MDX files\n", len(changedFiles))
	}

	return models.Metadata{
		Version:                   version,
		Features:                  features,
		Bugs:                      bugs,
		Security:                  security,
		Deprecations:              deprecations,
		SupportedOperatingSystems: supportedOperatingSystems,
		EOL:                       eol,
	}, nil
}

// LoadVersion loads the version from INPUT_VERSION
// Returns an error if no version can be determined or if version is not in valid X.Y.Z format
func LoadVersion() (string, error) {
	version := os.Getenv("INPUT_VERSION")
	if version == "" {
		return "", fmt.Errorf("unable to determine version: INPUT_VERSION not set")
	}

	// Validate strict semver format (X.Y.Z only)
	if !semverPattern.MatchString(version) {
		return "", fmt.Errorf("invalid version format: %s (must be X.Y.Z format, e.g., 1.2.3)", version)
	}

	return version, nil
}
