package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetChannel(t *testing.T) {
	t.Setenv("INPUT_CHANNEL", "REGULAR")
	assert.Equal(t, "REGULAR", GetChannel())
}

func TestGetPlatform(t *testing.T) {
	t.Setenv("INPUT_PLATFORM", "HOST")
	assert.Equal(t, "HOST", GetPlatform())
}

func TestGetOperatingSystem(t *testing.T) {
	t.Setenv("INPUT_OPERATING_SYSTEM", "LINUX")
	assert.Equal(t, "LINUX", GetOperatingSystem())
}

func TestGetNote(t *testing.T) {
	t.Setenv("INPUT_NOTE", "GA seed")
	assert.Equal(t, "GA seed", GetNote())
}

func TestGetChannel_Unset(t *testing.T) {
	assert.Equal(t, "", GetChannel())
}

func TestGetPlatform_Unset(t *testing.T) {
	assert.Equal(t, "", GetPlatform())
}

func TestGetOperatingSystem_Unset(t *testing.T) {
	assert.Equal(t, "", GetOperatingSystem())
}

func TestGetNote_Unset(t *testing.T) {
	assert.Equal(t, "", GetNote())
}
