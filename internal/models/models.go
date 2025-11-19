package models

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// AgentMetadata represents the complete agent metadata structure
type AgentMetadata struct {
	ConfigurationDefinitions []ConfigurationDefinition `json:"configurationDefinitions"`
	Metadata                 Metadata                  `json:"metadata"`
	AgentControl             []AgentControl            `json:"agentControl"`
}

// ConfigurationDefinition represents a configuration that can be read from YAML and sent as JSON
type ConfigurationDefinition struct {
	Version     string `yaml:"version" json:"version"` // schema version, not agent version
	Platform    string `yaml:"platform" json:"platform"`
	Description string `yaml:"description" json:"description"`
	Type        string `yaml:"type" json:"type"`
	Format      string `yaml:"format" json:"format"`
	Schema      string `yaml:"schema" json:"schema"`
}

// UnmarshalYAML implements custom unmarshaling with validation
func (c *ConfigurationDefinition) UnmarshalYAML(node *yaml.Node) error {
	// Use type alias to avoid infinite recursion when decoding
	type rawConfig ConfigurationDefinition
	var raw rawConfig

	if err := node.Decode(&raw); err != nil {
		return err
	}

	// Build context string for better error messages
	context := ""
	if raw.Type != "" && raw.Version != "" {
		context = fmt.Sprintf("config with type '%s' and version '%s'", raw.Type, raw.Version)
	}

	// Validate all required fields
	for _, check := range []struct {
		value string
		field string
	}{
		{raw.Version, "version"},
		{raw.Platform, "platform"},
		{raw.Description, "description"},
		{raw.Type, "type"},
		{raw.Format, "format"},
		{raw.Schema, "schema"},
	} {
		if err := requireField(check.value, check.field, context); err != nil {
			return err
		}
	}

	// Copy validated values
	*c = ConfigurationDefinition(raw)
	return nil
}

// Metadata represents version and changelog information
type Metadata struct {
	Version                   string   `json:"version"` // agent version
	Features                  []string `json:"features"`
	Bugs                      []string `json:"bugs"`
	Security                  []string `json:"security"`
	Deprecations              []string `json:"deprecations"`
	SupportedOperatingSystems []string `json:"supportedOperatingSystems"`
	EOL                       string   `json:"eol"`
}

// UnmarshalYAML implements custom unmarshaling with validation
func (m *Metadata) UnmarshalYAML(node *yaml.Node) error {
	// Use type alias to avoid infinite recursion when decoding
	type rawMetadata Metadata
	var raw rawMetadata

	if err := node.Decode(&raw); err != nil {
		return err
	}

	// Validate version is required
	if err := requireField(raw.Version, "version", ""); err != nil {
		return err
	}

	// Copy validated values
	*m = Metadata(raw)
	return nil
}

// AgentControl represents agent control content for a platform
type AgentControl struct {
	Platform string `json:"platform"`
	Content  string `json:"content"` // base64 encoded
}

// UnmarshalJSON implements custom unmarshaling with validation for AgentControl
func (a *AgentControl) UnmarshalJSON(data []byte) error {
	// Use type alias to avoid infinite recursion when decoding
	type rawAgentControl AgentControl
	var raw rawAgentControl

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Validate required fields
	if err := requireField(raw.Platform, "platform", "agentControl"); err != nil {
		return err
	}
	if err := requireField(raw.Content, "content", "agentControl"); err != nil {
		return err
	}

	// Copy validated values
	*a = AgentControl(raw)
	return nil
}

// ConfigFile represents the YAML file structure containing multiple configs
type ConfigFile struct {
	Configs []ConfigurationDefinition `yaml:"configurationDefinitions"`
}

// requireField validates that a field is not empty
func requireField(value, fieldName, contextName string) error {
	if value == "" {
		if contextName != "" {
			return fmt.Errorf("%s is required for %s", fieldName, contextName)
		}
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}
