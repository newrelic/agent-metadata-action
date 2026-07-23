package nrapp

import (
	"context"
	"testing"
)

func TestNew_NoLicenseKeyReturnsNil(t *testing.T) {
	t.Setenv("APM_CONTROL_NR_LICENSE_KEY", "")

	if app := New(context.Background()); app != nil {
		t.Fatalf("expected nil application when license key is unset, got %v", app)
	}
}
