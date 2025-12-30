package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/fileutil"
)

const ROOT_RELEASE_NOTES_DIR = "src/content/docs/release-notes"
const RELEASE_NOTES_FILE_EXTENSION = ".mdx"

var IGNORED_FILENAMES = []string{"index.mdx"}

// gitSHARegex validates Git SHA-1 hashes (40 hexadecimal characters)
var gitSHARegex = regexp.MustCompile(`^[0-9a-f]{40}$`)

// PREvent represents the GitHub PR event payload
type PREvent struct {
	PullRequest struct {
		Base struct {
			SHA string `json:"sha"`
			REF string `json:"ref"`
		} `json:"base"`
		Head struct {
			SHA string `json:"sha"`
			REF string `json:"ref"`
		} `json:"head"`
	} `json:"pull_request"`
}

// GetChangedMDXFiles returns RELEASE_NOTES_FILE_EXTENSION type files changed in the PR under ROOT_RELEASE_NOTES_DIR, excluding IGNORED_FILENAMES
func GetChangedMDXFiles() ([]string, error) {
	return GetChangedMDXFilesFunc()
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

// isValidGitSHA validates that a string is a valid Git SHA-1 hash
// Git SHA-1 hashes are exactly 40 hexadecimal characters
func isValidGitSHA(sha string) bool {
	return gitSHARegex.MatchString(sha)
}

// GetChangedMDXFilesFunc is a variable that holds the function to get changed MDX files
// This allows tests to override the implementation
var GetChangedMDXFilesFunc = getChangedMDXFilesImpl

// getChangedMDXFilesImpl is the actual implementation
func getChangedMDXFilesImpl() ([]string, error) {
	eventPath := config.GetEventPath()
	if eventPath == "" {
		return nil, fmt.Errorf("GITHUB_EVENT_PATH not set")
	}
	fmt.Printf("::debug::GH event path: %s", eventPath)

	data, err := fileutil.ReadFileSafe(eventPath, fileutil.MaxConfigFileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read event payload: %w", err)
	}

	var event PREvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event payload: %w", err)
	}

	fmt.Printf("::debug::GH branch names: base %s and head %s", event.PullRequest.Base.REF, event.PullRequest.Head.REF)
	fmt.Printf("::debug::GH SHAs: base %s and head %s", event.PullRequest.Base.SHA, event.PullRequest.Head.SHA)

	// Validate SHAs to prevent command injection
	/*
	   if !isValidGitSHA(event.PullRequest.Base.SHA) {
	   		return nil, fmt.Errorf("invalid base SHA format: must be 40 hexadecimal characters")
	   	}
	   	if !isValidGitSHA(event.PullRequest.Head.SHA) {
	   		return nil, fmt.Errorf("invalid head SHA format: must be 40 hexadecimal characters")
	   	}
	*/
	cmd := exec.Command("git", "diff", "--diff-filter=ACMR", "--name-only",
		fmt.Sprintf("%s...%s", event.PullRequest.Base.SHA, event.PullRequest.Head.SHA))

	// Set working directory to GITHUB_WORKSPACE so git can find the repository
	if workspace := config.GetWorkspace(); workspace != "" {
		cmd.Dir = workspace
	}

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	workspace := config.GetWorkspace()
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
			// Convert to absolute path if workspace is set
			if workspace != "" {
				line = filepath.Join(workspace, line)
			}
			mdxFiles = append(mdxFiles, line)
		}
	}

	return mdxFiles, nil
}
