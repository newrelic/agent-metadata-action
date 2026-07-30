package loader

import (
	"context"
	"testing"

	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeBindingsOverride(t *testing.T) {
	tests := []struct {
		name        string
		existing    []interface{}
		overrideRaw string
		expectErr   bool
		expected    []interface{}
	}{
		{
			name:        "empty override is a no-op",
			existing:    []interface{}{map[string]interface{}{"type": "REQUIRES", "targetType": "AGENT", "target": "NrInfra", "versions": []interface{}{">=1.0.0"}}},
			overrideRaw: "",
			expected:    []interface{}{map[string]interface{}{"type": "REQUIRES", "targetType": "AGENT", "target": "NrInfra", "versions": []interface{}{">=1.0.0"}}},
		},
		{
			name:        "override appends to a nil existing list",
			existing:    nil,
			overrideRaw: `[{"type":"SUPPORTS","targetType":"SPECIFICATION","target":"protocol_version","versions":["<=1.0"]}]`,
			expected:    []interface{}{map[string]interface{}{"type": "SUPPORTS", "targetType": "SPECIFICATION", "target": "protocol_version", "versions": []interface{}{"<=1.0"}}},
		},
		{
			name:        "override replaces a matching existing binding",
			existing:    []interface{}{map[string]interface{}{"type": "SUPPORTS", "targetType": "SPECIFICATION", "target": "protocol_version", "versions": []interface{}{"<=1.0.0"}}},
			overrideRaw: `[{"type":"SUPPORTS","targetType":"SPECIFICATION","target":"protocol_version","versions":["<=1.1"]}]`,
			expected:    []interface{}{map[string]interface{}{"type": "SUPPORTS", "targetType": "SPECIFICATION", "target": "protocol_version", "versions": []interface{}{"<=1.1"}}},
		},
		{
			name: "override leaves non-matching existing bindings untouched",
			existing: []interface{}{
				map[string]interface{}{"type": "REQUIRES", "targetType": "AGENT", "target": "NrInfra", "versions": []interface{}{">=1.0.0"}},
				map[string]interface{}{"type": "SUPPORTS", "targetType": "SPECIFICATION", "target": "protocol_version", "versions": []interface{}{"<=1.0.0"}},
			},
			overrideRaw: `[{"type":"SUPPORTS","targetType":"SPECIFICATION","target":"protocol_version","versions":["<=1.1"]}]`,
			expected: []interface{}{
				map[string]interface{}{"type": "REQUIRES", "targetType": "AGENT", "target": "NrInfra", "versions": []interface{}{">=1.0.0"}},
				map[string]interface{}{"type": "SUPPORTS", "targetType": "SPECIFICATION", "target": "protocol_version", "versions": []interface{}{"<=1.1"}},
			},
		},
		{
			name:        "multiple override entries are all applied",
			existing:    nil,
			overrideRaw: `[{"type":"SUPPORTS","targetType":"SPECIFICATION","target":"protocol_version","versions":["<=1.0"]},{"type":"REQUIRES","targetType":"RUNTIME","target":"java-runtime","versions":[">=11"]}]`,
			expected: []interface{}{
				map[string]interface{}{"type": "SUPPORTS", "targetType": "SPECIFICATION", "target": "protocol_version", "versions": []interface{}{"<=1.0"}},
				map[string]interface{}{"type": "REQUIRES", "targetType": "RUNTIME", "target": "java-runtime", "versions": []interface{}{">=11"}},
			},
		},
		{
			name:        "malformed JSON returns an error",
			existing:    nil,
			overrideRaw: `{not valid json`,
			expectErr:   true,
		},
		{
			name:        "override entry missing a required field returns an error",
			existing:    nil,
			overrideRaw: `[{"type":"SUPPORTS","targetType":"SPECIFICATION","versions":["<=1.0"]}]`,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := MergeBindingsOverride(context.Background(), tt.existing, tt.overrideRaw)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, merged)
		})
	}
}

func TestMergeBindingsOverride_SkipsMalformedExistingEntries(t *testing.T) {
	getStdout, _ := testutil.CaptureOutput(t)

	existing := []interface{}{
		"not a map",
		map[string]interface{}{"type": "REQUIRES", "target": "NrInfra"}, // missing targetType
	}

	merged, err := MergeBindingsOverride(context.Background(), existing, `[{"type":"SUPPORTS","targetType":"SPECIFICATION","target":"protocol_version","versions":["<=1.0"]}]`)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{
		map[string]interface{}{"type": "SUPPORTS", "targetType": "SPECIFICATION", "target": "protocol_version", "versions": []interface{}{"<=1.0"}},
	}, merged)

	stdout := getStdout()
	assert.Contains(t, stdout, "skipping binding entry with unexpected type")
	assert.Contains(t, stdout, "skipping existing binding entry")
}
