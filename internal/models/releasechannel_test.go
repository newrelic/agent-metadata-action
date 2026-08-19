package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetReleaseChannelRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		request     SetReleaseChannelRequest
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid HOST + LINUX",
			request: SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "REGULAR"},
			wantErr: false,
		},
		{
			name:    "valid HOST + WINDOWS",
			request: SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "WINDOWS", Channel: "REGULAR"},
			wantErr: false,
		},
		{
			name:    "valid KUBERNETESCLUSTER without operating system",
			request: SetReleaseChannelRequest{Platform: "KUBERNETESCLUSTER", Channel: "REGULAR"},
			wantErr: false,
		},
		{
			name:    "platform is case-insensitive",
			request: SetReleaseChannelRequest{Platform: "host", OperatingSystem: "linux", Channel: "regular"},
			wantErr: false,
		},
		{
			name:        "missing platform",
			request:     SetReleaseChannelRequest{Channel: "REGULAR"},
			wantErr:     true,
			errContains: "platform is required",
		},
		{
			name:        "invalid platform",
			request:     SetReleaseChannelRequest{Platform: "BAREMETAL", OperatingSystem: "LINUX", Channel: "REGULAR"},
			wantErr:     true,
			errContains: "invalid platform",
		},
		{
			name:        "missing operating system when platform is HOST",
			request:     SetReleaseChannelRequest{Platform: "HOST", Channel: "REGULAR"},
			wantErr:     true,
			errContains: "operatingSystem is required",
		},
		{
			name:        "invalid operating system when platform is HOST",
			request:     SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "MACOS", Channel: "REGULAR"},
			wantErr:     true,
			errContains: "invalid operatingSystem",
		},
		{
			name:        "operating system set when platform is KUBERNETESCLUSTER",
			request:     SetReleaseChannelRequest{Platform: "KUBERNETESCLUSTER", OperatingSystem: "LINUX", Channel: "REGULAR"},
			wantErr:     true,
			errContains: "must not be set",
		},
		{
			name:        "missing channel",
			request:     SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX"},
			wantErr:     true,
			errContains: "channel is required",
		},
		{
			name:        "invalid channel",
			request:     SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "BETA"},
			wantErr:     true,
			errContains: "invalid channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
