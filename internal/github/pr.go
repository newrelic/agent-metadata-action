package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const ROOT_RELEASE_NOTES_DIR = "src/content/docs/release-notes"
const RELEASE_NOTES_FILE_EXTENSION = ".mdx"

var IGNORED_FILENAMES = []string{"index.mdx"}

// PREvent represents the GitHub PR event payload
type PREvent struct {
	PullRequest struct {
		Base struct {
			SHA string `json:"sha"`
		} `json:"base"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
}

// GetChangedMDXFiles returns RELEASE_NOTES_FILE_EXTENSION type files changed in the PR under ROOT_RELEASE_NOTES_DIR, excluding IGNORED_FILENAMES
func GetChangedMDXFiles() ([]string, error) {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil, fmt.Errorf("GITHUB_EVENT_PATH not set")
	}

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read event payload: %w", err)
	}

	var event PREvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event payload: %w", err)
	}

	cmd := exec.Command("git", "diff", "--diff-filter=ACMR", "--name-only",
		fmt.Sprintf("%s...%s", event.PullRequest.Base.SHA, event.PullRequest.Head.SHA))
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	var mdxFiles []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasSuffix(line, RELEASE_NOTES_FILE_EXTENSION) {
			continue
		}
		if isIgnoredFilename(filepath.Base(line)) {
			continue
		}
		if strings.Contains(line, ROOT_RELEASE_NOTES_DIR) {
			mdxFiles = append(mdxFiles, line)
		}
	}

	return mdxFiles, nil
}

// isIgnoredFilename checks if the filename should be ignored
func isIgnoredFilename(filename string) bool {
	for _, ignored := range IGNORED_FILENAMES {
		if filename == ignored {
			return true
		}
	}
	return false
}
