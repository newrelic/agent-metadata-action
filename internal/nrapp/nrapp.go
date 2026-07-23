// Package nrapp builds the New Relic Go agent application used to monitor this
// action. It is shared so that both the main run and the failure reporter
// report under the same APM app.
package nrapp

import (
	"context"
	"time"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/logging"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// AppName is the APM application name this action reports under. Failure
// reports use the same name so they land in the same app as normal telemetry.
const AppName = "agent-metadata-action"

// New creates the New Relic application used to monitor this action.
// Returns nil if APM_CONTROL_NR_LICENSE_KEY is not set (silent no-op mode).
func New(ctx context.Context) *newrelic.Application {
	licenseKey := config.GetNRAgentLicenseKey()
	if licenseKey == "" {
		logging.Warn(ctx, "Failed to init New Relic - missing license key")
		return nil
	}

	// Hardcode staging environment
	if err := config.SetNRAgentHost(); err != nil {
		logging.Warnf(ctx, "Failed to init New Relic, missing host: %v", err)
		return nil
	}
	logging.Notice(ctx, "Using New Relic staging environment")

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(AppName),
		newrelic.ConfigLicense(licenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigFromEnvironment(), // This reads NEW_RELIC_HOST
		newrelic.ConfigLabels(map[string]string{
			"team": "APM Control Team",
		}),
	)
	if err != nil {
		logging.Warnf(ctx, "Failed to init New Relic: %v", err)
		return nil
	}

	logging.Notice(ctx, "New Relic APM enabled - waiting for connection...")

	// Wait for the app to connect (max 10 seconds)
	if err := app.WaitForConnection(10 * time.Second); err != nil {
		logging.Warnf(ctx, "New Relic connection timeout: %v - will try to send data anyway", err)
	} else {
		logging.Notice(ctx, "New Relic connected successfully")
	}

	return app
}
