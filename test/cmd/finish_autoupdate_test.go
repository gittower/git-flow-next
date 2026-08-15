package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFinishAutoUpdateFiltering tests that only branches with autoUpdate=true are updated
func TestFinishAutoUpdateFiltering(t *testing.T) {
	t.Parallel()
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a staging branch as child of main with autoUpdate=false
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "staging", "main")
	if err != nil {
		t.Fatalf("Failed to create staging branch: %v", err)
	}

	// Configure staging as a base branch with autoUpdate=false
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.autoUpdate", "false")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.downstreamStrategy", "merge")

	// Verify develop has autoUpdate=true (set it explicitly)
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")

	// Create and checkout a hotfix branch
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "critical-fix")
	if err != nil {
		t.Fatalf("Failed to create hotfix branch: %v\nOutput: %s", err, output)
	}

	// Make a change in the hotfix
	testutil.WriteFile(t, dir, "hotfix-change.txt", "Critical fix")

	_, err = testutil.RunGit(t, dir, "add", "hotfix-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	_, err = testutil.RunGit(t, dir, "commit", "-m", "Fix critical issue")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Add unique changes to staging and develop to ensure they can be identified
	testutil.RunGit(t, dir, "checkout", "staging")
	testutil.WriteFile(t, dir, "staging-only.txt", "Staging branch content")
	testutil.RunGit(t, dir, "add", "staging-only.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Staging branch commit")

	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop-only.txt", "Develop branch content")
	testutil.RunGit(t, dir, "add", "develop-only.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop branch commit")

	// Go back to hotfix branch
	testutil.RunGit(t, dir, "checkout", "hotfix/critical-fix")

	// Finish the hotfix
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "critical-fix")
	if err != nil {
		t.Fatalf("Failed to finish hotfix: %v\nOutput: %s", err, output)
	}

	// Verify the output mentions only develop was found for auto-update
	if !strings.Contains(output, "develop") || !strings.Contains(output, "auto-update") {
		t.Logf("Output doesn't mention develop auto-update: %s", output)
	}
	if strings.Contains(output, "staging") && strings.Contains(output, "auto-update") {
		t.Errorf("Output incorrectly mentions staging for auto-update: %s", output)
	}

	// Verify main has the hotfix
	testutil.RunGit(t, dir, "checkout", "main")
	if !testutil.FileExists(t, dir, "hotfix-change.txt") {
		t.Error("Hotfix changes not found in main branch")
	}

	// Verify develop was updated (autoUpdate=true)
	testutil.RunGit(t, dir, "checkout", "develop")
	if !testutil.FileExists(t, dir, "hotfix-change.txt") {
		t.Error("Hotfix changes not found in develop branch - should have been auto-updated")
	}
	// Verify develop still has its unique file
	if !testutil.FileExists(t, dir, "develop-only.txt") {
		t.Error("Develop branch lost its unique changes")
	}

	// Verify staging was NOT updated (autoUpdate=false)
	testutil.RunGit(t, dir, "checkout", "staging")
	if testutil.FileExists(t, dir, "hotfix-change.txt") {
		t.Error("Hotfix changes found in staging branch - should NOT have been auto-updated")
	}
	// Verify staging still has its unique file
	if !testutil.FileExists(t, dir, "staging-only.txt") {
		t.Error("Staging branch lost its unique changes")
	}
}

// TestFinishMultipleChildrenMixedAutoUpdate tests multiple children with mixed autoUpdate settings
func TestFinishMultipleChildrenMixedAutoUpdate(t *testing.T) {
	t.Parallel()
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create multiple child branches of main
	branches := []struct {
		name       string
		autoUpdate string
	}{
		{"staging", "false"},
		{"qa", "true"},
		{"preview", "false"},
		{"canary", "true"},
	}

	for _, branch := range branches {
		_, err = testutil.RunGit(t, dir, "checkout", "-b", branch.name, "main")
		if err != nil {
			t.Fatalf("Failed to create %s branch: %v", branch.name, err)
		}

		// Configure as base branch
		testutil.RunGit(t, dir, "config", "gitflow.branch."+branch.name+".type", "base")
		testutil.RunGit(t, dir, "config", "gitflow.branch."+branch.name+".parent", "main")
		testutil.RunGit(t, dir, "config", "gitflow.branch."+branch.name+".autoUpdate", branch.autoUpdate)
		testutil.RunGit(t, dir, "config", "gitflow.branch."+branch.name+".downstreamStrategy", "merge")

		// Add unique file to each branch
		testutil.WriteFile(t, dir, branch.name+"-file.txt", branch.name+" content")
		testutil.RunGit(t, dir, "add", branch.name+"-file.txt")
		testutil.RunGit(t, dir, "commit", "-m", branch.name+" initial commit")
	}

	// Keep develop with autoUpdate=true (set explicitly)
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")

	// Create and finish a hotfix
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "multi-test")
	if err != nil {
		t.Fatalf("Failed to create hotfix: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "hotfix-multi.txt", "Multi-child test")
	testutil.RunGit(t, dir, "add", "hotfix-multi.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Hotfix for multi-child test")

	// Finish the hotfix
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "multi-test")
	if err != nil {
		t.Fatalf("Failed to finish hotfix: %v\nOutput: %s", err, output)
	}

	// Check the output to see which branches were mentioned for auto-update
	expectedUpdated := []string{"develop", "qa", "canary"}
	expectedNotUpdated := []string{"staging", "preview"}

	for _, branchName := range expectedUpdated {
		testutil.RunGit(t, dir, "checkout", branchName)
		if !testutil.FileExists(t, dir, "hotfix-multi.txt") {
			t.Errorf("Branch %s should have been auto-updated but wasn't", branchName)
		}
		// Verify branch still has its unique file (if not develop)
		if branchName != "develop" {
			if !testutil.FileExists(t, dir, branchName+"-file.txt") {
				t.Errorf("Branch %s lost its unique changes", branchName)
			}
		}
	}

	for _, branchName := range expectedNotUpdated {
		testutil.RunGit(t, dir, "checkout", branchName)
		if testutil.FileExists(t, dir, "hotfix-multi.txt") {
			t.Errorf("Branch %s should NOT have been auto-updated but was", branchName)
		}
		// Verify branch still has its unique file
		if !testutil.FileExists(t, dir, branchName+"-file.txt") {
			t.Errorf("Branch %s lost its unique changes", branchName)
		}
	}
}

// TestFinishNoChildrenWithAutoUpdate tests that finish works when no children have autoUpdate=true
func TestFinishNoChildrenWithAutoUpdate(t *testing.T) {
	t.Parallel()
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set develop to autoUpdate=false
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "false")

	// Create a staging branch also with autoUpdate=false
	testutil.RunGit(t, dir, "checkout", "-b", "staging", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.autoUpdate", "false")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.downstreamStrategy", "merge")

	// Create and finish a hotfix
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "no-updates")
	if err != nil {
		t.Fatalf("Failed to create hotfix: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "hotfix-noupdate.txt", "No updates test")
	testutil.RunGit(t, dir, "add", "hotfix-noupdate.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Hotfix with no auto-updates")

	// Finish should succeed even with no children to update
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "no-updates")
	if err != nil {
		t.Fatalf("Failed to finish hotfix: %v\nOutput: %s", err, output)
	}

	// Verify main has the changes
	testutil.RunGit(t, dir, "checkout", "main")
	if !testutil.FileExists(t, dir, "hotfix-noupdate.txt") {
		t.Error("Hotfix changes not found in main branch")
	}

	// Verify develop was NOT updated
	testutil.RunGit(t, dir, "checkout", "develop")
	if testutil.FileExists(t, dir, "hotfix-noupdate.txt") {
		t.Error("Develop should not have been updated (autoUpdate=false)")
	}

	// Verify staging was NOT updated
	testutil.RunGit(t, dir, "checkout", "staging")
	if testutil.FileExists(t, dir, "hotfix-noupdate.txt") {
		t.Error("Staging should not have been updated (autoUpdate=false)")
	}
}

// foundChildOrder extracts the child base branch names from a finish run's
// output, in the order finish reported them. The reported order is the order the
// children are collected in, which is also the order they are integrated in and
// persisted into merge state, so it is the observable signal for issue #204.
func foundChildOrder(output string) []string {
	const prefix = "Found child base branch '"
	var names []string
	for _, line := range strings.Split(output, "\n") {
		start := strings.Index(line, prefix)
		if start < 0 {
			continue
		}
		rest := line[start+len(prefix):]
		end := strings.Index(rest, "'")
		if end < 0 {
			continue
		}
		names = append(names, rest[:end])
	}
	return names
}

// TestFinishChildBranchOrderIsDeterministic verifies that finish collects the
// auto-update child base branches in a stable, sorted order rather than in Go's
// randomized map iteration order.
//
// The order is not cosmetic: it decides the order children are integrated in,
// the order they appear in the output, and the ChildBranches list persisted into
// merge state (hence the resume order after a conflict). With three children an
// unsorted collection lands on sorted order by chance about one run in six, so
// the scenario is repeated across independent repositories — the probability of
// all iterations passing by luck is negligible.
func TestFinishChildBranchOrderIsDeterministic(t *testing.T) {
	t.Parallel()

	// Sorted order of the three auto-update children configured below.
	want := []string{"alpha", "develop", "zulu"}

	const iterations = 5
	for i := 0; i < iterations; i++ {
		dir := testutil.SetupTestRepo(t)
		defer testutil.CleanupTestRepo(t, dir)

		output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
		if err != nil {
			t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
		}

		// develop is already an auto-update child of main; add two more that
		// bracket it alphabetically, so a sorted result cannot be mistaken for
		// insertion or configuration order.
		testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")
		for _, name := range []string{"zulu", "alpha"} {
			if _, err := testutil.RunGit(t, dir, "checkout", "-b", name, "main"); err != nil {
				t.Fatalf("Failed to create %s branch: %v", name, err)
			}
			testutil.RunGit(t, dir, "config", "gitflow.branch."+name+".type", "base")
			testutil.RunGit(t, dir, "config", "gitflow.branch."+name+".parent", "main")
			testutil.RunGit(t, dir, "config", "gitflow.branch."+name+".autoUpdate", "true")
			testutil.RunGit(t, dir, "config", "gitflow.branch."+name+".downstreamStrategy", "merge")
		}

		output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "order-check")
		if err != nil {
			t.Fatalf("Failed to create hotfix: %v\nOutput: %s", err, output)
		}

		testutil.WriteFile(t, dir, "hotfix-order.txt", "Order determinism test")
		testutil.RunGit(t, dir, "add", "hotfix-order.txt")
		testutil.RunGit(t, dir, "commit", "-m", "Hotfix for order determinism test")

		output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "order-check")
		if err != nil {
			t.Fatalf("Failed to finish hotfix: %v\nOutput: %s", err, output)
		}

		got := foundChildOrder(output)
		if len(got) != len(want) {
			t.Fatalf("iteration %d: found %d child base branches %v, want %d %v\nOutput: %s",
				i, len(got), got, len(want), want, output)
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("iteration %d: child base branch order = %v, want %v\nOutput: %s",
					i, got, want, output)
			}
		}

		// Every child must still actually receive the hotfix — sorting decides
		// the order, never the selection.
		for _, name := range want {
			testutil.RunGit(t, dir, "checkout", name)
			if !testutil.FileExists(t, dir, "hotfix-order.txt") {
				t.Errorf("iteration %d: branch %s was not auto-updated", i, name)
			}
		}
	}
}
