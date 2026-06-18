package models

// AgentMetadata represents the complete agent metadata structure
type AgentMetadata struct {
	ConfigurationDefinitions []ConfigurationDefinition `json:"configurationDefinitions"`
	Metadata                 Metadata                  `json:"metadata"`
	AgentControlDefinitions  []AgentControlDefinition  `json:"agentControlDefinitions"`
	Bindings                 []interface{}             `json:"bindings,omitempty"`
	BreakingChange           *string                   `json:"breakingChange,omitempty"`
}

// ConfigurationDefinition represents a configuration that can be read from YAML and sent as JSON.
// It uses a map to allow any attributes to be added or removed without code changes.
// YAML fields are automatically translated to JSON.
type ConfigurationDefinition map[string]interface{}

// Metadata represents version and changelog information.
// It uses a map to allow any attributes to be added or removed without code changes.
// YAML/JSON fields are automatically translated.
type Metadata map[string]interface{}

// AgentControlDefinition represents agent control configuration that can be read from YAML and sent as JSON.
// It uses a map to allow any attributes to be added or removed without code changes.
// YAML fields are automatically translated to JSON.
type AgentControlDefinition map[string]interface{}

// AgentDefinition represents agent-level bindings and constraints declared in agentDefinition.yml.
// It uses a map to allow any attributes (bindings, breakingChange, etc.) to pass through without code changes.
type AgentDefinition map[string]interface{}

// ConfigFile represents the YAML file structure containing multiple configs
type ConfigFile struct {
	Configs []ConfigurationDefinition `yaml:"configurationDefinitions"`
}

// AgentControlFile represents the YAML file structure containing multiple agent control definitions
type AgentControlFile struct {
	AgentControls []AgentControlDefinition `yaml:"agentControlDefinitions"`
}
