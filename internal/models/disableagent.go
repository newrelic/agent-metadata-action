package models

// DisableAgentRequest represents a request to disable a published agent version
// so it is no longer returned to consumers. Once a version is disabled it cannot
// be re-published under the same number.
type DisableAgentRequest struct {
	Metadata DisableAgentMetadata `json:"metadata"`
}

// DisableAgentMetadata carries the enabled flag sent to the instrumentation service.
type DisableAgentMetadata struct {
	Enabled bool `json:"enabled"`
}
