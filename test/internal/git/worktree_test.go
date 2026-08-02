package git_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/mergestate"
	"github.com/gittower/git-flow-next/test/testutil"
)

func TestGetGitDirInRegularRepo(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	repo := openRepo(t, dir)
	gitDir := repo.GitDir()

	if !filepath.IsAbs(gitDir) {
		t.Errorf("Expected GitDir() to be absolute, got %q", gitDir)
	}
	if _, err := os.Stat(gitDir); err != nil {
		t.Errorf("GitDir() does not exist: %v", err)
	}
}

func TestGetGitDirInWorktree(t *testing.T) {
	t.Parallel()
	mainRepo := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, mainRepo)

	worktreePath, err := os.MkdirTemp("", "git-flow-worktree-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree: %v", err)
	}
	defer os.RemoveAll(worktreePath)
	os.RemoveAll(worktreePath)

	if _, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktreePath, "-b", "worktree-branch"); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	repo := openRepo(t, worktreePath)
	gitDir := repo.GitDir()

	if !filepath.IsAbs(gitDir) {
		t.Errorf("Expected worktree GitDir() to be absolute, got %q", gitDir)
	}
	if !strings.Contains(gitDir, "worktrees") {
		t.Errorf("Expected GitDir() path to contain 'worktrees', got %q", gitDir)
	}
}

func TestMergeStateSaveLoadInWorktree(t *testing.T) {
	t.Parallel()
	mainRepo := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, mainRepo)

	worktreePath, err := os.MkdirTemp("", "git-flow-worktree-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree: %v", err)
	}
	defer os.RemoveAll(worktreePath)
	os.RemoveAll(worktreePath)

	if _, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktreePath, "-b", "worktree-branch"); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	repo := openRepo(t, worktreePath)

	state := &mergestate.MergeState{
		Action:         "finish",
		BranchType:     "feature",
		BranchName:     "test-feature",
		CurrentStep:    "delete_branch",
		ParentBranch:   "main",
		MergeStrategy:  "merge",
		FullBranchName: "worktree-branch",
	}

	if err := mergestate.SaveMergeState(repo, state); err != nil {
		t.Fatalf("SaveMergeState failed in worktree: %v", err)
	}

	stateFilePath := filepath.Join(repo.GitDir(), "gitflow", "state", "merge.json")
	if _, err := os.Stat(stateFilePath); os.IsNotExist(err) {
		t.Errorf("State file was not created at expected path: %s", stateFilePath)
	}

	loaded, err := mergestate.LoadMergeState(repo)
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

	if !mergestate.IsMergeInProgress(repo) {
		t.Error("IsMergeInProgress returned false after saving state in worktree")
	}

	if err := mergestate.ClearMergeState(repo); err != nil {
		t.Fatalf("ClearMergeState failed in worktree: %v", err)
	}

	if mergestate.IsMergeInProgress(repo) {
		t.Error("IsMergeInProgress returned true after clearing state in worktree")
	}
}

func TestMergeStateNotSharedBetweenWorktrees(t *testing.T) {
	t.Parallel()
	mainRepo := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, mainRepo)

	worktree1, err := os.MkdirTemp("", "git-flow-worktree1-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree1: %v", err)
	}
	defer os.RemoveAll(worktree1)
	os.RemoveAll(worktree1)

	if _, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktree1, "-b", "worktree1-branch"); err != nil {
		t.Fatalf("Failed to create worktree1: %v", err)
	}

	worktree2, err := os.MkdirTemp("", "git-flow-worktree2-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree2: %v", err)
	}
	defer os.RemoveAll(worktree2)
	os.RemoveAll(worktree2)

	if _, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktree2, "-b", "worktree2-branch"); err != nil {
		t.Fatalf("Failed to create worktree2: %v", err)
	}

	repo1 := openRepo(t, worktree1)
	repo2 := openRepo(t, worktree2)

	state := &mergestate.MergeState{
		Action:         "finish",
		BranchType:     "feature",
		BranchName:     "worktree1-feature",
		CurrentStep:    "delete_branch",
		FullBranchName: "worktree1-branch",
		ParentBranch:   "main",
	}
	if err := mergestate.SaveMergeState(repo1, state); err != nil {
		t.Fatalf("SaveMergeState failed in worktree1: %v", err)
	}

	if mergestate.IsMergeInProgress(repo2) {
		t.Error("Worktree2 should not see merge state from worktree1")
	}
	loaded, err := mergestate.LoadMergeState(repo2)
	if err != nil {
		t.Fatalf("LoadMergeState failed in worktree2: %v", err)
	}
	if loaded != nil {
		t.Errorf("Worktree2 loaded state from worktree1: %+v", loaded)
	}

	if !mergestate.IsMergeInProgress(repo1) {
		t.Error("Worktree1 should still have merge state")
	}
}
