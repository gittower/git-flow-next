package config_test

import (
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/stretchr/testify/assert"
)

// These tests cover issue #210: ff / no-ff / ff-only is one tri-state setting whose
// third value is read in the "finish" Layer-2 namespace only, so integrate keeps
// today's behavior exactly.
//
// The config is built in memory (config.DefaultConfig plus CommandConfig entries) —
// the established pattern for resolver tests in this package. Loading from a plain
// `git init` repository would yield an empty CommandConfig and make every assertion
// below vacuous.

// TestResolveFinishOptionsFFOnlyConfigRequiresFastForward verifies the Layer-2
// finish.ff-only key resolves to RequireFastForward without also setting
// NoFastForward.
// Steps:
// 1. Builds a default config and sets gitflow.release.finish.ff-only=true
// 2. Resolves finish options for the release branch type
// 3. Verifies RequireFastForward is true and NoFastForward is false
func TestResolveFinishOptionsFFOnlyConfigRequiresFastForward(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.CommandConfig["gitflow.release.finish.ff-only"] = "true"

	resolved := config.ResolveFinishOptions(cfg, "release", "1.0.0", nil, nil, nil, nil, nil, nil, nil)

	assert.True(t, resolved.RequireFastForward, "expected ff-only config to require a fast-forward")
	assert.False(t, resolved.NoFastForward, "expected ff-only not to also select no-ff")
}

// TestResolveFinishOptionsFFOnlyBeatsFFWithinLayer2 verifies that within Layer 2 the
// ff-only key wins over the ff key, which pins the ordering of the two reads.
// Steps:
// 1. Builds a default config and sets both finish.ff=true and finish.ff-only=true
// 2. Resolves finish options for the release branch type
// 3. Verifies RequireFastForward is true
func TestResolveFinishOptionsFFOnlyBeatsFFWithinLayer2(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.CommandConfig["gitflow.release.finish.ff"] = "true"
	cfg.CommandConfig["gitflow.release.finish.ff-only"] = "true"

	resolved := config.ResolveFinishOptions(cfg, "release", "1.0.0", nil, nil, nil, nil, nil, nil, nil)

	assert.True(t, resolved.RequireFastForward, "expected ff-only to win over ff within Layer 2")
}

// TestResolveIntegrateOptionsIgnoresFFOnlyConfig verifies the ff-only key is inert in
// the integrate namespace: the resolution is identical to the no-ff-only baseline.
// Steps:
// 1. Builds a default config and sets both integrate.no-ff=true and integrate.ff-only=true
// 2. Resolves integrate options for the develop branch
// 3. Verifies RequireFastForward is false and NoFastForward is true (baseline behavior)
func TestResolveIntegrateOptionsIgnoresFFOnlyConfig(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.CommandConfig["gitflow.develop.integrate.no-ff"] = "true"
	cfg.CommandConfig["gitflow.develop.integrate.ff-only"] = "true"

	resolved := config.ResolveIntegrateOptions(cfg, "develop", nil, nil, nil)

	assert.False(t, resolved.RequireFastForward, "expected the ff-only key to be inert for integrate")
	assert.True(t, resolved.NoFastForward, "expected the integrate no-ff baseline to be preserved")
}
