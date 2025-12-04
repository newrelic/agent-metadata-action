package github

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetChangedMDXFiles(t *testing.T) {
	// Create a temporary workspace with git repository
	workspace := t.TempDir()

	// Initialize git repo
	gitInit := exec.Command("git", "init")
	gitInit.Dir = workspace
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Configure git user
	gitConfig := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfig.Dir = workspace
	if err := gitConfig.Run(); err != nil {
		t.Fatalf("Failed to configure git: %v", err)
	}

	gitConfig = exec.Command("git", "config", "user.name", "Test User")
	gitConfig.Dir = workspace
	if err := gitConfig.Run(); err != nil {
		t.Fatalf("Failed to configure git: %v", err)
	}

	// Create initial commit (without MDX files)
	releaseNotesDir := filepath.Join(workspace, ROOT_RELEASE_NOTES_DIR, "agent-release-notes", "java-release-notes")
	if err := os.MkdirAll(releaseNotesDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	gitCommit := exec.Command("git", "commit", "--allow-empty", "-m", "Initial commit")
	gitCommit.Dir = workspace
	if err := gitCommit.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Get base SHA
	baseSHACmd := exec.Command("git", "rev-parse", "HEAD")
	baseSHACmd.Dir = workspace
	baseSHAOut, err := baseSHACmd.Output()
	if err != nil {
		t.Fatalf("Failed to get base SHA: %v", err)
	}
	baseSHA := strings.TrimSpace(string(baseSHAOut))

	// Create MDX files
	mdxContent := `---
subject: Java Agent
releaseDate: '2024-01-15'
version: 1.3.0
features:
  - New feature
bugs:
  - Bug fix
---

# Release Notes
`

	mdxFile := filepath.Join(releaseNotesDir, "java-agent-130.mdx")
	if err := os.WriteFile(mdxFile, []byte(mdxContent), 0644); err != nil {
		t.Fatalf("Failed to write MDX file: %v", err)
	}

	// Also create an index.mdx that should be ignored
	indexFile := filepath.Join(releaseNotesDir, "index.mdx")
	if err := os.WriteFile(indexFile, []byte("# Index"), 0644); err != nil {
		t.Fatalf("Failed to write index file: %v", err)
	}

	// Add and commit MDX files
	gitAdd := exec.Command("git", "add", ".")
	gitAdd.Dir = workspace
	if err := gitAdd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	gitCommit = exec.Command("git", "commit", "-m", "Add release notes")
	gitCommit.Dir = workspace
	if err := gitCommit.Run(); err != nil {
		t.Fatalf("Failed to commit MDX files: %v", err)
	}

	// Get head SHA
	headSHACmd := exec.Command("git", "rev-parse", "HEAD")
	headSHACmd.Dir = workspace
	headSHAOut, err := headSHACmd.Output()
	if err != nil {
		t.Fatalf("Failed to get head SHA: %v", err)
	}
	headSHA := strings.TrimSpace(string(headSHAOut))

	// Create PR event
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

	// Set environment variables
	t.Setenv("GITHUB_EVENT_PATH", tmpFile)
	t.Setenv("GITHUB_WORKSPACE", workspace)

	// Run the function
	files, err := GetChangedMDXFiles()
	if err != nil {
		t.Fatalf("GetChangedMDXFiles failed: %v", err)
	}

	// Verify results
	if len(files) != 1 {
		t.Errorf("Expected 1 changed MDX file, got %d", len(files))
	}

	if len(files) > 0 {
		file := files[0]
		t.Logf("Found changed file: %s", file)

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

		// Verify it's the java-agent file, not index.mdx
		if !strings.Contains(file, "java-agent-130.mdx") {
			t.Errorf("Expected java-agent-130.mdx, got %s", file)
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
