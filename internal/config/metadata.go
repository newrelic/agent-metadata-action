package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"agent-metadata-action/internal/models"
)

// semverPattern validates strict semantic versioning format: MAJOR.MINOR.PATCH (e.g., 1.2.3)
// Does not allow prerelease identifiers or build metadata
var semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

// LoadMetadata loads metadata from INPUT_* environment variables
func LoadMetadata() (models.Metadata, error) {
	version, err := LoadVersion()
	if err != nil {
		return models.Metadata{}, err
	}

	features := parseCommaSeparated(os.Getenv("INPUT_FEATURES"))
	bugs := parseCommaSeparated(os.Getenv("INPUT_BUGS"))
	security := parseCommaSeparated(os.Getenv("INPUT_SECURITY"))

	return models.Metadata{
		Version:  version,
		Features: features,
		Bugs:     bugs,
		Security: security,
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

// parseCommaSeparated parses a comma-separated string into a slice
// Empty strings and whitespace are trimmed
func parseCommaSeparated(value string) []string {
	if value == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
