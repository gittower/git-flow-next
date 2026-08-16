# GIT-FLOW-LIST(1)

## NAME

git-flow-list - List topic branches

## SYNOPSIS

**git-flow** *topic* **list** [**--worktrees**]

## DESCRIPTION

List all topic branches of the specified type. This command works with any topic branch type (feature, release, hotfix, support, or custom types defined in your configuration).

The list command displays all local branches that match the topic branch type's prefix.

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

## OPTIONS

**--worktrees**
: Append a column reporting each branch's linked worktree. The cell takes one of four forms: `-` when the branch has no linked worktree, the worktree's path when it is live and clean, `<path> [n]` when it has *n* changed entries, and `<path> (missing)` when the worktree is still registered but its directory is gone. A worktree git-flow did not create is additionally tagged `(unmanaged)`, always last, so a dirty hand-made worktree reads `<path> [2] (unmanaged)`. Available for every topic branch type, including custom ones. There is no short form.

: The count is of **git status --porcelain** entries, not of files: under Git's default untracked handling an untracked directory is one entry however many files it holds, and `-uall` is deliberately never used, since it would walk every untracked directory in a worktree full of build output. A `(missing)` row never carries a count — there is nothing there to count.

: Paths are shown relative to the **main worktree root**, not to the directory the command was run from, so the same listing reads the same from anywhere in the repository. A path with no relative form — a different volume on Windows — is shown in full instead.

: A branch checked out in the **main worktree** shows `-`. The column reports *linked* worktrees, and the repository root is not one of them; **git-flow-checkout**(1) makes the opposite choice, because navigating to a branch and listing the worktrees it lives in are different questions.

: If the status of one worktree cannot be read, that row degrades to its bare path, a warning naming the worktree goes to standard error, and the rest of the listing is printed as usual. A failure to read the worktree list or the provenance markers is fatal instead (exit 3): reporting every branch as having no worktree, or every worktree as unmanaged, would be worse than an error.

## OUTPUT FORMAT

The list command prints a heading and one branch per line, indented by two spaces:
```
$ git flow feature list
Feature branches:
  api-v2
  docs
  user-auth
```

Branch names are shown without the prefix for readability.

With **--worktrees**, the branch column is padded and the worktree column follows:
```
$ git flow feature list --worktrees
Feature branches:
  api-v2     ../review-copy [3] (unmanaged)
  docs       -
  user-auth  ../my-project-worktrees/feature/user-auth
```

Without the flag the output is exactly what it was before the column existed.

## EXAMPLES

### Basic Usage

List all feature branches:
```bash
git flow feature list
```

List all release branches:
```bash
git flow release list
```

List all hotfix branches:
```bash
git flow hotfix list
```

### Worktree Status

See which features have a worktree, and which of those have uncommitted work:
```bash
git flow feature list --worktrees
```

### Custom Branch Types

List custom branch type:
```bash
git flow bugfix list
git flow epic list
```

## WORKFLOW INTEGRATION

### Before Starting Work
```bash
# Check existing features before starting new one
git flow feature list
git flow feature start new-feature
```

### During Development
```bash
# See all active releases
git flow release list
```

### Cleanup Planning
```bash
# List all features to identify candidates for deletion
git flow feature list
git flow feature delete old-feature
```

## EMPTY RESULTS

When no branches of the type exist:
```bash
$ git flow feature list
No feature branches found
```

## CONFIGURATION

List behavior is controlled by branch type configuration:

### Prefix Configuration
```bash
git config gitflow.branch.feature.prefix feature/
git config gitflow.branch.release.prefix release/
git config gitflow.branch.hotfix.prefix hotfix/
```

The list command automatically uses the configured prefix to identify branches of each type.

## REMOTE BRANCHES

The list command shows only local branches by default. To see remote branches:

```bash
# Show remote feature branches
git branch -r | grep feature/

# Or use Git directly
git ls-remote --heads origin | grep feature/
```

## EXIT STATUS

**0**
: Successful listing (even if no branches found)

**1**
: git-flow is not initialized in this repository, or the command line could not be parsed — an unknown flag, or an argument where the command takes none

**2**
: Invalid topic branch type

**3**
: A Git operation failed

## SEE ALSO

**git-flow**(1), **git-flow-start**(1), **git-flow-delete**(1), **git-flow-worktree**(1), **git-branch**(1), **git-ls-remote**(1)

## NOTES

- Only shows local branches - use **git branch -r** for remote branches
- Branch names are displayed without the prefix for readability
- The command takes no arguments; there is no name-pattern filter
- Empty results are not considered an error condition
- Custom topic branch types work exactly like built-in types
- **--worktrees** does not change the empty-result message, and the flag alone decides whether the column appears
- A worktree whose HEAD is detached has no branch, so it never appears in this listing; **git flow worktree list** shows it
- A `(missing)` row is a registered worktree whose directory was deleted by hand. **git flow worktree prune** drops the entry and the row with it