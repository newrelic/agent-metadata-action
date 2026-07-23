// Command notify-failure reports a GitHub Actions workflow failure to New Relic
// using the real Go agent, under the same APM app as the action's normal run.
// It is invoked by the action's failure() step so that failures occurring
// before the main binary runs (checkout, auth) are still reported.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/nrapp"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	ctx := context.Background()

	app := nrapp.New(ctx)
	if app == nil {
		// No license key or init failed. Don't turn a missing report into noise
		// on a step that only runs because something already failed.
		logging.Warn(ctx, "Skipping New Relic failure report - agent not initialized")
		return
	}

	reportFailure(ctx, app)

	logging.Notice(ctx, "Shutting down New Relic - waiting up to 15 seconds to send data...")
	app.Shutdown(15 * time.Second)
	logging.Notice(ctx, "New Relic shutdown complete")
}

// reportFailure records the workflow failure as a New Relic error on its own transaction.
func reportFailure(ctx context.Context, app *newrelic.Application) {
	txn := app.StartTransaction(failureTransactionName())
	defer txn.End()

	// Add transaction to context for logging
	ctx = newrelic.NewContext(ctx, txn)
	logging.Debug(ctx, "New Relic transaction started")
	defer logging.Debug(ctx, "New Relic transaction ended")

	err := failureError()
	logging.NoticeErrorWithCategory(ctx, err, "GitHubWorkflowError", failureAttributes())
	logging.Errorf(ctx, "Reported workflow failure to New Relic: %v", err)
}

// failureTransactionName names the transaction after the failed workflow and job.
func failureTransactionName() string {
	return fmt.Sprintf("Action/%s/%s", os.Getenv("GITHUB_WORKFLOW"), os.Getenv("GITHUB_JOB"))
}

// failureError describes the failure using the GitHub-provided workflow context.
func failureError() error {
	return fmt.Errorf("GitHub Actions workflow %q job %q failed",
		os.Getenv("GITHUB_WORKFLOW"), os.Getenv("GITHUB_JOB"))
}

// failureAttributes collects GitHub context for grouping and triage in New Relic.
func failureAttributes() map[string]interface{} {
	return map[string]interface{}{
		"github.workflow":   os.Getenv("GITHUB_WORKFLOW"),
		"github.job":        os.Getenv("GITHUB_JOB"),
		"github.run_id":     os.Getenv("GITHUB_RUN_ID"),
		"github.repository": os.Getenv("GITHUB_REPOSITORY"),
		"github.ref":        os.Getenv("GITHUB_REF"),
		"github.sha":        os.Getenv("GITHUB_SHA"),
	}
}
