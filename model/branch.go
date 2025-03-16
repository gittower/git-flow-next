package model

// Branch represents a Git branch
type Branch struct {
	Name     string
	Type     string
	Parent   string
	IsBase   bool
	IsTopic  bool
	Prefix   string
	FullName string
}

// NewBranch creates a new Branch
func NewBranch(name string, branchType string, parent string, prefix string) *Branch {
	isBase := branchType == "base"
	isTopic := branchType == "topic"

	var fullName string
	if isTopic && prefix != "" {
		fullName = prefix + name
	} else {
		fullName = name
	}

	return &Branch{
		Name:     name,
		Type:     branchType,
		Parent:   parent,
		IsBase:   isBase,
		IsTopic:  isTopic,
		Prefix:   prefix,
		FullName: fullName,
	}
}
