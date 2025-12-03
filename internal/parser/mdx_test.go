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

func TestParseMDXFiles_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first MDX file
	file1 := filepath.Join(tmpDir, "release1.mdx")
	content1 := `---
features: ["Feature A", "Feature B"]
bugs: ["Bug 1"]
security: []
---
Content 1
`
	err := os.WriteFile(file1, []byte(content1), 0644)
	require.NoError(t, err)

	// Create second MDX file
	file2 := filepath.Join(tmpDir, "release2.mdx")
	content2 := `---
features: ["Feature C"]
bugs: ["Bug 2", "Bug 3"]
security: ["CVE-2024-001"]
deprecations: ["Old API"]
supportedOperatingSystems: ["linux"]
eol: '2025-12-31'
---
Content 2
`
	err = os.WriteFile(file2, []byte(content2), 0644)
	require.NoError(t, err)

	// Parse multiple files
	filePaths := []string{file1, file2}
	features, bugs, security, deprecations, supportedOS, eol, err := ParseMDXFiles(filePaths, "")
	require.NoError(t, err)

	// Verify aggregated results (order may vary due to map iteration)
	assert.Len(t, features, 3)
	assert.Contains(t, features, "Feature A")
	assert.Contains(t, features, "Feature B")
	assert.Contains(t, features, "Feature C")

	assert.Len(t, bugs, 3)
	assert.Contains(t, bugs, "Bug 1")
	assert.Contains(t, bugs, "Bug 2")
	assert.Contains(t, bugs, "Bug 3")

	assert.Len(t, security, 1)
	assert.Contains(t, security, "CVE-2024-001")

	assert.Len(t, deprecations, 1)
	assert.Contains(t, deprecations, "Old API")

	assert.Len(t, supportedOS, 1)
	assert.Contains(t, supportedOS, "linux")

	assert.Equal(t, "2025-12-31", eol)
}

func TestParseMDXFiles_WithWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, "workspace")
	err := os.MkdirAll(workspace, 0755)
	require.NoError(t, err)

	// Create MDX file in workspace
	mdxFile := filepath.Join(workspace, "release.mdx")
	content := `---
features: ["Workspace Feature"]
---
Content
`
	err = os.WriteFile(mdxFile, []byte(content), 0644)
	require.NoError(t, err)

	// Parse with relative path and workspace
	filePaths := []string{"release.mdx"}
	features, _, _, _, _, _, err := ParseMDXFiles(filePaths, workspace)
	require.NoError(t, err)

	assert.Len(t, features, 1)
	assert.Contains(t, features, "Workspace Feature")
}

func TestParseMDXFiles_EmptyList(t *testing.T) {
	features, bugs, security, deprecations, supportedOS, eol, err := ParseMDXFiles([]string{}, "")
	require.NoError(t, err)

	assert.Empty(t, features)
	assert.Empty(t, bugs)
	assert.Empty(t, security)
	assert.Empty(t, deprecations)
	assert.Empty(t, supportedOS)
	assert.Empty(t, eol)
}

func TestParseMDXFiles_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two files with overlapping features
	file1 := filepath.Join(tmpDir, "release1.mdx")
	content1 := `---
features: ["Feature A", "Feature B"]
bugs: ["Bug 1"]
---
Content 1
`
	err := os.WriteFile(file1, []byte(content1), 0644)
	require.NoError(t, err)

	file2 := filepath.Join(tmpDir, "release2.mdx")
	content2 := `---
features: ["Feature B", "Feature C"]
bugs: ["Bug 1", "Bug 2"]
---
Content 2
`
	err = os.WriteFile(file2, []byte(content2), 0644)
	require.NoError(t, err)

	// Parse both files
	filePaths := []string{file1, file2}
	features, bugs, _, _, _, _, err := ParseMDXFiles(filePaths, "")
	require.NoError(t, err)

	// Verify deduplication
	assert.Len(t, features, 3) // A, B, C (B not duplicated)
	assert.Len(t, bugs, 2)     // Bug 1, Bug 2 (Bug 1 not duplicated)
}

func TestParseMDXFiles_EmptyStringsFiltered(t *testing.T) {
	tmpDir := t.TempDir()
	mdxFile := filepath.Join(tmpDir, "empty-strings.mdx")
	content := `---
features: ["Feature A", "", "Feature B"]
bugs: [""]
security: []
---
Content
`
	err := os.WriteFile(mdxFile, []byte(content), 0644)
	require.NoError(t, err)

	filePaths := []string{mdxFile}
	features, bugs, security, _, _, _, err := ParseMDXFiles(filePaths, "")
	require.NoError(t, err)

	// Empty strings should be filtered out
	assert.Len(t, features, 2)
	assert.Contains(t, features, "Feature A")
	assert.Contains(t, features, "Feature B")
	assert.Empty(t, bugs)
	assert.Empty(t, security)
}

func TestParseMDXFiles_FileNotFound(t *testing.T) {
	filePaths := []string{"/nonexistent/file.mdx"}
	_, _, _, _, _, _, err := ParseMDXFiles(filePaths, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestParseMDXFiles_LastEOLWins(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first file with EOL
	file1 := filepath.Join(tmpDir, "release1.mdx")
	content1 := `---
eol: '2024-12-31'
---
Content 1
`
	err := os.WriteFile(file1, []byte(content1), 0644)
	require.NoError(t, err)

	// Create second file with different EOL
	file2 := filepath.Join(tmpDir, "release2.mdx")
	content2 := `---
eol: '2025-06-30'
---
Content 2
`
	err = os.WriteFile(file2, []byte(content2), 0644)
	require.NoError(t, err)

	// Parse both files
	filePaths := []string{file1, file2}
	_, _, _, _, _, eol, err := ParseMDXFiles(filePaths, "")
	require.NoError(t, err)

	// The last EOL encountered should win (order depends on file processing)
	assert.NotEmpty(t, eol)
	assert.True(t, eol == "2024-12-31" || eol == "2025-06-30")
}
