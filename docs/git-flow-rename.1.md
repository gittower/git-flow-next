# GIT-FLOW-RENAME(1)

## NAME

git-flow-rename - Rename topic branches

## SYNOPSIS

**git-flow** *topic* **rename** *old-name* *new-name*

**git-flow rename** [*new-name*]

## DESCRIPTION

Rename a topic branch to a new name while preserving its Git history and maintaining the correct prefix for the branch type. This command works with any topic branch type (feature, release, hotfix, support, or custom types).

The rename operation updates the local branch name and can optionally update remote tracking branches.

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

*old-name*
: Current name of the topic branch to rename

*new-name*
: New name for the topic branch. When using the shorthand **git-flow rename** on a topic branch, this is the only required argument.

## BRANCH NAME HANDLING

The rename command automatically handles prefixes:

- **Input names**: Can be with or without prefix
- **Output names**: Always use the correct prefix for the branch type
- **Validation**: Ensures new name doesn't conflict with existing branches

Examples:
```bash
# These are equivalent:
git flow feature rename old-name new-name
git flow feature rename feature/old-name feature/new-name
```

## EXAMPLES

### Basic Usage

Rename a feature branch:
```bash
git flow feature rename user-auth user-authentication
```

Rename current topic branch (shorthand):
```bash
# When on a feature branch
git flow rename user-authentication
```

Rename a release:
```bash
git flow release rename 1.2.0 1.2.1
```

Rename a hotfix:
```bash
git flow hotfix rename quick-fix critical-security-fix
```

### Working with Current Branch

When on the branch to rename:
```bash
# Checkout the branch first
git checkout feature/old-name

# Rename using shorthand
git flow rename new-name
```

### Custom Branch Types

Rename custom branch types:
```bash
git flow bugfix rename bug-123 fix-login-issue
git flow epic rename user-management user-auth-system
```

## REMOTE BRANCH HANDLING

When a topic branch has a remote tracking branch:

**Local Only** (default)
: Renames only the local branch, remote tracking remains unchanged

**With Remote Update**
: Manual remote updates may be needed after renaming

Example workflow with remote:
```bash
# Rename local branch
git flow feature rename old-name new-name

# Update remote (delete old, push new)
git push origin :feature/old-name              # Delete old remote branch
git push -u origin feature/new-name            # Push new branch with tracking
```

## VALIDATION

The rename command performs several validations:

**Existing Branch Check**
: Verifies the old branch exists

**Name Conflict Check**
: Ensures new name doesn't conflict with existing branches

**Valid Name Check**
: Validates new name is a valid Git reference

**Current Branch Handling**
: Automatically updates your current branch if you're on the renamed branch

## WORKFLOW INTEGRATION

### During Development
```bash
# Rename for clarity
git flow feature rename temp-fix user-profile-fix
```

### Before Publishing
```bash
# Clean up name before sharing
git flow feature rename wip-feature finalized-user-auth
git push -u origin feature/finalized-user-auth
```

### Release Management
```bash
# Adjust release version
git flow release rename 1.2.0 1.3.0
```

## HISTORY PRESERVATION

Renaming preserves all Git history:
- **Commits**: All commits remain intact
- **Branch history**: Full development history preserved  
- **Merge history**: References to the branch in merge commits remain
- **Reflog**: Git reflog tracks the rename operation

## BRANCH STATE

git-flow records two pieces of per-branch state in the repository's local Git configuration, both keyed on the branch name. Renaming a branch carries them to the new name, so the renamed branch keeps everything git-flow knows about it and nothing is left behind under the old name:

**gitflow.worktree.*branch*.managed**
: Records that git-flow created the branch's worktree. Carrying it means a renamed branch's worktree is still reported as git-flow's rather than being demoted to `(unmanaged)` in **git-flow-worktree**(1) and **git-flow-list**(1), and that **git flow worktree remove** clears the marker under the new name instead of stranding it under the old one.

**gitflow.branch.*branch*.base**
: Records the start point the branch was created from. Nothing reads it today; it is written by **git-flow-start**(1) and cleared by **git-flow-finish**(1) and **git-flow-delete**(1) under the branch's current name. Carrying it is therefore about cleanup rather than behavior: left under the old name it is never cleared, so it outlives the branch it describes.

The worktree **directory** keeps its old-name path. Provenance is tracked by the marker, not by the shape of the path, so there is nothing to move on disk. Use **git-flow-worktree**(1) if you want the directory to match the new name.

If a key cannot be moved, git-flow prints a warning to standard error naming the key and continues. The branch rename is not undone: the ref has already moved and there is nothing to roll back to. The remaining keys are still migrated.

## CONFIGURATION

Rename behavior respects branch type configuration:

### Prefix Configuration
```bash
git config gitflow.branch.feature.prefix feature/
git config gitflow.branch.release.prefix release/
```

The rename command automatically applies the correct prefix based on the branch type.

## ERROR HANDLING

Common error scenarios:

**Branch doesn't exist**:
```bash
$ git flow feature rename nonexistent new-name
Error: Branch 'feature/nonexistent' not found
```

**Name conflict**:
```bash
$ git flow feature rename old-name existing-name  
Error: Branch 'feature/existing-name' already exists
```

**Invalid name**:
```bash
$ git flow feature rename old-name "invalid name with spaces"
Error: Invalid branch name 'invalid name with spaces'
```

## EXIT STATUS

**0**
: Successful rename operation

**1**
: Source branch not found

**2**
: Target branch already exists

**3**
: Invalid branch name

**4**
: Git operation failed

**5**
: Invalid topic branch type

## SEE ALSO

**git-flow**(1), **git-flow-list**(1), **git-flow-delete**(1), **git-flow-worktree**(1), **gitflow-config**(5), **git-branch**(1), **git-push**(1)

## NOTES

- Renaming preserves all Git history and commits
- Remote branches must be updated manually after local rename
- Cannot rename to a name that already exists
- Branch prefixes are automatically handled
- The branch rename itself is atomic - it either succeeds completely or leaves the branch untouched. Per-branch configuration is migrated afterwards; a migration that fails is reported on standard error and left for you to fix, not rolled back
- Consider team communication when renaming shared branches