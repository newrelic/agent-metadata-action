package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client represents a GitHub API client
type Client struct {
	client *http.Client
	token  string
}

// Content represents the GitHub API content response
type Content struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

var (
	instance *Client
	once     sync.Once
)

// GetClient returns a singleton GitHub API client instance
func GetClient(token string) *Client {
	once.Do(func() {
		instance = &Client{
			client: &http.Client{
				Timeout: 30 * time.Second,
			},
			token: token,
		}
	})
	return instance
}

// ResetClient resets the singleton instance (useful for testing)
func ResetClient() {
	once = sync.Once{}
	instance = nil
}

// FetchFile fetches a file from a GitHub repository
func (gc *Client) FetchFile(repo, path, branch string) ([]byte, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/contents/%s",
		repo,
		path,
	)

	// Add branch query parameter if specified
	if branch != "" {
		url = fmt.Sprintf("%s?ref=%s", url, branch)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := gc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"GitHub API returned status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var content Content
	if err := json.Unmarshal(body, &content); err != nil {
		return nil, fmt.Errorf(
			"failed to parse GitHub response: %w",
			err,
		)
	}

	if content.Encoding != "base64" {
		return nil, fmt.Errorf(
			"unexpected encoding: %s",
			content.Encoding,
		)
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to decode base64 content: %w",
			err,
		)
	}

	return decoded, nil
}
