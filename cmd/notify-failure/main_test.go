package main

import "testing"

func TestFailureTransactionName(t *testing.T) {
	t.Setenv("GITHUB_WORKFLOW", "Go Tests")
	t.Setenv("GITHUB_JOB", "tests")

	if got, want := failureTransactionName(), "Action/Go Tests/tests"; got != want {
		t.Fatalf("failureTransactionName() = %q, want %q", got, want)
	}
}

func TestFailureError(t *testing.T) {
	t.Setenv("GITHUB_WORKFLOW", "Go Tests")
	t.Setenv("GITHUB_JOB", "tests")

	want := `GitHub Actions workflow "Go Tests" job "tests" failed`
	if got := failureError().Error(); got != want {
		t.Fatalf("failureError() = %q, want %q", got, want)
	}
}

func TestFailureAttributes(t *testing.T) {
	env := map[string]string{
		"GITHUB_WORKFLOW":   "Go Tests",
		"GITHUB_JOB":        "tests",
		"GITHUB_RUN_ID":     "123",
		"GITHUB_REPOSITORY": "newrelic/agent-metadata-action",
		"GITHUB_REF":        "refs/heads/main",
		"GITHUB_SHA":        "abc123",
	}
	for k, v := range env {
		t.Setenv(k, v)
	}

	attrs := failureAttributes()
	want := map[string]string{
		"github.workflow":   "Go Tests",
		"github.job":        "tests",
		"github.run_id":     "123",
		"github.repository": "newrelic/agent-metadata-action",
		"github.ref":        "refs/heads/main",
		"github.sha":        "abc123",
	}
	for key, wantVal := range want {
		if got := attrs[key]; got != wantVal {
			t.Errorf("attrs[%q] = %v, want %q", key, got, wantVal)
		}
	}
}
