package github

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetChangedMDXFiles(t *testing.T) {
	// Get actual git SHAs for testing
	baseCmd := exec.Command("git", "rev-parse", "main")
	baseOut, err := baseCmd.Output()
	if err != nil {
		t.Skip("Skipping test: not in a git repository or main branch doesn't exist")
	}
	baseSHA := string(baseOut[:len(baseOut)-1]) // Remove trailing newline

	headCmd := exec.Command("git", "rev-parse", "HEAD")
	headOut, err := headCmd.Output()
	if err != nil {
		t.Skip("Skipping test: not in a git repository")
	}
	headSHA := string(headOut[:len(headOut)-1])

	// Check if there are any committed changes between main and HEAD
	diffCmd := exec.Command("git", "diff", "--name-only", fmt.Sprintf("%s...%s", baseSHA, headSHA))
	diffOut, err := diffCmd.Output()
	if err != nil {
		t.Fatalf("Failed to check for changes: %v", err)
	}
	if len(strings.TrimSpace(string(diffOut))) == 0 {
		t.Skip("Skipping test: no committed changes between main and HEAD (commit your changes to test)")
	}

	// Create mock event payload
	event := PREvent{}
	event.PullRequest.Base.SHA = baseSHA
	event.PullRequest.Head.SHA = headSHA

	eventData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(tmpFile, eventData, 0644); err != nil {
		t.Fatalf("Failed to write event file: %v", err)
	}

	// Set environment variable
	oldEventPath := os.Getenv("GITHUB_EVENT_PATH")
	os.Setenv("GITHUB_EVENT_PATH", tmpFile)
	defer func() {
		if oldEventPath != "" {
			os.Setenv("GITHUB_EVENT_PATH", oldEventPath)
		} else {
			os.Unsetenv("GITHUB_EVENT_PATH")
		}
	}()

	// Run the function
	files, err := GetChangedMDXFiles()
	if err != nil {
		t.Fatalf("GetChangedMDXFiles failed: %v", err)
	}

	// Verify results
	t.Logf("Found %d changed RELEASE_NOTES_FILE_EXTENSION files in ROOT_RELEASE_NOTES_DIR (excluding IGNORED_FILENAMES)", len(files))
	for _, file := range files {
		t.Logf("  - %s", file)

		// Verify it's under ROOT_RELEASE_NOTES_DIR
		if !strings.Contains(file, ROOT_RELEASE_NOTES_DIR) {
			t.Errorf("File %s is not under %s", file, ROOT_RELEASE_NOTES_DIR)
		}

		// Verify it's an .mdx file
		if filepath.Ext(file) != RELEASE_NOTES_FILE_EXTENSION {
			t.Errorf("File %s is not a %s file", file, RELEASE_NOTES_FILE_EXTENSION)
		}

		// Verify it's not in the ignored list
		if isIgnoredFilename(filepath.Base(file)) {
			t.Errorf("File %s is in IGNORED_FILENAMES but should be excluded", file)
		}
	}
}

func TestGetChangedMDXFiles_NoEventPath(t *testing.T) {
	oldEventPath := os.Getenv("GITHUB_EVENT_PATH")
	os.Unsetenv("GITHUB_EVENT_PATH")
	defer func() {
		if oldEventPath != "" {
			os.Setenv("GITHUB_EVENT_PATH", oldEventPath)
		}
	}()

	_, err := GetChangedMDXFiles()
	if err == nil {
		t.Fatal("Expected error when GITHUB_EVENT_PATH not set")
	}
	if !strings.Contains(err.Error(), "GITHUB_EVENT_PATH not set") {
		t.Errorf("Expected error about GITHUB_EVENT_PATH, got: %v", err)
	}
}

func TestIsIgnoredFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "index.mdx is ignored",
			filename: "index.mdx",
			expected: true,
		},
		{
			name:     "regular mdx file is not ignored",
			filename: "java-agent-130.mdx",
			expected: false,
		},
		{
			name:     "path with my-file.mdx in wrong directory",
			filename: "some/path/to/my-file.mdx",
			expected: false, // full path, not just the base
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIgnoredFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("isIgnoredFilename(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}
