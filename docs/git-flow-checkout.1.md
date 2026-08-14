# GIT-FLOW-CHECKOUT(1)

## NAME

git-flow-checkout - Switch to topic branches

## SYNOPSIS

**git-flow** *topic* **checkout** *name*|*nameprefix* [**--worktree**] [**--no-cd**] [**--clobber**] [**--quiet**] [**--showcommands**]

## DESCRIPTION

Switch to an existing topic branch of the specified type. This command works with any topic branch type (feature, release, hotfix, support, or custom types) and supports partial name matching for convenience.

The checkout command is a convenient wrapper around **git checkout** that automatically handles branch prefixes and provides partial name matching.

When the branch has a worktree, checkout **navigates to it** instead of switching the current worktree's branch. See **WORKTREES** below.

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

*name*|*nameprefix*
: Full name or partial name prefix of the topic branch to checkout. Supports partial matching for convenience.

## OPTIONS

**--worktree**, **-w**
: Create the branch's worktree if it does not exist yet, then navigate to it. The worktree is created at the path the **gitflow.worktreePath** template computes and is recorded as git-flow-created. Without this flag, a branch with no worktree is simply checked out.

**--no-cd**
: Do not write a navigation destination for the calling shell, even when **GIT_FLOW_CD_FILE** is set. The path is still printed for manual use, and the branch switch — when there is one — still happens.

**--clobber**
: Remove a plain directory standing in the way of a new worktree. Only meaningful together with **--worktree**; on its own it does nothing at all, because with no worktree to create there is nothing to clobber. Removal is refused when the target is a file, a registered worktree of this repository, or a directory containing a `.git` entry.

**--quiet**, **-q**
: Do not print the tip naming **git flow shell-init**.

**--showcommands**
: Show the underlying git commands as they are executed. Navigating to a worktree runs no git command, so nothing is shown for it.

## WORKTREES

A branch that has a worktree lives somewhere else on disk, and Git allows a branch in only one worktree at a time. Rather than failing the way a plain **git checkout** would, checkout **navigates**: it prints the worktree's path and offers it to the calling shell through **GIT_FLOW_CD_FILE**, leaving the current worktree's branch untouched.

| State | **--worktree** | Behaviour |
|---|---|---|
| The branch has a worktree elsewhere (linked or the main one) | either | Navigate to it. The current worktree's branch is unchanged |
| The branch's worktree **is** the worktree you are in | either | Ordinary checkout — `Switched to branch '<name>'`, nothing written to the channel |
| The branch has a worktree whose directory is gone, or was replaced by something that is not a worktree | either | Error naming **git flow worktree prune**; nothing is written |
| The branch has no worktree | no | Ordinary checkout |
| The branch has no worktree | yes | Create it, record it as git-flow-created, then navigate |

Navigating to a worktree that already exists **never changes its provenance**: a worktree created by hand with **git worktree add** stays `(unmanaged)` in **git flow worktree list** no matter how often you check the branch out. Only creation writes the marker.

The per-branch-type worktree default governs **start**, not checkout: checkout creates a missing worktree only when **--worktree** is given explicitly.

## ENVIRONMENT

**GIT_FLOW_CD_FILE**
: When set to a writable path, a navigation writes its absolute destination there. git-flow runs as a subprocess and cannot change its parent shell's working directory, so this variable is how a caller asks for that destination in a form it can act on. The variable is an input git-flow reads, never one it sets, which is why it is not a Git config key.

: The destination is written **only after** the worktree is known to exist, so a refused or failed command leaves the file empty; an ordinary branch switch is not a navigation and writes nothing. **--no-cd** suppresses the write. A failure to write is **not fatal**: the command still succeeds and a warning is printed to standard error.

: **git flow shell-init** prints a shell wrapper that supplies this file and changes directory for you. See **git-flow-shell-init**(1).

## PARTIAL NAME MATCHING

The checkout command supports partial name matching for convenience:

**Exact Match** (preferred)
: If a branch matches the exact name, it's selected

**Prefix Match**
: If no exact match, looks for branches starting with the given prefix

**Ambiguous Match**
: If multiple branches match the prefix, shows options and fails

## EXAMPLES

### Basic Usage

Checkout a specific feature branch:
```bash
git flow feature checkout user-authentication
```

Checkout using partial name:
```bash
# Assuming you have feature/user-authentication
git flow feature checkout user
```

Checkout a release branch:
```bash
git flow release checkout 1.2.0
```

### With and Without Prefixes

These commands are equivalent:
```bash
git flow feature checkout user-auth
git flow feature checkout feature/user-auth
```

### Partial Matching Examples

If you have these feature branches:
- `feature/user-authentication`
- `feature/user-profile`  
- `feature/api-endpoints`

```bash
git flow feature checkout user    # Ambiguous - shows options
git flow feature checkout user-a  # Matches user-authentication
git flow feature checkout api     # Matches api-endpoints
```

## AMBIGUOUS MATCHES

When multiple branches match a prefix:

```bash
$ git flow feature checkout user
Error: Ambiguous branch name 'user'. Matches:
  feature/user-authentication
  feature/user-profile
  
Please specify a more complete name.
```

## WORKFLOW INTEGRATION

### Daily Development
```bash
# Quick switch between features
git flow feature checkout api
git flow feature checkout user-a
```

### Context Switching
```bash
# Switch to hotfix for urgent work
git flow hotfix checkout critical-fix

# Return to feature work
git flow feature checkout main-feature
```

### Team Collaboration
```bash
# Checkout teammate's branch for review
git flow feature checkout teammate-feature
```

## BRANCH VALIDATION

The checkout command validates:

**Branch Exists**
: Verifies the target branch exists locally

**Branch Type Match**
: Ensures branch has the correct prefix for the topic type

**Unique Resolution**
: Handles partial name matching and ambiguity

## REMOTE BRANCHES

The checkout command works only with local branches. For remote branches:

```bash
# Fetch remote branches first
git fetch origin

# Create local tracking branch
git checkout -b feature/remote-feature origin/feature/remote-feature

# Then use git-flow commands normally
git flow feature checkout remote-feature
```

## CONFIGURATION

Checkout behavior respects branch type configuration:

### Prefix Configuration
```bash
git config gitflow.branch.feature.prefix feature/
git config gitflow.branch.release.prefix release/
git config gitflow.branch.hotfix.prefix hotfix/
```

The checkout command uses these prefixes to identify and validate branches.

## ERROR HANDLING

Common error scenarios:

**Branch not found**:
```bash
$ git flow feature checkout nonexistent
Error: Branch 'feature/nonexistent' not found
```

**Ambiguous match**:
```bash
$ git flow feature checkout user
Error: Ambiguous branch name 'user'. Multiple matches found.
```

**Wrong branch type**:
```bash
$ git flow feature checkout release/1.2.0
Error: Branch 'release/1.2.0' is not a feature branch
```

## ALTERNATIVES

For more advanced checkout operations, use Git directly:

```bash
# Checkout and create tracking branch
git checkout -b feature/new-branch origin/feature/new-branch

# Checkout specific commit
git checkout feature/branch-name~3

# Checkout with path specification
git checkout feature/branch -- specific-file.txt
```

## EXIT STATUS

**0**
: Successful checkout or navigation

**2**
: Invalid topic branch type

**3**
: A Git operation failed — an ambiguous branch name, a failed checkout, or a worktree that is no longer at its recorded path

**5**
: Branch not found

**6**
: A refusal: the target path of a new worktree is occupied, or **--clobber** was asked to remove a file, a registered worktree, or a directory containing a `.git` entry

## SEE ALSO

**git-flow**(1), **git-flow-list**(1), **git-flow-start**(1), **git-flow-worktree**(1), **git-flow-shell-init**(1), **git-checkout**(1)

## NOTES

- Only works with local branches - fetch remote branches first if needed
- Supports partial name matching for convenience
- Branch prefixes are automatically handled
- Use **git flow list** to see available branches before checkout
- For complex checkout operations, use **git checkout** directly
- Partial matching prioritizes exact matches over prefix matches
- The worktree lookup runs on the **resolved** branch name, so a partial name finds the worktree of the branch it resolves to
- Checkout uses **git checkout** rather than **git switch**, keeping the required Git version at 2.17 rather than 2.23