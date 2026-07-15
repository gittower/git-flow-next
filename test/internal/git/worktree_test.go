package git_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/mergestate"
	"github.com/gittower/git-flow-next/test/testutil"
)

func TestGetGitDirInRegularRepo(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Change to the test directory
	withGitRepo(t, dir, func() {
		gitDir, err := git.GetGitDir()
		if err != nil {
			t.Fatalf("GetGitDir failed: %v", err)
		}

		// In a regular repo, GetGitDir should return ".git"
		if gitDir != ".git" {
			t.Errorf("Expected GetGitDir to return '.git' in regular repo, got %q", gitDir)
		}
	})
}

func TestGetGitDirInWorktree(t *testing.T) {
	// Setup main repo
	mainRepo := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, mainRepo)

	// Create a worktree
	worktreePath, err := os.MkdirTemp("", "git-flow-worktree-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree: %v", err)
	}
	defer os.RemoveAll(worktreePath)

	// Remove the directory so git worktree add can create it
	os.RemoveAll(worktreePath)

	// Create worktree
	_, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktreePath, "-b", "worktree-branch")
	if err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Change to worktree and verify GetGitDir returns worktree git dir
	withGitRepo(t, worktreePath, func() {
		gitDir, err := git.GetGitDir()
		if err != nil {
			t.Fatalf("GetGitDir failed in worktree: %v", err)
		}

		// In a worktree, GetGitDir should NOT return ".git"
		// It should return a path containing "worktrees"
		if gitDir == ".git" {
			t.Error("Expected GetGitDir to return actual git directory path in worktree, not '.git'")
		}

		if !strings.Contains(gitDir, "worktrees") {
			t.Errorf("Expected GetGitDir path to contain 'worktrees', got %q", gitDir)
		}
	})
}

func TestMergeStateSaveLoadInWorktree(t *testing.T) {
	// Setup main repo
	mainRepo := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, mainRepo)

	// Create a worktree
	worktreePath, err := os.MkdirTemp("", "git-flow-worktree-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree: %v", err)
	}
	defer os.RemoveAll(worktreePath)

	// Remove the directory so git worktree add can create it
	os.RemoveAll(worktreePath)

	// Create worktree from main
	_, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktreePath, "-b", "worktree-branch")
	if err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Change to worktree and test merge state operations
	withGitRepo(t, worktreePath, func() {
		// Create test state using delete_branch step with an existing branch so that
		// IsMergeInProgress validation passes (it checks branch existence for non-conflict steps)
		state := &mergestate.MergeState{
			Action:         "finish",
			BranchType:     "feature",
			BranchName:     "test-feature",
			CurrentStep:    "delete_branch",
			ParentBranch:   "main",
			MergeStrategy:  "merge",
			FullBranchName: "worktree-branch",
		}

		// Test save
		err := mergestate.SaveMergeState(state)
		if err != nil {
			t.Fatalf("SaveMergeState failed in worktree: %v", err)
		}

		// Verify file was created in the correct location (worktree git dir, not .git)
		gitDir, err := git.GetGitDir()
		if err != nil {
			t.Fatalf("GetGitDir failed: %v", err)
		}
		stateFilePath := filepath.Join(gitDir, "gitflow", "state", "merge.json")
		if _, err := os.Stat(stateFilePath); os.IsNotExist(err) {
			t.Errorf("State file was not created at expected path: %s", stateFilePath)
		}

		// Test load
		loaded, err := mergestate.LoadMergeState()
		if err != nil {
			t.Fatalf("LoadMergeState failed in worktree: %v", err)
		}
		if loaded == nil {
			t.Fatal("LoadMergeState returned nil in worktree")
		}
		if loaded.BranchName != state.BranchName {
			t.Errorf("Loaded state BranchName = %q, want %q", loaded.BranchName, state.BranchName)
		}
		if loaded.BranchType != state.BranchType {
			t.Errorf("Loaded state BranchType = %q, want %q", loaded.BranchType, state.BranchType)
		}

		// Test IsMergeInProgress
		if !mergestate.IsMergeInProgress() {
			t.Error("IsMergeInProgress returned false after saving state in worktree")
		}

		// Test clear
		err = mergestate.ClearMergeState()
		if err != nil {
			t.Fatalf("ClearMergeState failed in worktree: %v", err)
		}

		// Verify state was cleared
		if mergestate.IsMergeInProgress() {
			t.Error("IsMergeInProgress returned true after clearing state in worktree")
		}
	})
}

func TestMergeStateNotSharedBetweenWorktrees(t *testing.T) {
	// Setup main repo
	mainRepo := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, mainRepo)

	// Create first worktree
	worktree1, err := os.MkdirTemp("", "git-flow-worktree1-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree1: %v", err)
	}
	defer os.RemoveAll(worktree1)
	os.RemoveAll(worktree1)

	_, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktree1, "-b", "worktree1-branch")
	if err != nil {
		t.Fatalf("Failed to create worktree1: %v", err)
	}

	// Create second worktree
	worktree2, err := os.MkdirTemp("", "git-flow-worktree2-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree2: %v", err)
	}
	defer os.RemoveAll(worktree2)
	os.RemoveAll(worktree2)

	_, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktree2, "-b", "worktree2-branch")
	if err != nil {
		t.Fatalf("Failed to create worktree2: %v", err)
	}

	// Save state in worktree1 using delete_branch step with an existing branch
	// so that IsMergeInProgress validation passes
	withGitRepo(t, worktree1, func() {
		state := &mergestate.MergeState{
			Action:         "finish",
			BranchType:     "feature",
			BranchName:     "worktree1-feature",
			CurrentStep:    "delete_branch",
			FullBranchName: "worktree1-branch",
			ParentBranch:   "main",
		}
		err := mergestate.SaveMergeState(state)
		if err != nil {
			t.Fatalf("SaveMergeState failed in worktree1: %v", err)
		}
	})

	// Verify state is NOT visible in worktree2
	withGitRepo(t, worktree2, func() {
		if mergestate.IsMergeInProgress() {
			t.Error("Worktree2 should not see merge state from worktree1")
		}

		loaded, err := mergestate.LoadMergeState()
		if err != nil {
			t.Fatalf("LoadMergeState failed in worktree2: %v", err)
		}
		if loaded != nil {
			t.Errorf("Worktree2 loaded state from worktree1: %+v", loaded)
		}
	})

	// Verify state is still in worktree1
	withGitRepo(t, worktree1, func() {
		if !mergestate.IsMergeInProgress() {
			t.Error("Worktree1 should still have merge state")
		}
	})
}
