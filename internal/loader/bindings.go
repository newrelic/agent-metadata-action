package loader

import (
	"agent-metadata-action/internal/logging"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// MergeBindingsOverride merges override bindings (parsed from a JSON array) into an existing
// bindings list, matching entries by (type, targetType, target). A match replaces the existing
// entry; no match appends. This lets callers supply binding values that must be computed at
// release time (e.g. a protocol version read from the caller's build metadata) as an action
// input, rather than hardcoding them in agentDefinition.yml.
//
// An empty overrideJSON is a no-op. Malformed JSON or an override entry missing a required
// matching field is a hard error, since it very likely indicates a workflow bug. Existing
// entries with an unexpected shape are dropped with a warning, consistent with how the rest of
// this loader degrades gracefully on malformed input.
func MergeBindingsOverride(ctx context.Context, existing []interface{}, overrideJSON string) ([]interface{}, error) {
	if overrideJSON == "" {
		return existing, nil
	}

	var overrides []map[string]interface{}
	if err := json.Unmarshal([]byte(overrideJSON), &overrides); err != nil {
		return nil, fmt.Errorf("failed to parse bindings-override JSON: %w", err)
	}

	overrideKeys := make(map[string]struct{}, len(overrides))
	for _, override := range overrides {
		key, err := bindingKey(override)
		if err != nil {
			return nil, fmt.Errorf("invalid bindings-override entry: %w", err)
		}
		overrideKeys[key] = struct{}{}
	}

	merged := make([]interface{}, 0, len(existing)+len(overrides))
	for _, item := range existing {
		binding, ok := item.(map[string]interface{})
		if !ok {
			logging.Warnf(ctx, "skipping binding entry with unexpected type %T while applying bindings-override", item)
			continue
		}
		key, err := bindingKey(binding)
		if err != nil {
			logging.Warnf(ctx, "skipping existing binding entry while applying bindings-override: %v", err)
			continue
		}
		if _, overridden := overrideKeys[key]; overridden {
			continue
		}
		merged = append(merged, binding)
	}

	for _, override := range overrides {
		merged = append(merged, override)
	}

	return merged, nil
}

// bindingKey builds a match key for a binding from its (type, targetType, target) fields,
// erroring if any of them is missing or not a non-empty string.
func bindingKey(binding map[string]interface{}) (string, error) {
	fields := make([]string, 0, 3)
	for _, field := range []string{"type", "targetType", "target"} {
		value, ok := binding[field].(string)
		if !ok || value == "" {
			return "", fmt.Errorf("missing or empty required field %q", field)
		}
		fields = append(fields, value)
	}
	return strings.Join(fields, "|"), nil
}
