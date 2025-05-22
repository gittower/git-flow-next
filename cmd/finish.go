package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/mergestate"
	"github.com/gittower/git-flow-next/internal/update"
)

// Step constants
const (
	stepMerge          = "merge"
	stepCreateTag      = "create_tag"
	stepUpdateChildren = "update_children"
	stepDeleteBranch   = "delete_branch"
)

// Strategy constants
const (
	strategyRebase = "rebase"
	strategySquash = "squash"
	strategyMerge  = "merge"
)

// TagOptions contains options for tag creation when finishing a branch
type TagOptions struct {
	ShouldTag   *bool  // Whether to create a tag (nil means use config default)
	ShouldSign  *bool  // Whether to sign the tag (nil means use config default)
	SigningKey  string // Key to use for signing
	Message     string // Custom message for the tag
	MessageFile string // File containing the message
	TagName     string // Custom tag name
}

// BranchRetentionOptions contains options for branch retention when finishing a branch
type BranchRetentionOptions struct {
	Keep        *bool // Whether to keep the branch (nil means use config default)
	KeepRemote  *bool // Whether to keep the remote branch (nil means use config default)
	KeepLocal   *bool // Whether to keep the local branch (nil means use config default)
	ForceDelete *bool // Whether to force delete the branch (nil means use config default)
}

// FinishCommand is the implementation of the finish command for topic branches
func FinishCommand(branchType string, name string, continueOp bool, abortOp bool, force bool, tagOptions *TagOptions, retentionOptions *BranchRetentionOptions) {
	if err := executeFinish(branchType, name, continueOp, abortOp, force, tagOptions, retentionOptions); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// executeFinish performs the actual branch finishing logic and returns any errors
func executeFinish(branchType string, name string, continueOp bool, abortOp bool, force bool, tagOptions *TagOptions, retentionOptions *BranchRetentionOptions) error {
	// Get configuration early
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Get branch configuration
	branchConfig, ok := cfg.Branches[branchType]
	if !ok {
		return &errors.InvalidBranchTypeError{BranchType: branchType}
	}

	// Check if there's a merge in progress
	if mergestate.IsMergeInProgress() {
		state, err := mergestate.LoadMergeState()
		if err != nil {
			return &errors.GitError{Operation: "load merge state", Err: err}
		}

		// Get the branch config for the state's branch type
		stateBranchConfig, ok := cfg.Branches[state.BranchType]
		if !ok {
			return &errors.InvalidBranchTypeError{BranchType: state.BranchType}
		}

		if abortOp {
			return handleAbort(state)
		}

		if continueOp {
			return handleContinue(state, stateBranchConfig, tagOptions, retentionOptions)
		}

		return &errors.MergeInProgressError{BranchName: state.FullBranchName}
	}

	// Don't allow continue or abort if no merge is in progress
	if continueOp || abortOp {
		return &errors.NoMergeInProgressError{}
	}

	// Resolve branch name (try with and without prefix)
	resolvedName, err := resolveBranchName(name, branchConfig)
	if err != nil {
		return err
	}
	name = resolvedName

	// If the branch exists but doesn't have the expected prefix
	if !strings.HasPrefix(name, branchConfig.Prefix) {
		if !force {
			// Get the short name for tag creation
			shortName := name
			if strings.Contains(name, "/") {
				parts := strings.Split(name, "/")
				shortName = parts[len(parts)-1]
			}

			// Prompt user for confirmation
			fmt.Printf("Warning: Branch '%s' is not a standard %s branch (missing prefix '%s').\n", name, branchType, branchConfig.Prefix)
			fmt.Printf("Finishing this branch will:\n")
			fmt.Printf("1. Merge it into '%s' using the %s strategy\n", branchConfig.Parent, branchConfig.UpstreamStrategy)

			// Adjust tag message based on tag options
			showTagMessage := branchConfig.Tag

			// Command-line flags override config
			if tagOptions != nil && tagOptions.ShouldTag != nil {
				showTagMessage = *tagOptions.ShouldTag
			}

			if showTagMessage {
				// Show tag name based on options
				displayTagName := shortName
				if tagOptions != nil && tagOptions.TagName != "" {
					displayTagName = tagOptions.TagName
				} else if branchConfig.TagPrefix != "" {
					displayTagName = branchConfig.TagPrefix + shortName
				}
				fmt.Printf("2. Create a tag '%s'\n", displayTagName)
			}

			fmt.Printf("3. Delete the branch after successful merge\n\n")
			fmt.Printf("Do you want to continue? [y/N]: ")

			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				return fmt.Errorf("operation cancelled by user")
			}
		}
	}

	// Regular finish command flow
	return finishBranch(branchType, name, branchConfig, tagOptions, retentionOptions)
}

func finishBranch(branchType string, name string, branchConfig config.BranchConfig, tagOptions *TagOptions, retentionOptions *BranchRetentionOptions) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Validate inputs
	if name == "" {
		return &errors.InvalidBranchNameError{Name: name}
	}

	// Get the short name by removing the prefix if it exists
	shortName := name
	if strings.HasPrefix(name, branchConfig.Prefix) {
		shortName = strings.TrimPrefix(name, branchConfig.Prefix)
	} else if strings.Contains(name, "/") {
		// For non-standard branches, use the last part after the slash
		parts := strings.Split(name, "/")
		shortName = parts[len(parts)-1]
	}

	// Check if branch exists
	if err := git.BranchExists(name); err != nil {
		return &errors.BranchNotFoundError{BranchName: name}
	}

	// Get target branch (always the parent branch)
	targetBranch := branchConfig.Parent

	// Check if target branch exists
	if err := git.BranchExists(targetBranch); err != nil {
		return &errors.BranchNotFoundError{BranchName: targetBranch}
	}

	// Find child base branches that need to be updated
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	childBranches := []string{}
	for branchName, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeBase) && branch.Parent == targetBranch {
			fmt.Printf("Found child base branch '%s' to update\n", branchName)
			childBranches = append(childBranches, branchName)
		}
	}

	// Save merge state before starting
	state := &mergestate.MergeState{
		Action:          "finish",
		BranchType:      branchType,
		BranchName:      shortName,
		CurrentStep:     stepMerge,
		ParentBranch:    targetBranch,
		MergeStrategy:   branchConfig.UpstreamStrategy,
		FullBranchName:  name,
		ChildBranches:   childBranches,
		UpdatedBranches: []string{},
	}
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}

	return finish(state, branchConfig, tagOptions, retentionOptions)
}

// resolveBranchName tries to find the branch name with and without prefix
func resolveBranchName(name string, branchConfig config.BranchConfig) (string, error) {
	// Try name as-is first
	if err := git.BranchExists(name); err == nil {
		return name, nil
	}

	// If not found as-is, try with prefix
	if !strings.HasPrefix(name, branchConfig.Prefix) {
		fullName := branchConfig.Prefix + name
		if err := git.BranchExists(fullName); err == nil {
			return fullName, nil
		}
	}

	return "", &errors.BranchNotFoundError{BranchName: name}
}

// handleCreateTagStep handles the tag creation step
func handleCreateTagStep(state *mergestate.MergeState, branchConfig config.BranchConfig, tagOptions *TagOptions, retentionOptions *BranchRetentionOptions) error {
	// 1. Start with branch configuration default
	shouldTag := branchConfig.Tag

	// 2. Check for branch-specific config override
	branchSpecificTagConfig, err := git.GetConfig(fmt.Sprintf("gitflow.%s.finish.notag", state.BranchType))
	if err == nil && branchSpecificTagConfig == "true" {
		// notag=true means don't create a tag
		shouldTag = false
	}

	// 3. Command-line flags override config
	if tagOptions != nil && tagOptions.ShouldTag != nil {
		shouldTag = *tagOptions.ShouldTag
	}

	if shouldTag {
		if err := createTagForBranch(state, branchConfig, tagOptions); err != nil {
			return err
		}
	}

	// Move to next step
	state.CurrentStep = stepUpdateChildren
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}
	return handleContinue(state, branchConfig, tagOptions, retentionOptions)
}

// createTagForBranch creates a tag for the finished branch
func createTagForBranch(state *mergestate.MergeState, branchConfig config.BranchConfig, tagOptions *TagOptions) error {
	// Determine tag name
	// 1. Start with branch name and apply prefix from branch config
	tagName := state.BranchName
	if branchConfig.TagPrefix != "" {
		tagName = branchConfig.TagPrefix + state.BranchName
	}

	// 2. Command-line custom tag name overrides config
	if tagOptions != nil && tagOptions.TagName != "" {
		tagName = tagOptions.TagName
	}

	// Determine tag message
	// Default message
	message := fmt.Sprintf("Tagging version %s", tagName)

	// Command-line message overrides default
	if tagOptions != nil && tagOptions.Message != "" {
		message = tagOptions.Message
	}

	// Handle message file
	useMessageFile := false
	messageFilePath := ""

	// 1. Check for branch-specific message file config
	configMessageFile, err := git.GetConfig(fmt.Sprintf("gitflow.%s.finish.messagefile", state.BranchType))
	if err == nil && configMessageFile != "" {
		useMessageFile = true
		messageFilePath = configMessageFile
	}

	// 2. Command-line message file overrides config
	if tagOptions != nil && tagOptions.MessageFile != "" {
		useMessageFile = true
		messageFilePath = tagOptions.MessageFile
	}

	// Determine signing options
	// 1. Start with not signing
	shouldSign := false

	// 2. Check branch-specific signing config
	signConfig, err := git.GetConfig(fmt.Sprintf("gitflow.%s.finish.sign", state.BranchType))
	if err == nil && signConfig == "true" {
		shouldSign = true
	}

	// 3. Command-line signing flags override config
	if tagOptions != nil && tagOptions.ShouldSign != nil {
		shouldSign = *tagOptions.ShouldSign
	}

	// Determine signing key
	signingKey := ""

	// 1. Check branch-specific signing key
	configSigningKey, err := git.GetConfig(fmt.Sprintf("gitflow.%s.finish.signingkey", state.BranchType))
	if err == nil && configSigningKey != "" {
		signingKey = configSigningKey
		shouldSign = true // Specifying a key implies signing
	}

	// 2. Command-line signing key overrides config
	if tagOptions != nil && tagOptions.SigningKey != "" {
		signingKey = tagOptions.SigningKey
		shouldSign = true // Specifying a key implies signing
	}

	// Now create the tag with all the options
	if err := createTagWithOptions(tagName, state.ParentBranch, message, shouldSign, signingKey, useMessageFile, messageFilePath); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("create tag '%s'", tagName), Err: err}
	}
	fmt.Printf("Created tag '%s'\n", tagName)
	return nil
}

// handleUpdateChildrenStep handles updating child base branches
func handleUpdateChildrenStep(state *mergestate.MergeState, branchConfig config.BranchConfig, tagOptions *TagOptions, retentionOptions *BranchRetentionOptions) error {
	// Find next child branch to update
	nextBranch := findNextBranchToUpdate(state)

	// If no more branches to update, move to final step
	if nextBranch == "" {
		state.CurrentStep = stepDeleteBranch
		if err := mergestate.SaveMergeState(state); err != nil {
			return &errors.GitError{Operation: "save merge state", Err: err}
		}
		return handleContinue(state, branchConfig, tagOptions, retentionOptions)
	}

	// Update the next child branch
	if err := updateChildBranch(nextBranch, state); err != nil {
		return err
	}

	// Mark this branch as updated
	state.UpdatedBranches = append(state.UpdatedBranches, nextBranch)
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}

	// Continue with next branch
	return handleContinue(state, branchConfig, tagOptions, retentionOptions)
}

// findNextBranchToUpdate finds the next child branch that needs updating
func findNextBranchToUpdate(state *mergestate.MergeState) string {
	for _, branch := range state.ChildBranches {
		alreadyUpdated := false
		for _, updated := range state.UpdatedBranches {
			if branch == updated {
				alreadyUpdated = true
				break
			}
		}
		if !alreadyUpdated {
			return branch
		}
	}
	return ""
}

// updateChildBranch updates a single child branch
func updateChildBranch(branchName string, state *mergestate.MergeState) error {
	fmt.Printf("Updating child base branch '%s' from '%s'...\n", branchName, state.ParentBranch)

	// Load config to get merge strategy for this child branch
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	childBranchConfig, ok := cfg.Branches[branchName]
	if !ok {
		return &errors.GitError{Operation: fmt.Sprintf("get config for branch '%s'", branchName), Err: fmt.Errorf("branch config not found")}
	}

	// Use the shared update logic
	err = update.UpdateBranchFromParent(branchName, state.ParentBranch, childBranchConfig.DownstreamStrategy, true, state)
	if err != nil {
		if _, ok := err.(*errors.UnresolvedConflictsError); ok {
			msg := fmt.Sprintf("Merge conflicts detected while updating base branch '%s'. Resolve conflicts and run 'git flow %s finish --continue %s'\n", branchName, state.BranchType, state.BranchName)
			msg += fmt.Sprintf("To abort the merge, run 'git flow %s finish --abort %s'", state.BranchType, state.BranchName)
			fmt.Println(msg)
			return err
		}
		return err
	}

	return nil
}

// handleDeleteBranchStep handles branch deletion
func handleDeleteBranchStep(state *mergestate.MergeState, retentionOptions *BranchRetentionOptions) error {
	// Ensure we're on the parent branch before deletion
	if err := git.Checkout(state.ParentBranch); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout parent branch '%s'", state.ParentBranch), Err: err}
	}

	// Get retention settings
	keep, keepRemote, keepLocal, forceDelete := getBranchRetentionSettings(state.BranchType, retentionOptions)

	// Delete branches based on settings
	if err := deleteBranchesIfNeeded(state, keep, keepRemote, keepLocal, forceDelete); err != nil {
		return err
	}

	// Clear the merge state
	if err := mergestate.ClearMergeState(); err != nil {
		return &errors.GitError{Operation: "clear merge state", Err: err}
	}

	fmt.Printf("Successfully finished branch '%s' and updated %d child base branches\n", state.FullBranchName, len(state.UpdatedBranches))
	return nil
}

// getBranchRetentionSettings determines branch retention settings
func getBranchRetentionSettings(branchType string, retentionOptions *BranchRetentionOptions) (keep, keepRemote, keepLocal, forceDelete bool) {
	// Start with defaults (delete both local and remote)
	keep = false
	keepRemote = false
	keepLocal = false
	forceDelete = false

	// Check branch-specific config
	configKeep, err := git.GetConfig(fmt.Sprintf("gitflow.%s.finish.keep", branchType))
	if err == nil && configKeep == "true" {
		keep = true
	}
	configKeepRemote, err := git.GetConfig(fmt.Sprintf("gitflow.%s.finish.keepremote", branchType))
	if err == nil && configKeepRemote == "true" {
		keepRemote = true
	}
	configKeepLocal, err := git.GetConfig(fmt.Sprintf("gitflow.%s.finish.keeplocal", branchType))
	if err == nil && configKeepLocal == "true" {
		keepLocal = true
	}
	configForceDelete, err := git.GetConfig(fmt.Sprintf("gitflow.%s.finish.force-delete", branchType))
	if err == nil && configForceDelete == "true" {
		forceDelete = true
	}

	// Command-line flags override config
	if retentionOptions != nil {
		if retentionOptions.Keep != nil {
			keep = *retentionOptions.Keep
		}
		if retentionOptions.KeepRemote != nil {
			keepRemote = *retentionOptions.KeepRemote
		}
		if retentionOptions.KeepLocal != nil {
			keepLocal = *retentionOptions.KeepLocal
		}
		if retentionOptions.ForceDelete != nil {
			forceDelete = *retentionOptions.ForceDelete
		}
	}

	// If keep is set, it overrides individual settings
	if keep {
		keepRemote = true
		keepLocal = true
	}

	return keep, keepRemote, keepLocal, forceDelete
}

// deleteBranchesIfNeeded deletes branches based on retention settings
func deleteBranchesIfNeeded(state *mergestate.MergeState, keep, keepRemote, keepLocal, forceDelete bool) error {
	// Delete remote branch if not keeping it and if remote branch exists
	if !keepRemote {
		// Only attempt to delete if the remote branch actually exists
		if git.RemoteBranchExists("origin", state.FullBranchName) {
			remoteBranch := fmt.Sprintf("origin/%s", state.FullBranchName)
			if err := git.DeleteRemoteBranch("origin", state.FullBranchName); err != nil {
				return &errors.GitError{Operation: fmt.Sprintf("delete remote branch '%s'", remoteBranch), Err: err}
			}
		}
	}

	// Delete local branch if not keeping it
	if !keepLocal {
		if err := git.DeleteBranch(state.FullBranchName, forceDelete); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("delete branch '%s'", state.FullBranchName), Err: err}
		}
	}

	return nil
}

func finish(state *mergestate.MergeState, branchConfig config.BranchConfig, tagOptions *TagOptions, retentionOptions *BranchRetentionOptions) error {
	// Checkout target branch
	err := git.Checkout(state.ParentBranch)
	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout target branch '%s'", state.ParentBranch), Err: err}
	}
	fmt.Printf("Switched to branch '%s'\n", state.ParentBranch)

	// Perform merge based on strategy
	fmt.Printf("Merging using strategy: %v\n", strings.ToLower(branchConfig.UpstreamStrategy))
	var mergeErr error
	switch strings.ToLower(branchConfig.UpstreamStrategy) {
	case strategyRebase:
		fmt.Printf("Rebase strategy selected\n")
		// For rebase, we need to:
		// 1. Stay on feature branch
		err = git.Checkout(state.FullBranchName)
		if err != nil {
			return &errors.GitError{Operation: "checkout feature branch for rebase", Err: err}
		}
		// 2. Rebase onto target branch
		mergeErr = git.Rebase(state.ParentBranch)
		if mergeErr == nil {
			// 3. If rebase succeeds, checkout target and merge (should be fast-forward)
			err = git.Checkout(state.ParentBranch)
			if err != nil {
				return &errors.GitError{Operation: "checkout target branch after rebase", Err: err}
			}
			mergeErr = git.Merge(state.FullBranchName)
		}
	case strategySquash:
		mergeErr = git.SquashMerge(state.FullBranchName)
	case strategyMerge:
		mergeErr = git.Merge(state.FullBranchName)
	default:
		return &errors.GitError{Operation: fmt.Sprintf("unknown merge strategy: %s", strings.ToLower(branchConfig.UpstreamStrategy)), Err: nil}
	}

	if mergeErr != nil {
		if strings.Contains(mergeErr.Error(), "conflict") {
			// Save state before returning conflict error
			state.CurrentStep = stepMerge
			if err := mergestate.SaveMergeState(state); err != nil {
				return &errors.GitError{Operation: "save merge state", Err: err}
			}

			msg := fmt.Sprintf("Merge conflicts detected. Resolve conflicts and run 'git flow %s finish --continue %s'\n", state.BranchType, state.BranchName)
			msg += fmt.Sprintf("To abort the merge, run 'git flow %s finish --abort %s'", state.BranchType, state.BranchName)
			fmt.Println(msg)
			return &errors.UnresolvedConflictsError{}
		}
		return &errors.GitError{Operation: "merge branch", Err: mergeErr}
	}

	// Move to next step (tag creation)
	state.CurrentStep = stepCreateTag
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}

	return handleContinue(state, branchConfig, tagOptions, retentionOptions)
}

func handleContinue(state *mergestate.MergeState, branchConfig config.BranchConfig, tagOptions *TagOptions, retentionOptions *BranchRetentionOptions) error {
	switch state.CurrentStep {
	case stepMerge:
		// Check if there are still conflicts
		if git.HasConflicts() {
			return &errors.UnresolvedConflictsError{}
		}

		// Move to next step
		state.CurrentStep = stepCreateTag
		if err := mergestate.SaveMergeState(state); err != nil {
			return &errors.GitError{Operation: "save merge state", Err: err}
		}
		return handleContinue(state, branchConfig, tagOptions, retentionOptions)

	case stepCreateTag:
		return handleCreateTagStep(state, branchConfig, tagOptions, retentionOptions)

	case stepUpdateChildren:
		return handleUpdateChildrenStep(state, branchConfig, tagOptions, retentionOptions)

	case stepDeleteBranch:
		return handleDeleteBranchStep(state, retentionOptions)

	default:
		return &errors.GitError{Operation: fmt.Sprintf("unknown step '%s'", state.CurrentStep), Err: nil}
	}
}

func handleAbort(state *mergestate.MergeState) error {
	// Abort the merge based on strategy
	var err error
	switch state.MergeStrategy {
	case strategyMerge:
		err = git.MergeAbort()
	case strategyRebase:
		err = git.RebaseAbort()
	default:
		err = git.MergeAbort() // Default to merge abort
	}

	if err != nil {
		return &errors.GitError{Operation: "abort merge", Err: err}
	}

	// Checkout the original branch
	if err := git.Checkout(state.FullBranchName); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout original branch '%s'", state.FullBranchName), Err: err}
	}

	// Clear the merge state
	if err := mergestate.ClearMergeState(); err != nil {
		return &errors.GitError{Operation: "clear merge state", Err: err}
	}

	return nil
}

// createTagWithOptions creates a new Git tag with the given name and message, and handles signing and message file
func createTagWithOptions(tagName, targetBranch, message string, shouldSign bool, signingKey string, useMessageFile bool, messageFilePath string) error {
	// Check if tag already exists
	cmd := exec.Command("git", "show-ref", "--tags", tagName)
	if err := cmd.Run(); err == nil {
		// Tag exists
		return nil
	}

	// Build command arguments
	args := []string{"tag"}

	// Use annotated tag
	args = append(args, "-a")

	// Apply signing if requested
	if shouldSign {
		args = append(args, "-s")

		// Apply signing key if specified
		if signingKey != "" {
			args = append(args, "-u", signingKey)
		}
	}

	// Apply tag name
	args = append(args, tagName)

	// Apply message
	if useMessageFile {
		args = append(args, "-F", messageFilePath)
	} else {
		args = append(args, "-m", message)
	}

	// Execute tag command
	cmd = exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tag: %w (output: %s)", err, string(output))
	}

	return nil
}

// Old createTag function can now delegate to createTagWithOptions
func createTag(tagName, targetBranch, message string) error {
	return createTagWithOptions(tagName, targetBranch, message, false, "", false, "")
}

// getBoolFlag converts two opposite boolean flags into a single *bool value
// If positive is true, returns &true
// If negative is true, returns &false
// If neither is set, returns nil
func getBoolFlag(positive, negative bool) *bool {
	if positive {
		return &positive
	}
	if negative {
		falseBool := false
		return &falseBool
	}
	return nil
}
