package fileutil

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFileSafe_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test content"

	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	data, err := ReadFileSafe(testFile, 1024)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestReadFileSafe_FileNotFound(t *testing.T) {
	_, err := ReadFileSafe("/nonexistent/file.txt", 1024)
	assert.Error(t, err)
}

func TestReadFileSafe_ExceedsMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// Create a file that's 2KB
	content := strings.Repeat("a", 2048)
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Try to read with 1KB limit
	_, err = ReadFileSafe(testFile, 1024)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
	assert.Contains(t, err.Error(), "1024 bytes")
	assert.Contains(t, err.Error(), "2048 bytes")
}

func TestReadFileSafe_ExactlyMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "exact.txt")

	// Create a file that's exactly 1KB
	content := strings.Repeat("a", 1024)
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Should succeed when size equals limit
	data, err := ReadFileSafe(testFile, 1024)
	require.NoError(t, err)
	assert.Equal(t, 1024, len(data))
}

func TestReadFileSafe_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(testFile, []byte(""), 0644)
	require.NoError(t, err)

	data, err := ReadFileSafe(testFile, 1024)
	require.NoError(t, err)
	assert.Equal(t, 0, len(data))
}

func TestReadAllSafe_Success(t *testing.T) {
	content := "test content"
	reader := bytes.NewReader([]byte(content))

	data, err := ReadAllSafe(reader, 1024)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestReadAllSafe_ExceedsMaxSize(t *testing.T) {
	// Create a reader with 2KB of data
	content := strings.Repeat("a", 2048)
	reader := bytes.NewReader([]byte(content))

	// Try to read with 1KB limit
	_, err := ReadAllSafe(reader, 1024)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
	assert.Contains(t, err.Error(), "1024 bytes")
}

func TestReadAllSafe_ExactlyMaxSize(t *testing.T) {
	// Create a reader with exactly 1KB of data
	content := strings.Repeat("a", 1024)
	reader := bytes.NewReader([]byte(content))

	// Should succeed when size equals limit
	data, err := ReadAllSafe(reader, 1024)
	require.NoError(t, err)
	assert.Equal(t, 1024, len(data))
}

func TestReadAllSafe_Empty(t *testing.T) {
	reader := bytes.NewReader([]byte(""))

	data, err := ReadAllSafe(reader, 1024)
	require.NoError(t, err)
	assert.Equal(t, 0, len(data))
}

func TestReadAllSafe_LargeFile(t *testing.T) {
	// Create a 100MB reader (would cause DoS without limit)
	largeContent := strings.Repeat("x", 100*1024*1024)
	reader := bytes.NewReader([]byte(largeContent))

	// Should fail with 10MB limit
	_, err := ReadAllSafe(reader, MaxConfigFileSize)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}
