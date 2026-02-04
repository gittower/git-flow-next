# GIT-FLOW-CONFIG(1)

## NAME

git-flow-config - Manage git-flow configuration

## SYNOPSIS

**git-flow config** *command* [*args*] [*options*]

## DESCRIPTION

Manage git-flow configuration for base branches and topic branch types. The **config** command provides full CRUD operations for customizing your git-flow workflow.

**Base branches** are long-living branches like main, develop, staging, production that form the backbone of your workflow.

**Topic branches** are short-living branches like feature, release, hotfix that represent units of work.

## COMMANDS

### Listing Configuration

**list**
: Display current git-flow configuration showing branch hierarchy and settings

### Adding Configuration

**add base** *name* [*parent*] [*options*]
: Add a base branch configuration. Creates the Git branch immediately if it doesn't exist.

**add topic** *name* *parent* [*options*]  
: Add a topic branch type configuration. Saves configuration for use with start command.

### Editing Configuration

**edit base** *name* [*options*]
: Edit an existing base branch configuration

**edit topic** *name* [*options*]
: Edit an existing topic branch type configuration

### Renaming

**rename base** *old-name* *new-name*
: Rename a base branch in both configuration and Git. Updates all dependent references.

**rename topic** *old-name* *new-name*
: Rename a topic branch type configuration. Does not affect existing branches.

### Deleting Configuration

**delete base** *name*
: Delete a base branch configuration. Keeps the Git branch but removes git-flow management.

**delete topic** *name*
: Delete a topic branch type configuration. Does not affect existing branches of this type.

## COMMAND OPTIONS

### Add Base Branch (`add base`)

**--upstream-strategy**=*strategy*
: Merge strategy when merging to parent. Values: **merge**, **rebase**, **squash**

**--downstream-strategy**=*strategy*
: Merge strategy when updating from parent. Values: **merge**, **rebase**

**--auto-update**[=*bool*]
: Auto-update from parent on finish. Default: **false**

### Add Topic Branch (`add topic`)

**--prefix**=*prefix*
: Branch name prefix. Default: *name*/ (e.g., "feature/")

**--starting-point**=*branch*
: Branch to create from (defaults to parent)

**--upstream-strategy**=*strategy*
: Merge strategy when merging to parent. Values: **merge**, **rebase**, **squash**

**--downstream-strategy**=*strategy*
: Merge strategy when updating from parent. Values: **merge**, **rebase**

**--tag**[=*bool*]
: Create tags on finish. Default: **false**

### Edit Base Branch (`edit base`)

Same options as `add base`:
- **--upstream-strategy**, **--downstream-strategy**, **--auto-update**

### Edit Topic Branch (`edit topic`)

Same options as `add topic`:
- **--prefix**, **--starting-point**, **--upstream-strategy**, **--downstream-strategy**, **--tag**

### Rename and Delete Commands

The following commands take only positional arguments and have no options:
- **`rename base`** *old-name* *new-name*
- **`rename topic`** *old-name* *new-name*  
- **`delete base`** *name*
- **`delete topic`** *name*

## MERGE STRATEGIES

**merge**
: Standard Git merge creating a merge commit

**rebase**
: Rebase branch onto target, creating linear history

**squash** 
: Squash all commits into a single commit when merging

**none**
: No automatic merging (manual merge required)

## EXAMPLES

### Base Branch Management

Add a production trunk branch:
```bash
git flow config add base production
```

Add staging branch that auto-updates from production:
```bash  
git flow config add base staging production --auto-update=true
```

Edit develop branch to disable auto-update:
```bash
git flow config edit base develop --auto-update=false
```

Rename develop branch to integration:
```bash
git flow config rename base develop integration
```

### Topic Branch Management

Add a feature branch type:
```bash
git flow config add topic feature develop --prefix=feat/
```

Add release branch type with tagging:
```bash
git flow config add topic release main --starting-point=develop --tag=true
```

Add bugfix branch type with squash merging:
```bash
git flow config add topic bugfix develop --upstream-strategy=squash --prefix=bug/
```

Edit feature branches to use rebase when finishing:
```bash
git flow config edit topic feature --upstream-strategy=rebase
```

### Complex Workflow Setup

Set up GitLab Flow workflow:
```bash
# Base branches
git flow config add base production
git flow config add base staging production  
git flow config add base main staging

# Topic branches  
git flow config add topic feature main --prefix=feature/
git flow config add topic hotfix production --starting-point=production --tag=true
```

## CONFIGURATION HIERARCHY

git-flow-next follows a three-layer configuration hierarchy:

1. **Branch defaults** (gitflow.branch.*) - Default behavior defined here
2. **Command overrides** (gitflow.*type*.*command*.*) - Override defaults for specific operations
3. **Command-line flags** - Always take precedence

Example showing precedence:
```bash
# Layer 1: Branch default
git config gitflow.branch.release.tag true

# Layer 2: Command override (takes precedence over Layer 1)
git config gitflow.release.finish.notag true

# Layer 3: Command flag (takes precedence over both)
git flow release finish --tag  # Forces tag creation
```

## COMMAND-SPECIFIC CONFIGURATION

Command-specific options are configured via git config at Layer 2:

### Publish Command Options

**gitflow.*type*.publish.push-option**
: Push options to pass to git push (supports multiple values). Example:
```bash
# Single push option
git config gitflow.feature.publish.push-option "ci.skip"

# Multiple push options (use --add for additional values)
git config gitflow.release.publish.push-option "merge_request.create"
git config --add gitflow.release.publish.push-option "merge_request.target=main"
```

### Finish Command Options

**gitflow.*type*.finish.fetch**
: Fetch from remote before finishing. Default: **false**

**gitflow.*type*.finish.notag**
: Disable tag creation (overrides branch.*.tag setting)

**gitflow.*type*.finish.sign**
: Sign the tag with GPG

**gitflow.*type*.finish.signingkey**
: GPG key to use for signing

**gitflow.*type*.finish.keep**
: Keep branch after finishing

**gitflow.*type*.finish.keepremote**
: Keep remote branch after finishing

**gitflow.*type*.finish.keeplocal**
: Keep local branch after finishing

**gitflow.*type*.finish.rebase**
: Use rebase strategy when finishing

**gitflow.*type*.finish.squash**
: Use squash strategy when finishing

## VALIDATION

The config command performs validation to ensure:

- **Valid Git ref names** - Branch names must be valid Git references
- **No circular dependencies** - Parent relationships cannot form cycles  
- **Parent existence** - Parent branches must exist before creating children
- **Valid strategies** - Merge strategies must be recognized values
- **No conflicts** - Branch names must be unique across types

## STORAGE

All configuration is stored in **.git/config** under the **gitflow.*** namespace:

```ini
[gitflow]
    version = 1.0
    initialized = true
[gitflow "branch.main"]
    type = base
    upstreamstrategy = none  
    downstreamstrategy = none
[gitflow "branch.develop"]
    type = base
    parent = main
    autoupdate = true
[gitflow "branch.feature"]
    type = topic
    parent = develop
    prefix = feature/
    upstreamstrategy = merge
    downstreamstrategy = rebase
```

## EXIT STATUS

**0**
: Successful operation

**1**
: Repository not initialized with git-flow

**2**
: Branch not found or invalid name

**3**
: Validation error (circular dependency, invalid strategy, etc.)

**4**
: Git operation failed

## SEE ALSO

**git-flow**(1), **git-flow-init**(1), **gitflow-config**(5), **git-config**(1)

## NOTES

- Configuration changes take effect immediately
- Base branches are created automatically when added
- Topic branch configurations are templates for the **start** command
- Delete operations preserve Git branches, only removing git-flow management
- Renaming base branches updates both Git and all configuration references
