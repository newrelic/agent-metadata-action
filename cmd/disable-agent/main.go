package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"agent-metadata-action/internal/client"
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/nrapp"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// disableClient interface for testing
type disableClient interface {
	DisableAgentVersion(ctx context.Context, agentType string, agentVersion string) error
}

// createDisableClientFunc is a variable that holds the function to create a disable client
// This allows tests to override the implementation
var createDisableClientFunc = func(baseURL, token string) disableClient {
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
		txn := nrApp.StartTransaction("disable-agent")
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

	disClient := createDisableClientFunc(config.GetMetadataURL(), token)

	logging.Noticef(ctx, "Disabling %s version %s", agentType, agentVersion)

	if err := disClient.DisableAgentVersion(ctx, agentType, agentVersion); err != nil {
		return fmt.Errorf("failed to disable %s version %s: %w", agentType, agentVersion, err)
	}

	logging.Noticef(ctx, "Disabled %s version %s", agentType, agentVersion)

	return nil
}
