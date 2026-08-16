package config_test

import (
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
)

// TestResolveStartShouldCreateWorktree covers E13: the precedence of the worktree
// decision — the branch type's Layer-1 worktree property, overridden by the
// --worktree/--no-worktree flags. There is deliberately no Layer-2 key (SC-12).
//
// Table-driven by exception: ResolveStartShouldCreateWorktree is a pure function,
// which is TESTING_GUIDELINES.md's documented carve-out from the
// one-case-per-function rule.
// Steps:
// 1. Builds a configuration whose feature branch type carries a known worktree default
// 2. Calls config.ResolveStartShouldCreateWorktree for each combination of default and flag
// 3. Verifies an absent flag leaves the Layer-1 property in charge
// 4. Verifies a flag of either polarity always wins over the Layer-1 property
// 5. Verifies an unknown branch type resolves to false instead of panicking
func TestResolveStartShouldCreateWorktree(t *testing.T) {
	t.Parallel()

	yes := true
	no := false

	cases := []struct {
		name        string
		branchType  string
		typeDefault bool
		flag        *bool
		want        bool
	}{
		{"default false and no flag", "feature", false, nil, false},
		{"default true and no flag", "feature", true, nil, true},
		{"flag true beats default false", "feature", false, &yes, true},
		{"flag false beats default true", "feature", true, &no, false},
		{"unknown branch type", "nosuchtype", true, nil, false},
	}

	for _, tc := range cases {
		cfg := &config.Config{
			Branches: map[string]config.BranchConfig{
				"feature": {Type: string(config.BranchTypeTopic), Worktree: tc.typeDefault},
			},
		}
		got := config.ResolveStartShouldCreateWorktree(cfg, tc.branchType, tc.flag)
		if got != tc.want {
			t.Errorf("%s: ResolveStartShouldCreateWorktree = %v, want %v", tc.name, got, tc.want)
		}
	}
}
