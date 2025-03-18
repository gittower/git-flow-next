package model

// MergeState represents the state of a merge operation
type MergeState struct {
	Action          string   `json:"action"`          // "finish"
	BranchType      string   `json:"branchType"`      // feature, release, hotfix, etc.
	BranchName      string   `json:"branchName"`      // name of the branch being merged
	CurrentStep     string   `json:"currentStep"`     // current step in the process (merge, update_children, delete_branch)
	ParentBranch    string   `json:"parentBranch"`    // target branch for the merge
	MergeStrategy   string   `json:"mergeStrategy"`   // merge strategy being used
	FullBranchName  string   `json:"fullBranchName"`  // full name of the branch (with prefix)
	ChildBranches   []string `json:"childBranches"`   // child branches that need to be updated
	UpdatedBranches []string `json:"updatedBranches"` // child branches that have been updated
}
