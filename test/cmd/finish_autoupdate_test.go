package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFinishAutoUpdateFiltering tests that only branches with autoUpdate=true are updated
func TestFinishAutoUpdateFiltering(t *testing.T) {
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
