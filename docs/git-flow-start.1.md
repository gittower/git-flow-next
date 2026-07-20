# GIT-FLOW-START(1)

## NAME

git-flow-start - Create and checkout topic branches

## SYNOPSIS

**git-flow** *topic* **start** [*name*] [*base*] [*options*]

## DESCRIPTION

Create and checkout a new topic branch of the specified type. This command works with any topic branch type (feature, release, hotfix, support, or custom types defined in your configuration).

The new branch is created from the configured starting point for the topic branch type, or from the specified base commit/branch if provided.

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

*name*
: Name of the new topic branch (without the prefix - that's added automatically). Optional: when omitted, git-flow runs the `filter-flow-<type>-start-version` filter with an empty version argument and uses its trimmed output as the branch name. If no such filter is configured or it yields no output, the command fails with `branch name cannot be empty`.

*base*
: Optional base commit, tag, or branch to start from instead of the configured starting point

## OPTIONS

**--fetch**
: Fetch from remote before creating branch to ensure latest state. If no remote is configured, the fetch is automatically skipped.

**--no-fetch**
: Don't fetch from remote before creating branch (default behavior)

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

### With Remote Synchronization

Fetch latest changes before starting:
```bash
git flow feature start new-api --fetch
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

### Command Overrides
```bash
# Always fetch before starting features
git config gitflow.feature.start.fetch true

# Always fetch before starting releases
git config gitflow.release.start.fetch true
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
# Start and immediately publish
git flow feature start new-api --fetch
git push -u origin feature/new-api
```

### Team Workflow
```bash
# Ensure latest state before starting
git flow feature start team-feature --fetch
```

## EXIT STATUS

**0**
: Successful branch creation

**1**
: Invalid topic branch type

**2**
: Branch already exists

**3**
: Invalid branch name

**4**
: Base commit/branch not found

**5**
: Git operation failed

## SEE ALSO

**git-flow**(1), **git-flow-finish**(1), **git-flow-config**(1), **git-flow-list**(1), **gitflow-config**(5)

## NOTES

- Branch names should not include the prefix (it's added automatically)
- Use **--fetch** in team environments to ensure you start from latest changes
- Custom topic branch types work exactly like built-in types
- The base argument overrides the configured starting point for this specific branch
- Branch creation fails if a branch with the same full name already exists