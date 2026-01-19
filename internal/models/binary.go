package models

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type ArtifactDefinition struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Format string `json:"format"`
}

func (a *ArtifactDefinition) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("name is required for artifact")
	}

	namePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !namePattern.MatchString(a.Name) {
		return fmt.Errorf("invalid artifact name '%s': must contain only alphanumeric characters, hyphens, and underscores", a.Name)
	}

	if a.Path == "" {
		return fmt.Errorf("path is required for artifact '%s'", a.Name)
	}

	if a.OS == "" {
		return fmt.Errorf("os is required for artifact '%s'", a.Name)
	}

	if !strings.EqualFold(a.OS, "any") {
		osPattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
		if !osPattern.MatchString(a.OS) {
			return fmt.Errorf("invalid os '%s' for artifact '%s': must be alphanumeric or 'any'", a.OS, a.Name)
		}
	}

	if a.Arch == "" {
		return fmt.Errorf("arch is required for artifact '%s'", a.Name)
	}

	if !strings.EqualFold(a.Arch, "any") {
		archPattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
		if !archPattern.MatchString(a.Arch) {
			return fmt.Errorf("invalid arch '%s' for artifact '%s': must be alphanumeric or 'any'", a.Arch, a.Name)
		}
	}

	if a.Format == "" {
		return fmt.Errorf("format is required for artifact '%s'", a.Name)
	}

	if !strings.EqualFold(a.Format, "tar") && !strings.EqualFold(a.Format, "tar+gzip") && !strings.EqualFold(a.Format, "zip") {
		return fmt.Errorf("invalid format '%s' for artifact '%s': must be 'tar', 'tar+gzip', or 'zip'", a.Format, a.Name)
	}

	return nil
}

func (a *ArtifactDefinition) GetMediaType() string {
	return fmt.Sprintf("application/vnd.newrelic.agent.v1+%s", a.Format)
}

func (a *ArtifactDefinition) GetArtifactType() string {
	return fmt.Sprintf("application/vnd.newrelic.agent.v1+%s", a.Format)
}

func (a *ArtifactDefinition) GetPlatformString() string {
	return fmt.Sprintf("%s/%s", a.OS, a.Arch)
}

func (a *ArtifactDefinition) GetFilename() string {
	return filepath.Base(a.Path)
}

type OCIConfig struct {
	Registry  string               // OCI registry URL (e.g., docker.io/newrelic/agents)
	Username  string               // Registry username
	Password  string               // Registry password or token
	Artifacts []ArtifactDefinition // Array of artifact definitions
}

func (o *OCIConfig) IsEnabled() bool {
	return o.Registry != ""
}

func (o *OCIConfig) Validate() error {
	if !o.IsEnabled() {
		return nil // OCI upload is optional
	}

	if o.Username == "" && o.Password == "" {
		// This is fine for local registries like localhost:5000
	}

	if len(o.Artifacts) == 0 {
		return fmt.Errorf("binaries input is required when oci-registry is set")
	}

	for i, artifact := range o.Artifacts {
		if err := artifact.Validate(); err != nil {
			return fmt.Errorf("artifact %d validation failed: %w", i, err)
		}
	}

	if err := o.ValidateUniqueNames(); err != nil {
		return err
	}

	return nil
}

func (o *OCIConfig) ValidateUniqueNames() error {
	seen := make(map[string]bool)
	for _, artifact := range o.Artifacts {
		if seen[artifact.Name] {
			return fmt.Errorf("duplicate artifact name: '%s'", artifact.Name)
		}
		seen[artifact.Name] = true
	}
	return nil
}

type ArtifactUploadResult struct {
	Name         string
	Path         string
	OS           string
	Arch         string
	Format       string
	Digest       string
	Size         int64
	Uploaded     bool
	Error        string
	Signed       bool   // Whether artifact was signed
	SigningError string // Signing error if Signed=false
}
