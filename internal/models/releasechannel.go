package models

import (
	"fmt"
	"slices"
	"strings"
)

// validChannels lists the release channels accepted by the instrumentation service.
var validChannels = []string{"EXPERIMENTAL", "RAPID", "REGULAR", "STABLE", "EXTENDED"}

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
	if !slices.ContainsFunc(validChannels, func(c string) bool { return strings.EqualFold(r.Channel, c) }) {
		return fmt.Errorf("invalid channel '%s': must be one of %s", r.Channel, strings.Join(validChannels, ", "))
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
