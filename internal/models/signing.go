package models

import (
	"fmt"
	"strings"
)

// SigningRequest represents a request to sign an artifact
type SigningRequest struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Digest     string `json:"digest"`
}

// Validate checks that all required fields are present and valid
func (s *SigningRequest) Validate() error {
	if s.Registry == "" {
		return fmt.Errorf("registry is required")
	}
	if s.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if s.Tag == "" {
		return fmt.Errorf("tag is required")
	}
	if s.Digest == "" {
		return fmt.Errorf("digest is required")
	}
	if !strings.HasPrefix(s.Digest, "sha256:") {
		return fmt.Errorf("digest must be in format sha256:... but got: %s", s.Digest)
	}
	return nil
}

// SigningResult tracks signing operation outcome per artifact
type SigningResult struct {
	Name   string // Artifact name
	Digest string // Artifact digest
	Signed bool   // Whether signing succeeded
	Error  string // Error message if signing failed
}
