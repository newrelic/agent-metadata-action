package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMDXFile_Success(t *testing.T) {
	// Create a temporary MDX file with valid frontmatter
	tmpDir := t.TempDir()
	mdxFile := filepath.Join(tmpDir, "test.mdx")
	content := `---
subject: Test Agent
releaseDate: '2024-01-01'
version: 1.0.0
features: ["Feature 1", "Feature 2"]
bugs: ["Bug fix 1"]
security: ["CVE-2024-1234"]
deprecations: ["Deprecated feature"]
supportedOperatingSystems: ["linux", "windows"]
eol: '2025-12-31'
---

# Test Release Notes

This is the content.
`
	err := os.WriteFile(mdxFile, []byte(content), 0644)
	require.NoError(t, err)

	// Parse the file
	frontmatter, err := ParseMDXFile(mdxFile)
	require.NoError(t, err)
	assert.NotNil(t, frontmatter)

	// Verify all fields
	assert.Equal(t, "Test Agent", frontmatter.Subject)
	assert.Equal(t, "2024-01-01", frontmatter.ReleaseDate)
	assert.Equal(t, "1.0.0", frontmatter.Version)
	assert.Equal(t, []string{"Feature 1", "Feature 2"}, frontmatter.Features)
	assert.Equal(t, []string{"Bug fix 1"}, frontmatter.Bugs)
	assert.Equal(t, []string{"CVE-2024-1234"}, frontmatter.Security)
	assert.Equal(t, []string{"Deprecated feature"}, frontmatter.Deprecations)
	assert.Equal(t, []string{"linux", "windows"}, frontmatter.SupportedOperatingSystems)
	assert.Equal(t, "2025-12-31", frontmatter.EOL)
}

func TestParseMDXFile_MinimalFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	mdxFile := filepath.Join(tmpDir, "minimal.mdx")
	content := `---
subject: Minimal Agent
releaseDate: '2024-01-01'
version: 1.0.0
---

Content here.
`
	err := os.WriteFile(mdxFile, []byte(content), 0644)
	require.NoError(t, err)

	frontmatter, err := ParseMDXFile(mdxFile)
	require.NoError(t, err)
	assert.NotNil(t, frontmatter)

	assert.Equal(t, "Minimal Agent", frontmatter.Subject)
	assert.Equal(t, "2024-01-01", frontmatter.ReleaseDate)
	assert.Equal(t, "1.0.0", frontmatter.Version)
	assert.Nil(t, frontmatter.Features)
	assert.Nil(t, frontmatter.Bugs)
	assert.Empty(t, frontmatter.EOL)
}

func TestParseMDXFile_NoFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	mdxFile := filepath.Join(tmpDir, "no-frontmatter.mdx")
	content := `# Just Content

No frontmatter here.
`
	err := os.WriteFile(mdxFile, []byte(content), 0644)
	require.NoError(t, err)

	_, err = ParseMDXFile(mdxFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not start with frontmatter delimiter")
}

func TestParseMDXFile_MissingClosingDelimiter(t *testing.T) {
	tmpDir := t.TempDir()
	mdxFile := filepath.Join(tmpDir, "unclosed.mdx")
	content := `---
subject: Test
version: 1.0.0

Content without closing delimiter
`
	err := os.WriteFile(mdxFile, []byte(content), 0644)
	require.NoError(t, err)

	_, err = ParseMDXFile(mdxFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing closing frontmatter delimiter")
}

func TestParseMDXFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	mdxFile := filepath.Join(tmpDir, "invalid-yaml.mdx")
	content := `---
subject: Test
version: [invalid yaml structure
---

Content
`
	err := os.WriteFile(mdxFile, []byte(content), 0644)
	require.NoError(t, err)

	_, err = ParseMDXFile(mdxFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML frontmatter")
}

func TestParseMDXFile_FileNotFound(t *testing.T) {
	_, err := ParseMDXFile("/nonexistent/file.mdx")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read MDX file")
}
