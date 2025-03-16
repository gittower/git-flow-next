package model

// Workflow represents a git-flow workflow
type Workflow struct {
	Name          string
	BaseBranches  map[string]*Branch
	TopicBranches map[string]*Branch
}

// NewWorkflow creates a new Workflow
func NewWorkflow(name string) *Workflow {
	return &Workflow{
		Name:          name,
		BaseBranches:  make(map[string]*Branch),
		TopicBranches: make(map[string]*Branch),
	}
}

// AddBaseBranch adds a base branch to the workflow
func (w *Workflow) AddBaseBranch(branch *Branch) {
	w.BaseBranches[branch.Name] = branch
}

// AddTopicBranch adds a topic branch to the workflow
func (w *Workflow) AddTopicBranch(branch *Branch) {
	w.TopicBranches[branch.Name] = branch
}

// GetBranch gets a branch by name
func (w *Workflow) GetBranch(name string) *Branch {
	if branch, ok := w.BaseBranches[name]; ok {
		return branch
	}
	if branch, ok := w.TopicBranches[name]; ok {
		return branch
	}
	return nil
}
