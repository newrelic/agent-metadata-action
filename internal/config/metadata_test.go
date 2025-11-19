package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadVersion_FromInput(t *testing.T) {
	// Set INPUT_VERSION (mimics GitHub Actions input)
	t.Setenv("INPUT_VERSION", "1.2.3")

	version, err := LoadVersion()
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", version)
}

func TestLoadVersion_ValidFormat_Simple(t *testing.T) {
	t.Setenv("INPUT_VERSION", "1.0.0")

	version, err := LoadVersion()
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version)
}

func TestLoadVersion_ValidFormat_LargeNumbers(t *testing.T) {
	t.Setenv("INPUT_VERSION", "100.200.300")

	version, err := LoadVersion()
	require.NoError(t, err)
	assert.Equal(t, "100.200.300", version)
}

func TestLoadVersion_InvalidFormat_WithV(t *testing.T) {
	t.Setenv("INPUT_VERSION", "v1.2.3")

	_, err := LoadVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version format")
}

func TestLoadVersion_InvalidFormat_Prerelease(t *testing.T) {
	t.Setenv("INPUT_VERSION", "1.2.3-alpha")

	_, err := LoadVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version format")
}

func TestLoadVersion_InvalidFormat_BuildMetadata(t *testing.T) {
	t.Setenv("INPUT_VERSION", "1.2.3+build")

	_, err := LoadVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version format")
}

func TestLoadVersion_InvalidFormat_TwoComponents(t *testing.T) {
	t.Setenv("INPUT_VERSION", "1.2")

	_, err := LoadVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version format")
}

func TestLoadVersion_InvalidFormat_FourComponents(t *testing.T) {
	t.Setenv("INPUT_VERSION", "1.2.3.4")

	_, err := LoadVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version format")
}

func TestLoadVersion_InvalidFormat_LeadingZero(t *testing.T) {
	t.Setenv("INPUT_VERSION", "01.2.3")

	_, err := LoadVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version format")
}

func TestLoadVersion_InvalidFormat_NonNumeric(t *testing.T) {
	t.Setenv("INPUT_VERSION", "abc")

	_, err := LoadVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version format")
}

func TestLoadVersion_NotSet_Error(t *testing.T) {
	t.Setenv("INPUT_VERSION", "")

	_, err := LoadVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to determine version")
	assert.Contains(t, err.Error(), "INPUT_VERSION not set")
}

func TestParseCommaSeparated_Empty(t *testing.T) {
	result := parseCommaSeparated("")
	assert.Empty(t, result)
}

func TestParseCommaSeparated_Single(t *testing.T) {
	result := parseCommaSeparated("feature1")
	assert.Equal(t, []string{"feature1"}, result)
}

func TestParseCommaSeparated_Multiple(t *testing.T) {
	result := parseCommaSeparated("feature1,feature2,feature3")
	assert.Equal(t, []string{"feature1", "feature2", "feature3"}, result)
}

func TestParseCommaSeparated_WithSpaces(t *testing.T) {
	result := parseCommaSeparated("feature1, feature2 , feature3")
	assert.Equal(t, []string{"feature1", "feature2", "feature3"}, result)
}

func TestParseCommaSeparated_WithEmptyElements(t *testing.T) {
	result := parseCommaSeparated("feature1,,feature2,")
	assert.Equal(t, []string{"feature1", "feature2"}, result)
}

func TestLoadMetadata_AllInputs(t *testing.T) {
	t.Setenv("INPUT_VERSION", "1.2.3")
	t.Setenv("INPUT_FEATURES", "feature1,feature2")
	t.Setenv("INPUT_BUGS", "bug1,bug2,bug3")
	t.Setenv("INPUT_SECURITY", "CVE-2024-1234")
	t.Setenv("INPUT_DEPRECATIONS", "deprecated1,deprecated2")
	t.Setenv("INPUT_SUPPORTEDOPERATINGSYSTEMS", "linux,windows,darwin")
	t.Setenv("INPUT_EOL", "2025-12-31")

	metadata, err := LoadMetadata()
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", metadata.Version)
	assert.Equal(t, []string{"feature1", "feature2"}, metadata.Features)
	assert.Equal(t, []string{"bug1", "bug2", "bug3"}, metadata.Bugs)
	assert.Equal(t, []string{"CVE-2024-1234"}, metadata.Security)
	assert.Equal(t, []string{"deprecated1", "deprecated2"}, metadata.Deprecations)
	assert.Equal(t, []string{"linux", "windows", "darwin"}, metadata.SupportedOperatingSystems)
	assert.Equal(t, "2025-12-31", metadata.EOL)
}

func TestLoadMetadata_VersionOnly(t *testing.T) {
	t.Setenv("INPUT_VERSION", "2.0.0")
	t.Setenv("INPUT_FEATURES", "")
	t.Setenv("INPUT_BUGS", "")
	t.Setenv("INPUT_SECURITY", "")
	t.Setenv("INPUT_DEPRECATIONS", "")
	t.Setenv("INPUT_SUPPORTEDOPERATINGSYSTEMS", "")
	t.Setenv("INPUT_EOL", "")

	metadata, err := LoadMetadata()
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", metadata.Version)
	assert.Empty(t, metadata.Features)
	assert.Empty(t, metadata.Bugs)
	assert.Empty(t, metadata.Security)
	assert.Empty(t, metadata.Deprecations)
	assert.Empty(t, metadata.SupportedOperatingSystems)
	assert.Empty(t, metadata.EOL)
}

func TestLoadMetadata_NoVersion_Error(t *testing.T) {
	t.Setenv("INPUT_VERSION", "")
	t.Setenv("INPUT_FEATURES", "feature1")

	_, err := LoadMetadata()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to determine version")
}
