# GITFLOW-CONFIG(5)

## NAME

gitflow-config - git-flow configuration file format and options reference

## DESCRIPTION

git-flow-next stores all configuration in Git's configuration system under the **gitflow.*** namespace. This manual page describes all available configuration options, their hierarchy, and usage patterns.

Configuration is stored in standard Git config files (**.git/config** for repository-specific, **~/.gitconfig** for user-global, **/etc/gitconfig** for system-wide).

## CONFIGURATION SCOPE

git-flow-next follows Git's native configuration scope precedence:

1. **Local** (**.git/config**) - Highest priority, repository-specific
2. **Global** (**~/.gitconfig**) - User-wide defaults
3. **System** (**/etc/gitconfig**) - System-wide defaults, lowest priority

When configuration exists in multiple scopes, git-flow uses the value from the highest-priority scope where it is found.

### Initialization Scope Control

The **git flow init** command supports scope options to control where configuration is stored:

- **--local** - Store in repository's **.git/config** (default)
- **--global** - Store in user's **~/.gitconfig**
- **--system** - Store in system-wide **/etc/gitconfig**
- **--file**=*path* - Store in specified file

### Scope Behavior

When checking initialization status:

- **Without scope flag**: Checks merged config (local > global > system). If configuration is found in a non-local scope, git-flow reports which scope contains the configuration and suggests using **--local** to create repository-specific config.

- **With explicit scope flag**: Checks only that specific scope. This allows initializing local config even when global config exists.

### Runtime Behavior

All runtime commands (start, finish, update, etc.) always read from merged configuration using Git's standard precedence. The scope options only affect the **init** command.

### Multi-Scope Use Cases

**Team Defaults**: Store common configuration globally, override per-repository:

```bash
# Set team defaults globally
git flow init --defaults --global

# Override specific settings in a repository
git flow init --defaults --local
```

**Shared Configuration**: Use **--file** for configuration shared across systems:

```bash
git flow init --defaults --file=/shared/team-gitflow.config
```

## CONFIGURATION HIERARCHY

git-flow-next follows a strict three-layer configuration hierarchy:

### Layer 1: Branch Type Definition
**gitflow.branch.*name*.*property***

Defines the **identity and process characteristics** of a branch type — what it *is* and how it participates in the workflow. This includes structural properties (type, parent, prefix) and process characteristics (merge strategies, tagging, auto-update). For example, `tag=true` on a release branch means "releases are the kind of branch that produces tags on finish" — it describes the release process, not merely a tunable default.

Layer 1 is reserved for essential branch-type configuration only.

### Layer 2: Command-Specific Configuration
**gitflow.*branchtype*.*command*.*option***

Controls **how commands execute** for a branch type. These are operational settings (fetch, sign, keep, push-options) that adjust command behavior without changing the branch type's identity. Some options at this layer can override Layer 1 process characteristics (e.g., `notag` overrides the branch type's `tag` setting).

Many command options exist only at Layer 2 — they have no Layer 1 equivalent because they don't describe a branch type characteristic.

### Layer 3: Command-Line Flags
Command-line flags always take the highest precedence and override both configuration layers. Use these for one-off overrides.

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

## PATH CONFIGURATION

**gitflow.path.hooks**
: Override the directory where git-flow looks for hook and filter scripts. When set, git-flow uses this directory instead of the default `.git/hooks/` or Git's `core.hooksPath`. Supports absolute paths or paths relative to the repository root.
: *Default*: not set (falls back to `core.hooksPath`, then `.git/hooks`)
: *Compatibility*: matches git-flow-avh `gitflow.path.hooks` behavior

```bash
# Absolute path
git config gitflow.path.hooks /shared/team-hooks

# Relative path (resolved from repository root)
git config gitflow.path.hooks .githooks

# Global default for all repositories
git config --global gitflow.path.hooks .githooks
```

## BRANCH CONFIGURATION

Branch configuration uses the pattern: **gitflow.branch.*name*.*property***

These properties define the branch type's identity and process characteristics (Layer 1). They describe *what the branch type is*, not how individual commands behave.

### Branch Name Case Sensitivity

Branch identities (the *name* subsection of `gitflow.branch.<name>`) are matched **case-insensitively**, following the same principle as Git's own **core.ignorecase**. The original case you use when a branch is first added is preserved as its **canonical** form and written back verbatim; git-flow always operates on that canonical name — including the underlying Git ref — so operations behave correctly on case-sensitive filesystems.

Any later reference to that branch may be given in any case. For example, after `git flow config add base V9_Release`, all of the following resolve to the single canonical `V9_Release` entry:

```
git flow config add base V10_Release v9_release   # parent given in lowercase
git flow config edit base v9_RELEASE --upstream-strategy rebase
git flow config delete base V9_release
```

Adding or renaming to a name that differs only in case from a *different* existing branch is rejected as an "already exists" condition, naming the existing canonical form. Renaming a branch to a case-only variant of *itself* (e.g., `V9_Release` → `v9_release`) is allowed and re-canonicalizes the entry in place.

Property names (the last segment, e.g., **upstreamStrategy**) are case-insensitive in Git config and are always read regardless of the case they were stored in.

Branch names may contain dots (`.`) in the subsection (e.g., `custom.main`, `V10.5`). Git keeps the section and the variable name (the last segment) dot-free, so git-flow reads the property from the final segment and reconstructs the branch name from every segment before it — the full dotted name round-trips correctly.

**Limitations**:

- Two branch types whose names differ only in case cannot coexist as distinct entities — they are treated as one branch identity.
- Names that Git itself rejects as refs are still invalid (see [VALIDATION RULES](#validation-rules)) — a leading dot, consecutive dots (`..`), or a trailing `.lock`.

### Structural Properties

Define where the branch type fits in the hierarchy:

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

### Process Characteristics

Define how the branch type participates in the workflow:

**upstreamStrategy**
: How changes flow TO parent branch on finish.
: *Values*: **none**, **merge**, **rebase**, **squash**
: *Default*: **merge**

**downstreamStrategy**
: How updates flow FROM parent branch.
: *Values*: **none**, **merge**, **rebase**
: *Default*: **merge**

**autoUpdate**
: Branch automatically receives updates from parent on finish (base branches only).
: *Default*: false

**tag**
: Branch type produces tags on finish (topic branches only). Setting this to **true** means the branch type's process includes tagging — e.g., releases and hotfixes produce tags as part of their workflow.
: *Default*: false

**tagprefix**
: Prefix for created tags (topic branches only).
: *Default*: "" (no prefix)

## COMMAND OVERRIDES

Command overrides (Layer 2) control **how commands execute** for a branch type, using the pattern: **gitflow.*branchtype*.*command*.*option***

These are operational settings that adjust command behavior. Some can override Layer 1 process characteristics (e.g., **notag** overrides a branch type's **tag** setting), while others exist only at this layer.

### Common Commands

**start**, **finish**, **update**, **delete**, **rename**, **publish**

### Common Options

**merge**
: Override upstream merge strategy for this command.
: *Values*: **merge**, **rebase**, **squash**

**fetch**
: Fetch from remote before the operation. This is a Layer 2 operational setting only (there is no Layer 1 branch-type knob). Resolution is uniform across commands: per-command default, then `gitflow.<type>.<cmd>.fetch`, then the `--fetch`/`--no-fetch` CLI flag (which always wins).
: *Default*: **true** for `finish` (so the sync check runs against fresh data), **false** for `start`.

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

[gitflow "branch.hotfix"]
    type = topic
    parent = main
    prefix = hotfix/
    upstreamstrategy = merge
    downstreamstrategy = merge
    tag = true
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

### Custom Merge Message Overrides

```ini
# Feature branches: conventional commit format for merges
[gitflow "feature.finish"]
    mergemessage = "feat: merge %b into %p"

# Release branches: sync message for develop updates
[gitflow "release.finish"]
    mergemessage = "release: merge %b into %p"
    updatemessage = "chore: sync %b from main after release"

# Hotfix branches: clear tracking of hotfix merges
[gitflow "hotfix.finish"]
    mergemessage = "fix: merge %b into %p"
    updatemessage = "chore: sync %b with hotfix changes"
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
: Fetch from remote before finishing a topic branch. When enabled, fetches the base branch (best-effort) and the topic branch from the remote to ensure the latest remote state is known before merging. A failure fetching the topic branch against a reachable-but-failing remote is fatal; a remote with no ref for the topic (never pushed, or deleted after a remote merge) is benign and its sync check is skipped.
: After fetching, if the local topic branch is ahead of, behind, or diverged from its remote tracking branch, the finish operation aborts with an error to prevent accidental data loss or merging unpublished work. Use `--force` to bypass this safety check. `--no-fetch` skips only the fetch; the sync check still runs against existing tracking data.
: *Type*: boolean
: *Default*: true

**gitflow.*type*.start.fetch**
: Fetch from remote before starting a topic branch. When no remote is configured, the fetch is skipped silently. A fetch failure is a non-fatal warning — the branch is still created (start has no sync gate).
: *Type*: boolean
: *Default*: false

**gitflow.*type*.delete.fetch**
: Fetch from remote before deleting a topic branch. When enabled, updates local refs so that Git can correctly detect whether the branch has been merged remotely (e.g., via a GitHub PR merge), avoiding the need for `--force`. The parent is fast-forwarded from its remote only when it is the branch currently checked out (`git merge --ff-only` acts on HEAD); delete auto-checks-out the parent when you delete the branch you are on, which is the common case. When no remote is configured, the fetch is skipped silently. A fetch failure against an unreachable remote is a non-fatal note — deletion is not blocked by the fetch itself, but the topic sync check still runs against existing local tracking data and can still abort (behind/diverged) unless `--force` is given. `--no-fetch` skips only the fetch; it does not skip the sync check.
: *Type*: boolean
: *Default*: false

### Remote Push Options

**gitflow.*type*.finish.push**
: Push the results of a completed finish to the configured remote (`gitflow.remote`, default `origin`) as a final stage. When enabled, the target (parent) branch and each auto-updated child base branch are pushed (parent first), and — unless overridden by the `pushtag` key — the created tag is pushed too. The finished topic branch itself is never pushed. If no remote is configured, the push stage is skipped with a note and finish still succeeds. A rejected (non-fast-forward) push causes finish to exit with an error, leaving the already-completed local finish untouched. Corresponds to `--push`/`--no-push`.
: *Type*: boolean
: *Default*: false

**gitflow.*type*.finish.pushtag**
: Push the created tag on finish. When this key is unset, tag pushing follows the resolved `push` decision, so enabling `push` also pushes the tag. Set this key explicitly to decouple the tag from the branch decision: `true` pushes the tag even when branches are not pushed, `false` suppresses the tag push even when branches are. Has no effect when no tag was created. Corresponds to `--pushtag`/`--no-pushtag`.
: *Type*: boolean
: *Default*: (follows `gitflow.*type*.finish.push`)

### Merge Message Options

**gitflow.*type*.finish.mergemessage**
: Custom commit message for upstream merge (topic → parent). Supports placeholders: `%b` (branch name), `%B` (full refname), `%p` (parent branch), `%P` (full parent refname), `%%` (literal percent).
: *Type*: string
: *Default*: (none, uses Git's default merge message)

**gitflow.*type*.finish.updatemessage**
: Custom commit message for child branch updates (parent → child). Supports the same placeholders as mergemessage.
: *Type*: string
: *Default*: (none, uses auto-generated message)

**--squash-message** (CLI-only)
: Custom commit message for squash merges. This option has no git config equivalent, as squash messages are specific to each branch being finished.

These options are useful for teams using commit message validation hooks (e.g., conventional commits) where auto-generated messages would be rejected.

### Hook Control Options

**gitflow.*type*.finish.noverify**
: Bypass pre-commit and commit-msg hooks during merge and commit operations. When enabled, passes `--no-verify` to the underlying `git merge` and `git commit` commands. This is useful in CI/CD pipelines or when hooks would interfere with automated workflows. The setting is persisted through `--continue` operations after conflict resolution.
: *Type*: boolean
: *Default*: false

### Strategy Precedence

1. **Command-line flags** (Layer 3 — highest priority, one-off overrides)
2. **gitflow.*type*.finish.*** configuration (Layer 2 — operational command settings)
3. **gitflow.branch.*type*.upstreamstrategy** (Layer 1 — branch type process characteristic)

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

## INTEGRATE COMMAND CONFIGURATION

The **git-flow-integrate**(1) command reads its operational defaults from the
`gitflow.<branch>.integrate.*` namespace, keyed by the **base-branch name**
(the same identifier used as the `gitflow.branch.<name>.*` subsection). Integrate
applies only to base branches.

The available keys mirror the finish command's merge-strategy, tag, message, and
fetch options, but the Layer-1 defaults differ.

### Layer-1 Defaults

Unlike finish, integrate defaults tagging and fetching **off**:

- **Tagging is off by default.** Base branches have no version from which to
  derive a tag name, so tagging must be enabled explicitly (via `--tag <name>`
  or `integrate.tag` + `integrate.tagname`). Enabling tagging without a
  resolvable name is an error.
- **Fetching is off by default** (opt-in), unlike finish where fetch defaults on.

### Tag Options

**gitflow.*branch*.integrate.tag**
: Enable tag creation on integrate (boolean; default false).

**gitflow.*branch*.integrate.tagname**
: Tag name to create. Required when tagging is enabled and no `--tag` is given.

**gitflow.*branch*.integrate.sign**
: Sign the tag cryptographically (boolean).

**gitflow.*branch*.integrate.signingkey**
: GPG key id to use for the signature.

**gitflow.*branch*.integrate.messagefile**
: Read the tag message from the given file.

### Merge Strategy Options

**gitflow.*branch*.integrate.squash**, **gitflow.*branch*.integrate.no-squash**
: Force (or disable) a squash merge.

**gitflow.*branch*.integrate.rebase**, **gitflow.*branch*.integrate.no-rebase**
: Force (or disable) rebasing the base branch before merging.

**gitflow.*branch*.integrate.preserve-merges**, **gitflow.*branch*.integrate.no-preserve-merges**
: Preserve or flatten merges during rebase.

**gitflow.*branch*.integrate.no-ff**, **gitflow.*branch*.integrate.ff**
: Force a merge commit for a fast-forward, or allow fast-forward.

### Message and Fetch Options

**gitflow.*branch*.integrate.mergemessage**
: Custom commit message for the upstream merge.

**gitflow.*branch*.integrate.updatemessage**
: Custom commit message for child branch updates.

**gitflow.*branch*.integrate.fetch**
: Fetch from the remote before integrating (boolean; default false).

### Strategy Precedence

1. **Command-line flags** (Layer 3 — highest priority)
2. **gitflow.*branch*.integrate.*** configuration (Layer 2)
3. **gitflow.branch.*branch*.upstreamStrategy** (Layer 1)

### Examples

```ini
# Tag main whenever develop is integrated
[gitflow "develop.integrate"]
    tag = true
    tagname = latest

# Rebase a private staging branch onto main and fetch first
[gitflow "staging.integrate"]
    rebase = true
    fetch = true
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

Branch names are validated with **git check-ref-format**, so they follow Git's
own reference-name rules:

- May contain dots (`.`) inside the name (e.g., `custom.main`, `V10.5`)
- Cannot begin a path component with a dot, or contain consecutive dots (`..`)
- Cannot end with `/` or `.lock`
- Cannot contain spaces, control characters, or any of `~ ^ : ? * [ \`
- Cannot begin with `-` — a git-flow restriction beyond Git's own rules, since
  branch names are passed as bare operands to commands like `git checkout -b` /
  `git branch -m`, where a leading dash would be misparsed as an option

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
# Layer 1: Branch type process characteristic — "features merge into parent"
git config gitflow.branch.feature.upstreamStrategy merge

# Layer 2: Command-specific override — "but finish uses rebase"
git config gitflow.feature.finish.rebase true

# Layer 3: CLI flag — "this time, squash instead"
git flow feature finish --squash
```

Final merge strategy: **squash** (Layer 3 CLI flag always wins)

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
