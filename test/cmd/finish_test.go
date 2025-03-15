package cmd_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestFinishFeatureBranch tests the finish command for feature branches
func TestFinishFeatureBranch(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = runGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Make a change on the feature branch
	cmd := exec.Command("bash", "-c", "echo 'Feature change' > feature.txt && git add feature.txt && git commit -m 'Add feature'")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to make change on feature branch: %v", err)
	}

	// Finish the feature branch
	output, err = runGitFlow(t, dir, "feature", "finish", "my-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected messages
	if !strings.Contains(output, "Merged branch 'feature/my-feature'") {
		t.Errorf("Expected output to contain 'Merged branch 'feature/my-feature'', got: %s", output)
	}

	if !strings.Contains(output, "Deleted branch 'feature/my-feature'") {
		t.Errorf("Expected output to contain 'Deleted branch 'feature/my-feature'', got: %s", output)
	}

	// Check if the branch was deleted
	if branchExists(t, dir, "feature/my-feature") {
		t.Errorf("Expected 'feature/my-feature' branch to be deleted")
	}

	// Check if we're on the develop branch
	currentBranch := getCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on 'develop' branch, got: %s", currentBranch)
	}

	// Check if the feature change is in the develop branch
	cmd = exec.Command("bash", "-c", "cat feature.txt")
	cmd.Dir = dir
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Failed to read feature.txt: %v", err)
	} else if strings.TrimSpace(string(outputBytes)) != "Feature change" {
		t.Errorf("Expected feature.txt to contain 'Feature change', got: %s", strings.TrimSpace(string(outputBytes)))
	}
}

// TestFinishReleaseAndHotfixBranches tests the finish command for release and hotfix branches
func TestFinishReleaseAndHotfixBranches(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a release branch
	output, err = runGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Make a change on the release branch
	cmd := exec.Command("bash", "-c", "echo 'Release change' > release.txt && git add release.txt && git commit -m 'Add release'")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to make change on release branch: %v", err)
	}

	// Finish the release branch
	output, err = runGitFlow(t, dir, "release", "finish", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Check if the branch was deleted
	if branchExists(t, dir, "release/1.0.0") {
		t.Errorf("Expected 'release/1.0.0' branch to be deleted")
	}

	// Create a hotfix branch
	output, err = runGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to create hotfix branch: %v\nOutput: %s", err, output)
	}

	// Make a change on the hotfix branch
	cmd = exec.Command("bash", "-c", "echo 'Hotfix change' > hotfix.txt && git add hotfix.txt && git commit -m 'Add hotfix'")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to make change on hotfix branch: %v", err)
	}

	// Finish the hotfix branch
	output, err = runGitFlow(t, dir, "hotfix", "finish", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to finish hotfix branch: %v\nOutput: %s", err, output)
	}

	// Check if the branch was deleted
	if branchExists(t, dir, "hotfix/1.0.1") {
		t.Errorf("Expected 'hotfix/1.0.1' branch to be deleted")
	}
}

// TestFinishWithCustomConfig tests the finish command with custom configuration
func TestFinishWithCustomConfig(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration
	input := "custom-main\ncustom-dev\nf/\nr/\nh/\ns/\n"
	output, err := runGitFlowWithInput(t, dir, input, "init", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = runGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Make a change on the feature branch
	cmd := exec.Command("bash", "-c", "echo 'Feature change' > feature.txt && git add feature.txt && git commit -m 'Add feature'")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to make change on feature branch: %v", err)
	}

	// Finish the feature branch
	output, err = runGitFlow(t, dir, "feature", "finish", "my-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected messages
	if !strings.Contains(output, "Merged branch 'f/my-feature'") {
		t.Errorf("Expected output to contain 'Merged branch 'f/my-feature'', got: %s", output)
	}

	// Check if the branch was deleted
	if branchExists(t, dir, "f/my-feature") {
		t.Errorf("Expected 'f/my-feature' branch to be deleted")
	}

	// Check if we're on the custom-dev branch
	currentBranch := getCurrentBranch(t, dir)
	if currentBranch != "custom-dev" {
		t.Errorf("Expected to be on 'custom-dev' branch, got: %s", currentBranch)
	}
}

// TestFinishNonExistentBranch tests the finish command with a non-existent branch
func TestFinishNonExistentBranch(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Finish a non-existent feature branch
	output, err = runGitFlow(t, dir, "feature", "finish", "non-existent")
	if err != nil {
		t.Fatalf("Failed to run finish command: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Branch 'feature/non-existent' does not exist") {
		t.Errorf("Expected output to contain 'Branch 'feature/non-existent' does not exist', got: %s", output)
	}
}

// Helper function to get the current branch
func getCurrentBranch(t *testing.T, dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	return strings.TrimSpace(string(output))
}
