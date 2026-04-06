# GIT-FLOW-TRACK(1)

## NAME

git-flow-track - Track a remote topic branch locally

## SYNOPSIS

**git-flow** *topic* **track** *name*

## DESCRIPTION

Creates a local branch that tracks a remote topic branch. This command is useful when you want to collaborate on a branch that was started by another team member. A configured remote is required; the command will fail with a clear error if no remote is available.

The command will:

1. Fetch the latest changes from the remote repository
2. Verify the branch exists on the remote
3. Create a local branch that tracks the remote branch
4. Check out the new local branch

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

*name*
: The name of the branch to track. Can be specified with or without the branch prefix.

## BRANCH NAME HANDLING

The track command handles branch names flexibly:

**Short name**
: `git flow feature track user-auth` tracks `feature/user-auth`

**Full name**
: `git flow feature track feature/user-auth` also tracks `feature/user-auth` (no double prefix)

## EXAMPLES

### Basic Usage

Track a remote feature branch:
```bash
git flow feature track user-authentication
```

Track a remote release branch:
```bash
git flow release track 1.0.0
```

Track a remote hotfix branch:
```bash
git flow hotfix track 1.0.1
```

### Using Full Branch Names

Track a branch by its full name:
```bash
git flow feature track feature/user-authentication
```

### Team Workflow

When a colleague starts a feature and you want to collaborate:
```bash
# Colleague creates and publishes the feature
git flow feature start shared-feature
git push -u origin feature/shared-feature

# You track the remote feature to work on it locally
git flow feature track shared-feature
```

## CONFIGURATION

The track command uses these configuration keys:

### Remote Name
```bash
# Set custom remote name (default is "origin")
git config gitflow.origin upstream
```

### Branch Prefixes
```bash
git config gitflow.branch.feature.prefix feature/
git config gitflow.branch.release.prefix release/
git config gitflow.branch.hotfix.prefix hotfix/
```

## VALIDATION

The track command performs several validations:

**Local branch check**
: Fails if a local branch with the same name already exists

**Remote branch check**
: Verifies the branch exists on the remote before creating local tracking branch

**Remote connectivity**
: Attempts to fetch from remote and reports clear errors on connection failures

## EXIT STATUS

**0**
: Successful branch tracking

**1**
: git-flow is not initialized

**2**
: Invalid input (missing branch name or invalid branch type)

**3**
: Git operation failed (fetch, checkout, etc.)

**4**
: Branch already exists locally

**5**
: Branch not found on remote

## SEE ALSO

**git-flow**(1), **git-flow-start**(1), **git-flow-finish**(1), **git-flow-update**(1)

## NOTES

- The local branch will have the same name as the remote branch
- If the branch already exists locally, the command will fail with an appropriate error
- Use `git flow <type> update` to sync with remote changes after tracking
- The command always fetches from the remote before checking for the branch
