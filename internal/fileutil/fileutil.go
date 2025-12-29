package fileutil

import (
	"fmt"
	"io"
	"os"
)

const (
	// MaxConfigFileSize is the maximum size for configuration files (10MB)
	MaxConfigFileSize = 10 * 1024 * 1024

	// MaxMDXFileSize is the maximum size for MDX files (1MB)
	MaxMDXFileSize = 1 * 1024 * 1024

	// MaxHTTPResponseSize is the maximum size for HTTP response bodies (50MB)
	MaxHTTPResponseSize = 50 * 1024 * 1024
)

// ReadFileSafe reads a file with a size limit to prevent DoS attacks
func ReadFileSafe(path string, maxSize int64) ([]byte, error) {
	// Get file info to check size before reading
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Check file size
	if info.Size() > maxSize {
		return nil, fmt.Errorf("file %s exceeds maximum size of %d bytes (actual: %d bytes)", path, maxSize, info.Size())
	}

	// Read file
	return os.ReadFile(path)
}

// ReadAllSafe reads from an io.Reader with a size limit to prevent DoS attacks
func ReadAllSafe(r io.Reader, maxSize int64) ([]byte, error) {
	// Use io.LimitReader to prevent reading beyond maxSize
	limitedReader := io.LimitReader(r, maxSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	// Check if we hit the limit
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("data exceeds maximum size of %d bytes", maxSize)
	}

	return data, nil
}
