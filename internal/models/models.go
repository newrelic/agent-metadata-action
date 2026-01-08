package models

// AgentMetadata represents the complete agent metadata structure
type AgentMetadata struct {
	ConfigurationDefinitions []ConfigurationDefinition `json:"configurationDefinitions"`
	Metadata                 Metadata                  `json:"metadata"`
	AgentControl             []AgentControl            `json:"agentControl"`
}

// ConfigurationDefinition represents a configuration that can be read from YAML and sent as JSON.
// It uses a map to allow any attributes to be added or removed without code changes.
// YAML fields are automatically translated to JSON.
type ConfigurationDefinition map[string]interface{}

// Metadata represents version and changelog information.
// It uses a map to allow any attributes to be added or removed without code changes.
// YAML/JSON fields are automatically translated.
type Metadata map[string]interface{}

// AgentControl represents agent control content for a platform
type AgentControl struct {
	Platform string `json:"platform"`
	Content  string `json:"content"` // base64 encoded
}

// ConfigFile represents the YAML file structure containing multiple configs
type ConfigFile struct {
	Configs []ConfigurationDefinition `yaml:"configurationDefinitions"`
}
