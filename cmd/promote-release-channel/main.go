package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"agent-metadata-action/internal/client"
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/nrapp"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// promotionClient interface for testing
type promotionClient interface {
	PromoteToReleaseChannel(ctx context.Context, agentType string, agentVersion string, req *models.SetReleaseChannelRequest) (*models.ReleaseChannelPromotion, error)
}

// createPromotionClientFunc is a variable that holds the function to create a promotion client
// This allows tests to override the implementation
var createPromotionClientFunc = func(baseURL, token string) promotionClient {
	return client.NewInstrumentationClient(baseURL, token)
}

func main() {
	ctx := context.Background()

	nrApp := nrapp.New(ctx)

	err := run(nrApp)

	if nrApp != nil {
		logging.Notice(ctx, "Shutting down New Relic - waiting up to 15 seconds to send data...")
		nrApp.Shutdown(15 * time.Second)
		logging.Notice(ctx, "New Relic shutdown complete")
	}

	if err != nil {
		logging.Noticef(ctx, "%v", err)
		os.Exit(1)
	}
}

func run(nrApp *newrelic.Application) error {
	ctx := context.Background()

	if nrApp != nil {
		txn := nrApp.StartTransaction("promote-release-channel")
		defer txn.End()

		ctx = newrelic.NewContext(ctx, txn)
		logging.Debug(ctx, "New Relic transaction started")
		defer logging.Debug(ctx, "New Relic transaction ended")
	}

	token := config.GetToken()
	if token == "" {
		return fmt.Errorf("NEWRELIC_TOKEN is required but not set")
	}

	agentType := config.GetAgentType()
	if agentType == "" {
		return fmt.Errorf("agent-type is required but not set")
	}

	agentVersion := config.GetVersion()
	if agentVersion == "" {
		return fmt.Errorf("version is required but not set")
	}

	req := &models.SetReleaseChannelRequest{
		Platform:        config.GetPlatform(),
		OperatingSystem: config.GetOperatingSystem(),
		Channel:         config.GetChannel(),
		Note:            config.GetNote(),
	}
	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid release channel promotion request: %w", err)
	}

	promClient := createPromotionClientFunc(config.GetMetadataURL(), token)

	logging.Noticef(ctx, "Promoting %s version %s to release channel %s (platform=%s, os=%s)",
		agentType, agentVersion, req.Channel, req.Platform, req.OperatingSystem)

	promotion, err := promClient.PromoteToReleaseChannel(ctx, agentType, agentVersion, req)
	if err != nil {
		return fmt.Errorf("failed to promote %s version %s to %s: %w", agentType, agentVersion, req.Channel, err)
	}

	logging.PrintJSON(ctx, "Release Channel Promotion", promotion)

	if promotion.PreviousVersion != nil {
		logging.Noticef(ctx, "Promoted %s %s to channel %s (previous: %s)", agentType, agentVersion, req.Channel, *promotion.PreviousVersion)
	} else {
		logging.Noticef(ctx, "Promoted %s %s to channel %s (first promotion for platform=%s os=%s)",
			agentType, agentVersion, req.Channel, req.Platform, req.OperatingSystem)
	}

	return nil
}
