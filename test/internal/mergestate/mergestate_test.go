package mergestate_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/mergestate"
	"github.com/gittower/git-flow-next/test/testutil"
)

// sampleState returns a MergeState with the critical fields populated.
func sampleState() *mergestate.MergeState {
	return &mergestate.MergeState{
		Action:         "finish",
		BranchType:     "feature",
		BranchName:     "cwd-indep",
		CurrentStep:    "delete_branch",
		ParentBranch:   "main",
		MergeStrategy:  "merge",
		FullBranchName: "feature/cwd-indep",
		ChildBranches:  []string{"develop"},
	}
}

// cwdCandidateStatePath returns the merge-state path that a CWD-implicit
// implementation would have written to: the state file under the process
// working directory's own git dir (the git-flow-next checkout).
func cwdCandidateStatePath(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	repo, err := git.Open(cwd)
	if err != nil {
		t.Fatalf("Failed to open ambient repo: %v", err)
	}
	return filepath.Join(repo.GitDir(), "gitflow", "state", "merge.json")
}

// snapshotFile records whether path exists and its contents.
func snapshotFile(t *testing.T, path string) (existed bool, content []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		t.Fatalf("Failed to snapshot %s: %v", path, err)
	}
	return true, data
}

func assertUnchanged(t *testing.T, path string, existedBefore bool, contentBefore []byte) {
	t.Helper()
	existedAfter, contentAfter := snapshotFile(t, path)
	if existedBefore != existedAfter {
		t.Errorf("Ambient CWD state path existence changed (before=%v after=%v): %s", existedBefore, existedAfter, path)
	}
	if existedBefore && !bytes.Equal(contentBefore, contentAfter) {
		t.Errorf("Ambient CWD state path content changed unexpectedly: %s", path)
	}
}

func initRepoB(t *testing.T) *git.Repo {
	t.Helper()
	dir := testutil.SetupTestRepo(t)
	t.Cleanup(func() { testutil.CleanupTestRepo(t, dir) })
	// Mark the repo git-flow-initialized via git config directly (no binary
	// dependency): merge-state persistence only needs a resolvable git dir.
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.version", "1.0"); err != nil {
		t.Fatalf("Failed to set gitflow.version in B: %v", err)
	}
	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("Failed to open B: %v", err)
	}
	return repo
}

func TestSaveMergeStateWritesUnderTargetGitDir(t *testing.T) {
	t.Parallel()
	repo := initRepoB(t)

	cwdCandidate := cwdCandidateStatePath(t)
	existedBefore, contentBefore := snapshotFile(t, cwdCandidate)

	if err := mergestate.SaveMergeState(repo, sampleState()); err != nil {
		t.Fatalf("SaveMergeState failed: %v", err)
	}

	target := filepath.Join(repo.GitDir(), "gitflow", "state", "merge.json")
	if _, err := os.Stat(target); err != nil {
		t.Errorf("Expected state file at B's git dir %s: %v", target, err)
	}

	assertUnchanged(t, cwdCandidate, existedBefore, contentBefore)
}

func TestLoadMergeStateRoundTripsOffCwd(t *testing.T) {
	t.Parallel()
	repo := initRepoB(t)

	want := sampleState()
	if err := mergestate.SaveMergeState(repo, want); err != nil {
		t.Fatalf("SaveMergeState failed: %v", err)
	}

	got, err := mergestate.LoadMergeState(repo)
	if err != nil {
		t.Fatalf("LoadMergeState failed: %v", err)
	}
	if got == nil {
		t.Fatal("LoadMergeState returned nil")
	}
	if got.Action != want.Action || got.BranchType != want.BranchType ||
		got.BranchName != want.BranchName || got.CurrentStep != want.CurrentStep ||
		got.ParentBranch != want.ParentBranch || got.MergeStrategy != want.MergeStrategy ||
		got.FullBranchName != want.FullBranchName {
		t.Errorf("Loaded state does not match saved state:\n got=%+v\nwant=%+v", got, want)
	}
	if len(got.ChildBranches) != 1 || got.ChildBranches[0] != "develop" {
		t.Errorf("ChildBranches not round-tripped: %v", got.ChildBranches)
	}
}

func TestClearMergeStateRemovesFileOffCwd(t *testing.T) {
	t.Parallel()
	repo := initRepoB(t)

	if err := mergestate.SaveMergeState(repo, sampleState()); err != nil {
		t.Fatalf("SaveMergeState failed: %v", err)
	}
	target := filepath.Join(repo.GitDir(), "gitflow", "state", "merge.json")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("Precondition failed: state file not created: %v", err)
	}

	if err := mergestate.ClearMergeState(repo); err != nil {
		t.Fatalf("ClearMergeState failed: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("Expected state file removed, stat err = %v", err)
	}
}

func TestLoadMergeStateAbsentReturnsNil(t *testing.T) {
	t.Parallel()
	repo := initRepoB(t)

	got, err := mergestate.LoadMergeState(repo)
	if err != nil {
		t.Errorf("Expected no error for absent state, got %v", err)
	}
	if got != nil {
		t.Errorf("Expected nil state when absent, got %+v", got)
	}
}
