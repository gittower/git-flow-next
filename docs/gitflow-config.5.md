# GITFLOW-CONFIG(5)

## NAME

gitflow-config - git-flow configuration file format and options reference

## DESCRIPTION

git-flow-next stores all configuration in Git's configuration system under the **gitflow.*** namespace. This manual page describes all available configuration options, their hierarchy, and usage patterns.

Configuration is stored in standard Git config files (**.git/config** for repository-specific, **~/.gitconfig** for user-global, **/etc/gitconfig** for system-wide).

## CONFIGURATION HIERARCHY

git-flow-next follows a strict three-layer configuration hierarchy:

### Layer 1: Branch Type Defaults
**gitflow.branch.*name*.*property***

Defines default behavior for branch types (base and topic branches).

### Layer 2: Command-Specific Overrides  
**gitflow.*branchtype*.*command*.*option***

Overrides branch defaults for specific commands and operations.

### Layer 3: Command-Line Flags
Command-line flags always take the highest precedence and override both configuration layers.

## GLOBAL CONFIGURATION

### Core Settings

**gitflow.version**
: Internal version marker for compatibility tracking. Set automatically during initialization.
: *Default*: "1.0"

**gitflow.initialized**  
: Marks repository as initialized with git-flow.
: *Default*: false

**gitflow.origin**, **gitflow.remote**
: Name of the remote repository to use for operations.
: *Default*: "origin"

## BRANCH CONFIGURATION

Branch configuration uses the pattern: **gitflow.branch.*name*.*property***

### Branch Properties

**type**
: Branch type. Values: **base**, **topic**
: *Required for all branches*

**parent**
: Parent branch name for hierarchical relationships.
: *Optional for base branches (trunk branches have no parent)*

**startPoint**  
: Branch to start new branches from (topic branches only).
: *Default*: Same as parent

**prefix**
: Prefix for branch names (topic branches only).
: *Default*: *branchname*/ (e.g., "feature/")

**upstreamStrategy**
: Merge strategy when merging TO parent branch.
: *Values*: **none**, **merge**, **rebase**, **squash**
: *Default*: **merge**

**downstreamStrategy**
: Merge strategy when merging FROM parent branch.  
: *Values*: **none**, **merge**, **rebase**
: *Default*: **merge**

**autoUpdate**
: Automatically update from parent branch (base branches only).
: *Default*: false

**tag**
: Create tags when finishing branches (topic branches only).
: *Default*: false

**tagprefix**
: Prefix for created tags.
: *Default*: "" (no prefix)

## COMMAND OVERRIDES

Command overrides use the pattern: **gitflow.*branchtype*.*command*.*option***

### Common Commands

**start**, **finish**, **update**, **delete**, **rename**, **publish**

### Common Options

**merge**
: Override upstream merge strategy for this command.
: *Values*: **merge**, **rebase**, **squash**

**fetch**
: Fetch from remote before operation.
: *Default*: false

**keep**
: Keep branch after finishing (finish command only).
: *Default*: false

**tag**, **notag**
: Force tag creation or skip tag creation (finish command only).

**sign**
: Sign tags with GPG (finish command only).
: *Default*: false

**signingkey**
: GPG key ID for tag signing.

**message**
: Custom message for tags.

**push-option**
: Push option to transmit to the server during publish (publish command only). This is a multi-value key; use `git config --add` to specify multiple options. CLI options are combined with config defaults (additive). Use `--no-push-option` flag to suppress config defaults.
: *Default*: none
: *Example*: `git config gitflow.feature.publish.push-option "ci.skip"`

### Push Option Examples

```bash
# Skip CI when publishing feature branches
git config gitflow.feature.publish.push-option "ci.skip"

# Auto-create merge request when publishing release branches (GitLab)
git config gitflow.release.publish.push-option "merge_request.create"
git config --add gitflow.release.publish.push-option "merge_request.target=main"

# Set topic for Gerrit when publishing hotfix branches
git config gitflow.hotfix.publish.push-option "%topic=hotfix"
```

## EXAMPLE CONFIGURATIONS

### Classic GitFlow

```ini
[gitflow]
    version = 1.0
    initialized = true

# Base branches
[gitflow "branch.main"]
    type = base
    upstreamstrategy = none
    downstreamstrategy = none
    
[gitflow "branch.develop"]
    type = base
    parent = main
    autoupdate = true
    upstreamstrategy = merge
    downstreamstrategy = merge

# Topic branches
[gitflow "branch.feature"]
    type = topic
    parent = develop
    prefix = feature/
    upstreamstrategy = merge
    downstreamstrategy = rebase
    
[gitflow "branch.release"]
    type = topic
    parent = main
    startpoint = develop
    prefix = release/
    upstreamstrategy = merge
    downstreamstrategy = merge
    tag = true
    tagprefix = v
    
[gitflow "branch.hotfix"]
    type = topic
    parent = main
    prefix = hotfix/
    upstreamstrategy = merge
    downstreamstrategy = merge
    tag = true
    tagprefix = v
```

### GitHub Flow

```ini
[gitflow]
    version = 1.0
    
[gitflow "branch.main"]
    type = base
    upstreamstrategy = none
    downstreamstrategy = none
    
[gitflow "branch.feature"]
    type = topic
    parent = main
    prefix = feature/
    upstreamstrategy = merge
    downstreamstrategy = rebase
```

### GitLab Flow

```ini
[gitflow]
    version = 1.0
    
[gitflow "branch.production"]
    type = base
    upstreamstrategy = none
    downstreamstrategy = none
    
[gitflow "branch.staging"]
    type = base
    parent = production
    upstreamstrategy = merge
    downstreamstrategy = merge
    
[gitflow "branch.main"]
    type = base
    parent = staging
    upstreamstrategy = merge
    downstreamstrategy = merge
    
[gitflow "branch.feature"]
    type = topic
    parent = main
    prefix = feature/
    upstreamstrategy = merge
    downstreamstrategy = rebase
    
[gitflow "branch.hotfix"]
    type = topic
    parent = production
    prefix = hotfix/
    upstreamstrategy = merge
    downstreamstrategy = merge
    tag = true
```

## COMMAND OVERRIDE EXAMPLES

### Feature Branch Overrides

```ini
# Merge strategy configuration for features
[gitflow "feature.finish"]
    rebase = true                # Always rebase features
    preserve-merges = false      # Flatten merges during rebase
    no-ff = true                 # Always create merge commits
    
# Alternative: Use squash merge for features
[gitflow "feature.finish"]
    squash = true                # Squash all commits into one
    
# Always fetch before starting features
[gitflow "feature.start"]  
    fetch = true
    
# Keep feature branches after finishing
[gitflow "feature.finish"]
    keep = true
```

### Release Branch Overrides

```ini
# Always sign release tags
[gitflow "release.finish"]
    sign = true
    signingkey = ABC123DEF456
    
# Use custom tag message pattern
[gitflow "release.finish"]
    message = "Release version %s"
    
# Always fetch before release operations
[gitflow "release"]
    fetch = true
```

### Hotfix Branch Overrides

```ini
# Always sign hotfix tags for auditability  
[gitflow "hotfix.finish"]
    sign = true
    signingkey = ABC123DEF456
    
# Use squash merge for hotfixes to keep clean history
[gitflow "hotfix.finish"]
    squash = true
    no-ff = false
```

## FINISH COMMAND CONFIGURATION

The finish command supports extensive merge strategy configuration through command-specific overrides.

### Merge Strategy Options

**gitflow.*type*.finish.rebase**
: Force rebase strategy for this branch type.
: *Type*: boolean
: *Default*: false

**gitflow.*type*.finish.no-rebase**
: Disable rebase strategy for this branch type.
: *Type*: boolean
: *Default*: false

**gitflow.*type*.finish.squash**
: Force squash merge strategy for this branch type.
: *Type*: boolean
: *Default*: false

**gitflow.*type*.finish.no-squash**
: Disable squash merge strategy for this branch type.
: *Type*: boolean
: *Default*: false

**gitflow.*type*.finish.preserve-merges**
: Preserve merges during rebase operations.
: *Type*: boolean
: *Default*: false

**gitflow.*type*.finish.no-preserve-merges**
: Flatten merges during rebase operations.
: *Type*: boolean
: *Default*: true

**gitflow.*type*.finish.no-ff**
: Force creation of merge commits (disable fast-forward).
: *Type*: boolean
: *Default*: false

**gitflow.*type*.finish.ff**
: Allow fast-forward merges when possible.
: *Type*: boolean
: *Default*: true

### Remote Fetch Options

**gitflow.*type*.finish.fetch**
: Fetch from remote before finishing a topic branch. When enabled, performs `git fetch <remote>` to update remote tracking branches before merging the topic branch into its parent.
: *Type*: boolean
: *Default*: false

### Strategy Precedence

1. **Command-line flags** (highest priority)
2. **gitflow.*type*.finish.*** configuration
3. **gitflow.branch.*type*.upstreamstrategy** (lowest priority)

### Examples

```ini
# Always rebase features with preserved merges
[gitflow "feature.finish"]
    rebase = true
    preserve-merges = true

# Squash merge for small fixes, no fast-forward for visibility
[gitflow "bugfix.finish"]
    squash = true
    no-ff = true

# Clean linear history for experimental branches
[gitflow "experimental.finish"]
    rebase = true
    preserve-merges = false
```

## MERGE STRATEGY REFERENCE

### none
No automatic merging. Manual merge required.

### merge
Standard Git merge creating merge commit.
- *Preserves branch history*
- *Shows clear integration points*
- *Good for release and hotfix branches*

### rebase  
Rebase branch onto target creating linear history.
- *Clean, linear history*
- *No merge commits*  
- *Good for feature branch updates*

### squash
Combine all branch commits into single commit.
- *Clean target branch history*
- *Loses individual commit history*
- *Good for small feature branches*

## GIT-FLOW-AVH COMPATIBILITY

git-flow-next automatically translates git-flow-avh configuration:

| AVH Configuration | git-flow-next Equivalent |
|-------------------|---------------------------|
| gitflow.branch.master | gitflow.branch.main.* |
| gitflow.branch.develop | gitflow.branch.develop.* |
| gitflow.prefix.feature | gitflow.branch.feature.prefix |
| gitflow.prefix.release | gitflow.branch.release.prefix |
| gitflow.prefix.hotfix | gitflow.branch.hotfix.prefix |
| gitflow.prefix.support | gitflow.branch.support.prefix |
| gitflow.prefix.versiontag | gitflow.branch.*.tagprefix |

Translation happens at runtime without modifying existing configuration.

## VALIDATION RULES

### Branch Names
- Must be valid Git reference names
- Cannot contain spaces or special characters (@, ~, ^, :, ?, *, [)
- Cannot end with .lock
- Cannot contain consecutive dots (..)

### Branch Relationships  
- Parent branches must exist before creating children
- No circular dependencies allowed
- Base branches can only have base or topic children
- Topic branches cannot have children

### Merge Strategies
- Must be valid strategy names
- **none** only valid for trunk branches
- **squash** not recommended for base branches

## PRECEDENCE EXAMPLES

Configuration with multiple layers:

```bash
# Layer 1: Branch default
git config gitflow.branch.feature.upstreamStrategy merge

# Layer 2: Command override (takes precedence)
git config gitflow.feature.finish.rebase true

# Layer 3: Command flag (highest precedence)  
git flow feature finish --squash
```

Final merge strategy: **squash** (from command flag)

## MIGRATION

### From git-flow-avh

git-flow-next automatically imports AVH configuration. No manual migration needed.

### Manual Migration

To migrate manually:

```bash
# Export AVH config
git config --get-regexp '^gitflow\.' > avh-config.txt

# Remove AVH config (optional)
git config --remove-section gitflow

# Initialize git-flow-next
git flow init --custom

# Recreate configuration using config commands
git flow config add base main
git flow config add base develop main --auto-update=true
git flow config add topic feature develop --prefix=feature/
```

## SECURITY CONSIDERATIONS

### GPG Signing
```ini
# Always sign tags for production workflows
[gitflow "release.finish"]
    sign = true
    signingkey = YOUR-GPG-KEY-ID
    
[gitflow "hotfix.finish"]  
    sign = true
    signingkey = YOUR-GPG-KEY-ID
```

### Remote Configuration
```ini
# Use specific remote for git-flow operations
[gitflow]
    origin = secure-origin
    
# Require fetch before operations to ensure latest state
[gitflow "feature.start"]
    fetch = true
[gitflow "release.start"]
    fetch = true  
[gitflow "hotfix.start"]
    fetch = true
```

## SEE ALSO

**git-flow**(1), **git-flow-config**(1), **git-flow-init**(1), **git-config**(1), **gitignore**(5)

## NOTES

- All configuration is stored in Git's configuration system
- Changes take effect immediately without restart
- Configuration is repository-specific by default
- Use **git config --global** for user-wide defaults
- Command overrides are more specific than branch defaults
- Validation occurs when configuration is accessed, not when set