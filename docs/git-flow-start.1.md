# GIT-FLOW-START(1)

## NAME

git-flow-start - Create a topic branch, checked out or in its own worktree

## SYNOPSIS

**git-flow** *topic* **start** [*name*] [*base*] [**--worktree**|**--no-worktree**] [**--worktree-path** *path*] [**--no-cd**] [**--quiet**] [*options*]

## DESCRIPTION

Create a new topic branch of the specified type and check it out, or create it in its own worktree. This command works with any topic branch type (feature, release, hotfix, support, or custom types defined in your configuration).

The new branch is created from the configured starting point for the topic branch type, or from the specified base commit/branch if provided.

With **--worktree** — or a branch type whose `gitflow.branch.<type>.worktree` default is true — the branch is created **without being checked out here** and a worktree is created for it instead. The current worktree's HEAD is unchanged, because Git allows a branch to be checked out in only one worktree at a time. See **WORKTREES** below.

By default, start fetches from the remote before creating the branch, so its remote-tracking refs are current first. (The branch is still created from the configured local start point; the fetch refreshes remote-tracking refs but does not fast-forward that start point.) Opt out with **--no-fetch** or by setting `gitflow.<type>.start.fetch false`. If no remote is configured, the fetch is skipped silently, and a fetch failure is a non-fatal warning — the branch is still created (start has no sync gate).

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

*name*
: Name of the new topic branch (without the prefix - that's added automatically). Optional: when omitted, git-flow runs the `filter-flow-<type>-start-version` filter with an empty version argument and uses its trimmed output as the branch name. If no such filter is configured or it yields no output, the command fails with `branch name cannot be empty`.

*base*
: Optional base commit, tag, or branch to start from instead of the configured starting point

## OPTIONS

**--fetch**
: Fetch from remote before creating branch to ensure latest state (this is the default). If no remote is configured, the fetch is skipped silently. A fetch failure is a non-fatal warning — the branch is still created (start has no sync gate). Overrides git config setting `gitflow.<type>.start.fetch`.

**--no-fetch**
: Don't fetch from remote before creating branch. Use this to opt out of the default fetch. Overrides git config setting `gitflow.<type>.start.fetch`.

**--worktree**, **-w**
: Create a worktree for the new branch instead of checking the branch out here. The worktree is created at the path the **gitflow.worktreePath** template computes and is recorded as git-flow-created. Overrides git config setting `gitflow.branch.<type>.worktree`.

**--no-worktree**
: Don't create a worktree, even when the branch type defaults to one. Overrides git config setting `gitflow.branch.<type>.worktree`.

: Passing **--worktree** and **--no-worktree** together is not an error: the positive flag wins regardless of the order they appear in, as it does for every **--x**/**--no-x** pair in git-flow. **--worktree-path** counts as a positive flag too, because it implies creation.

**--worktree-path** *path*
: Create the worktree at *path* instead of at the computed one. Implies **--worktree**. A relative *path* resolves against the **invocation directory**, the way a hand-typed path is meant to; note the deliberate asymmetry with the **gitflow.worktreePath** template, whose relative values resolve against the main worktree root.

**--no-cd**
: Do not write a navigation destination for the calling shell, even when **GIT_FLOW_CD_FILE** is set. The path is still printed for manual use, and the branch and worktree are still created.

**--quiet**, **-q**
: Do not print the tip about **git flow shell-init**. The tip is printed only when a worktree was created and **GIT_FLOW_CD_FILE** is unset, so this matters exactly when the calling shell has no wrapper installed.

## WORKTREES

A worktree lets the new branch live in its own directory, so an in-progress branch does not have to be stashed away before another one can be started. Because Git allows a branch in only one worktree at a time, **start --worktree** creates the branch **without checking it out** where you ran it: the invocation worktree stays on the branch it was on, and the new branch is checked out in the new worktree.

The worktree's path comes from the **gitflow.worktreePath** template unless **--worktree-path** overrides it. Missing parent directories are created. An existing **empty** directory is accepted as the target, matching plain **git worktree add**; a file or a directory with content in it is refused, and nothing is created. A **--worktree-path** inside the repository work tree is allowed, again matching Git's own behavior; the worktree simply shows up as an untracked directory.

The target path is validated **before** anything happens — before the fetch and before the `pre-flow-<type>-start` hook — so a command that cannot succeed leaves no branch, no worktree, no provenance marker and no side effect behind. One consequence is worth knowing: when the target path is occupied **and** the branch already exists, the occupied path is what you are told about (exit status **6**), not the existing branch. Clear the path first.

If worktree creation fails *after* the branch was created — a full disk, a permission problem, a race — the **branch is kept** and the failure is reported. Nothing deletes a branch on an error path. Retry with **git flow worktree add** *branch* once the cause is fixed.

## BRANCH NAMING

Topic branches are named using the configured prefix pattern:
- **Full name**: *prefix* + *name*
- **Example**: For `git flow feature start user-auth` with prefix `feature/`, creates `feature/user-auth`

## STARTING POINTS

Each topic branch type has a configured starting point:

**feature**
: Typically starts from develop branch

**release**  
: Usually starts from develop but merges to main

**hotfix**
: Starts from main branch for emergency fixes

**support**
: Starts from main for long-term maintenance

**Custom types**
: Use configured `startPoint` or fall back to `parent` branch

## EXAMPLES

### Basic Usage

Start a new feature:
```bash
git flow feature start user-authentication
```

Start a release:
```bash
git flow release start 1.2.0
```

Start a hotfix:
```bash
git flow hotfix start critical-security-fix
```

### Deriving the Name from a Version Filter

Start a release without a name, letting the configured `filter-flow-release-start-version` filter supply the version:
```bash
git flow release start
```

### Custom Starting Points

Start feature from specific commit:
```bash
git flow feature start emergency-fix abc123def
```

Start release from specific develop commit:
```bash
git flow release start 1.2.0 develop~3
```

Start hotfix from specific tag:
```bash
git flow hotfix start 1.1.1 v1.1.0
```

### Starting in a Worktree

Start a feature in its own worktree, leaving the current one where it is:
```bash
git flow feature start user-authentication --worktree
```

Choose the worktree's location instead of using the computed one:
```bash
git flow feature start review-copy --worktree-path ../review-copy
```

Combine a worktree with an explicit start point:
```bash
git flow hotfix start 1.1.1 v1.1.0 --worktree
```

### Without Remote Synchronization

Start fetches by default; skip the fetch to work offline or avoid touching the network:
```bash
git flow feature start new-api --no-fetch
```

## CONFIGURATION

Start behavior is controlled by these configuration keys:

### Branch Starting Points
```bash
git config gitflow.branch.feature.startPoint develop
git config gitflow.branch.release.startPoint develop
git config gitflow.branch.hotfix.startPoint main
```

### Prefixes
```bash
git config gitflow.branch.feature.prefix feature/
git config gitflow.branch.release.prefix release/
git config gitflow.branch.hotfix.prefix hotfix/
```

### Worktree Defaults
```bash
# Give every new feature its own worktree, without passing --worktree
git config gitflow.branch.feature.worktree true
```

`gitflow.branch.<type>.worktree` is a branch-type property, like `prefix` and `startPoint`, and it defaults to **false**. It applies to topic branch types only — git-flow writes it for those and not for base branches, where a worktree default would have no effect. **--worktree** and **--no-worktree** override it per command. Where the worktree ends up is a separate setting, `gitflow.worktreePath`; see **git-flow-worktree**(1).

### Command Overrides
```bash
# Never fetch before starting features (opt out of the default)
git config gitflow.feature.start.fetch false

# Never fetch before starting releases (opt out of the default)
git config gitflow.release.start.fetch false
```

## VALIDATION

The start command performs several validations:

**Branch name validation**
: Ensures the name is a valid Git reference

**Prefix handling**  
: Automatically adds the configured prefix if not already present

**Conflict detection**
: Prevents creating branches that already exist

**Base validation**
: Verifies the base commit/branch exists if specified

## WORKFLOW INTEGRATION

### Remote Workflow
```bash
# Start (fetches from the remote by default) and immediately publish
git flow feature start new-api
git push -u origin feature/new-api
```

### Team Workflow
```bash
# Start fetches by default, so you begin from the latest state
git flow feature start team-feature
```

## ENVIRONMENT

**GIT_FLOW_CD_FILE**
: When a worktree is created and this variable points at a writable file, start writes the worktree's absolute path there, so the calling shell can change directory to it. **--no-cd** suppresses the write; a start that creates no worktree never writes anything. See **git-flow-worktree**(1) for the full description of the channel.

## EXIT STATUS

**0**
: Successful branch creation

**1**
: git-flow is not initialized in this repository

**2**
: Invalid input — an unknown topic branch type, an empty branch name, or **--worktree** in a repository with no commits

**3**
: A Git operation failed, including a worktree that could not be created after the branch was

**4**
: Branch already exists

**5**
: The start point branch, tag, or commit does not exist

**6**
: A refusal: the target path of the new worktree is occupied

## SEE ALSO

**git-flow**(1), **git-flow-finish**(1), **git-flow-config**(1), **git-flow-list**(1), **git-flow-worktree**(1), **git-flow-checkout**(1), **git-flow-shell-init**(1), **gitflow-config**(5)

## NOTES

- Branch names should not include the prefix (it's added automatically)
- Start fetches by default so you begin from the latest changes; use **--no-fetch** (or `gitflow.<type>.start.fetch false`) to opt out
- Custom topic branch types work exactly like built-in types
- The base argument overrides the configured starting point for this specific branch
- Branch creation fails if a branch with the same full name already exists
- With a worktree the current worktree's HEAD is **unchanged**; without one the new branch is checked out as it always was
- An existing **empty** target directory is accepted; a file or a non-empty directory is refused
- A worktree created by start is recorded through its provenance marker, so **git flow worktree list** shows it as git-flow-created and later cleanup can recognize it
- When the target path is occupied **and** the branch already exists, the occupied-path refusal (exit **6**) is the one reported — clear the path first
- The EXIT STATUS codes above were corrected against the implementation; earlier versions of this page listed codes that the command never returned