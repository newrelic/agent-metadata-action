package models

import (
	"fmt"
	"strings"
)

// SetReleaseChannelRequest represents a request to promote an agent version to a release channel.
type SetReleaseChannelRequest struct {
	Platform        string `json:"platform"`
	OperatingSystem string `json:"operatingSystem,omitempty"`
	Channel         string `json:"channel"`
	Note            string `json:"note,omitempty"`
}

// Validate checks that all required fields are present and hold a recognized value.
func (r *SetReleaseChannelRequest) Validate() error {
	if r.Platform == "" {
		return fmt.Errorf("platform is required")
	}
	if !strings.EqualFold(r.Platform, "HOST") && !strings.EqualFold(r.Platform, "KUBERNETESCLUSTER") {
		return fmt.Errorf("invalid platform '%s': must be 'HOST' or 'KUBERNETESCLUSTER'", r.Platform)
	}

	if strings.EqualFold(r.Platform, "HOST") {
		if r.OperatingSystem == "" {
			return fmt.Errorf("operatingSystem is required when platform is 'HOST'")
		}
		if !strings.EqualFold(r.OperatingSystem, "LINUX") && !strings.EqualFold(r.OperatingSystem, "WINDOWS") {
			return fmt.Errorf("invalid operatingSystem '%s': must be 'LINUX' or 'WINDOWS'", r.OperatingSystem)
		}
	} else if r.OperatingSystem != "" {
		return fmt.Errorf("operatingSystem must not be set when platform is '%s'", r.Platform)
	}

	if r.Channel == "" {
		return fmt.Errorf("channel is required")
	}
	// REGULAR is the only channel defined at MVP; extend this check as new channels ship.
	if !strings.EqualFold(r.Channel, "REGULAR") {
		return fmt.Errorf("invalid channel '%s': must be 'REGULAR'", r.Channel)
	}

	return nil
}

// ReleaseChannelPromotion is the instrumentation service's response after promoting a
// version onto a release channel.
type ReleaseChannelPromotion struct {
	AgentType       string  `json:"agentType"`
	Platform        string  `json:"platform"`
	OperatingSystem string  `json:"operatingSystem,omitempty"`
	Channel         string  `json:"channel"`
	CurrentVersion  string  `json:"currentVersion"`
	PreviousVersion *string `json:"previousVersion,omitempty"`
	PromotedAt      string  `json:"promotedAt"`
	PromotedBy      string  `json:"promotedBy"`
}
