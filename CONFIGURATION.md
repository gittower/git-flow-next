# Configuration Reference

This document provides a comprehensive reference for all git-flow-next configuration options. All configuration is stored in Git configuration using the `gitflow.*` key namespace.

## Configuration Overview

git-flow-next uses a hierarchical configuration system with **three levels of precedence**:

1. **Branch Type Definition** (`gitflow.branch.*`) - The identity and process characteristics of a branch type
2. **Command-Specific Configuration** (`gitflow.<branchtype>.*`) - How specific commands behave for a branch type
3. **Command-Line Flags** - **Highest priority** - one-off overrides that always win

### Understanding the Layers

**Layer 1 defines what a branch type *is*** — its role in the workflow, not tunable defaults. These properties describe the branch type's identity (where it lives in the hierarchy, its naming convention) and its process characteristics (how it participates in the workflow). For example, `tag=true` on a release branch means "releases are the kind of branch that produces tags on finish" — it describes the release process, not merely a default that you toggle.

**Layer 2 controls how commands execute** — operational settings like whether to fetch before an operation, sign tags, or keep branches after finishing. These are genuine behavioral knobs that adjust command execution without changing what the branch type fundamentally is.

**Layer 3 provides one-off overrides** via CLI flags for ad-hoc situations.

Many command options intentionally have no Layer 1 equivalent — they exist only in Layer 2 and Layer 3. For example, publish `push-options` are purely operational and don't define any branch type characteristic.

> **For developers adding new options**: Most new options need only Layer 2 + Layer 3. Add Layer 1 support only when the option defines a branch type characteristic. See [CODING_GUIDELINES.md](CODING_GUIDELINES.md) → "Adding New Configuration Options" for the full decision guide and checklist.

## Core System Configuration

### System Settings

| Key | Description | Default | Example |
|-----|-------------|---------|---------|
| `gitflow.version` | Internal version marker for compatibility | `1.0` | `1.0` |
| `gitflow.initialized` | Marks repository as git-flow initialized | `false` | `true` |
| `gitflow.origin` | Remote name to use for operations | `origin` | `upstream` |
| `gitflow.remote` | Alias for `gitflow.origin` | `origin` | `upstream` |

## Branch Type Configuration (Layer 1)

Branch type configuration defines the **identity and process characteristics** of each branch type using the pattern:
`gitflow.branch.<branchname>.<property>`

These properties describe *what the branch type is* — its structural role and how it participates in the workflow. They should only contain essential branch-type-relevant configuration.

### Structural Properties

Define where the branch type fits in the hierarchy:

| Property | Description | Values | Default |
|----------|-------------|--------|---------|
| `type` | Branch type classification | `base`, `topic` | Required |
| `parent` | Parent branch for this branch type | Branch name | Varies by type |
| `startPoint` | Branch to start new branches from | Branch name | Same as parent |
| `prefix` | Branch name prefix (topic only) | String ending in `/` | `<type>/` |

### Process Characteristics

Define how the branch type participates in the workflow:

| Property | Description | Values | Default |
|----------|-------------|--------|---------|
| `upstreamStrategy` | How changes flow TO parent on finish | `merge`, `rebase`, `squash` | `merge` |
| `downstreamStrategy` | How updates flow FROM parent | `merge`, `rebase` | `merge` |
| `tag` | Branch type produces tags on finish (topic only) | `true`, `false` | `false` |
| `tagprefix` | Prefix for created tags (topic only) | String | `""` |
| `autoUpdate` | Auto-update from parent on finish (base only) | `true`, `false` | `false` |
| `deleteRemote` | Delete remote branch on finish (topic only) | `true`, `false` | `false` |

For example, setting `tag=true` on release branches means "releases produce tags" — it characterizes the release process. This can still be overridden per-command (Layer 2: `gitflow.release.finish.notag`) or per-invocation (Layer 3: `--notag`), but the branch config establishes the branch type's intended role.

### Default Branch Types

#### Base Branches

```bash
# Main branch (production releases)
gitflow.branch.main.type=base
gitflow.branch.main.parent=             # No parent (root)
gitflow.branch.main.upstreamStrategy=merge
gitflow.branch.main.downstreamStrategy=merge
gitflow.branch.main.autoUpdate=false

# Develop branch (integration)  
gitflow.branch.develop.type=base
gitflow.branch.develop.parent=main
gitflow.branch.develop.startPoint=main
gitflow.branch.develop.upstreamStrategy=merge
gitflow.branch.develop.downstreamStrategy=merge
gitflow.branch.develop.autoUpdate=true    # Auto-updates from main
```

#### Topic Branches

```bash
# Feature branches
gitflow.branch.feature.type=topic
gitflow.branch.feature.parent=develop
gitflow.branch.feature.startPoint=develop
gitflow.branch.feature.prefix=feature/
gitflow.branch.feature.upstreamStrategy=merge
gitflow.branch.feature.downstreamStrategy=rebase
gitflow.branch.feature.tag=false
gitflow.branch.feature.autoUpdate=false

# Release branches
gitflow.branch.release.type=topic
gitflow.branch.release.parent=main
gitflow.branch.release.startPoint=develop
gitflow.branch.release.prefix=release/
gitflow.branch.release.upstreamStrategy=merge
gitflow.branch.release.downstreamStrategy=merge
gitflow.branch.release.tag=true
gitflow.branch.release.autoUpdate=false

# Hotfix branches  
gitflow.branch.hotfix.type=topic
gitflow.branch.hotfix.parent=main
gitflow.branch.hotfix.startPoint=main
gitflow.branch.hotfix.prefix=hotfix/
gitflow.branch.hotfix.upstreamStrategy=merge
gitflow.branch.hotfix.downstreamStrategy=merge
gitflow.branch.hotfix.tag=true
gitflow.branch.hotfix.autoUpdate=false
```

#### Custom Branch Types

You can define custom branch types by setting the appropriate configuration:

```bash
# Example: Support branches
gitflow.branch.support.type=topic
gitflow.branch.support.parent=main
gitflow.branch.support.startPoint=main
gitflow.branch.support.prefix=support/
gitflow.branch.support.tag=true
gitflow.branch.support.tagprefix=support-
```

## Command-Specific Configuration (Layer 2)

Command-specific configuration controls **how commands execute** for a branch type, using the pattern:
`gitflow.<branchtype>.<command>.<option>`

These are operational settings — they adjust command behavior without changing the branch type's identity. Some of these can override Layer 1 process characteristics (e.g., `notag` overrides the branch type's `tag` setting), while others exist only at this layer (e.g., `sign`, `keep`, `fetch`).

### Finish Command Options

The finish command supports per-branch-type configuration:

| Option | Description | Values | Default |
|--------|-------------|--------|---------|
| `merge` | Override merge strategy | `merge`, `rebase`, `squash` | From branch config |
| `notag` | Disable tag creation | `true`, `false` | Opposite of branch `tag` |
| `sign` | Sign created tags | `true`, `false` | `false` |
| `signingkey` | GPG key for signing | Key ID | Git default |
| `messagefile` | File containing tag message | File path | None |
| `keep` | Keep branch after finish | `true`, `false` | `false` |
| `keepremote` | Keep remote branch | `true`, `false` | `false` |
| `keeplocal` | Keep local branch | `true`, `false` | `false` |
| `force-delete` | Force delete branch | `true`, `false` | `false` |
| `fetch` | Fetch before operation | `true`, `false` | `false` |

#### Examples

```bash
# Override feature finish to use rebase
gitflow.feature.finish.merge=rebase

# Disable tags for feature branches
gitflow.feature.finish.notag=true

# Sign all release tags
gitflow.release.finish.sign=true
gitflow.release.finish.signingkey=ABC123DEF

# Keep hotfix branches locally after finish
gitflow.hotfix.finish.keeplocal=true
```

### Update Command Options

| Option | Description | Values | Default |
|--------|-------------|--------|---------|
| `downstreamStrategy` | Override update strategy | `merge`, `rebase` | From branch config |

#### Examples

```bash
# Force rebase for feature updates
gitflow.feature.downstreamStrategy=rebase

# Use merge for release updates
gitflow.release.downstreamStrategy=merge
```

## Configuration Precedence

git-flow-next follows a strict three-layer precedence hierarchy:

### Layer 1: Branch Type Definition (Lowest Priority)
Defines the branch type's identity and process characteristics via `gitflow.branch.<branchtype>.*`
```bash
gitflow.branch.release.tag=true    # "Releases produce tags" — a process characteristic
```

### Layer 2: Command-Specific Configuration (Medium Priority)
Operational settings that control how commands execute via `gitflow.<branchtype>.<command>.*`
```bash
gitflow.release.finish.notag=true  # Override: skip tagging for this command
```

### Layer 3: Command-Line Flags (Highest Priority)
One-off overrides that **always win**
```bash
git flow release finish v1.0 --tag  # Force tagging for this invocation
```

Not every option spans all three layers. Layer 1 is reserved for essential branch-type properties — many command options exist only at Layer 2 and Layer 3. For example, `sign`, `keep`, and publish `push-options` are purely operational and have no Layer 1 equivalent.

## Legacy Compatibility

### Git-Flow-AVH Compatibility

git-flow-next automatically imports configuration from git-flow-avh installations:

| AVH Key | git-flow-next Equivalent | Description |
|---------|--------------------------|-------------|
| `gitflow.branch.master` | `gitflow.branch.main` | Main branch name |
| `gitflow.branch.develop` | `gitflow.branch.develop` | Develop branch name |
| `gitflow.prefix.feature` | `gitflow.branch.feature.prefix` | Feature prefix |
| `gitflow.prefix.release` | `gitflow.branch.release.prefix` | Release prefix |
| `gitflow.prefix.hotfix` | `gitflow.branch.hotfix.prefix` | Hotfix prefix |
| `gitflow.prefix.support` | `gitflow.branch.support.prefix` | Support prefix |
| `gitflow.prefix.versiontag` | `gitflow.branch.*.tagprefix` | Tag prefix |

### Migration from git-flow-avh

1. **Automatic Import**: Run `git flow init` to automatically import existing configuration
2. **Manual Migration**: Use the mapping table above to convert AVH config to git-flow-next format
3. **Verification**: Run `git flow overview` to verify imported configuration

## Configuration Examples

### Simple GitHub Flow

```bash
# Minimal configuration - feature branches only
git config gitflow.branch.main.type base
git config gitflow.branch.feature.type topic
git config gitflow.branch.feature.parent main
git config gitflow.branch.feature.prefix feature/
```

### Traditional Git-Flow

```bash
# Full git-flow with main, develop, and all topic types
git config gitflow.branch.main.type base
git config gitflow.branch.develop.type base
git config gitflow.branch.develop.parent main
git config gitflow.branch.develop.autoUpdate true

git config gitflow.branch.feature.type topic
git config gitflow.branch.feature.parent develop
git config gitflow.branch.feature.prefix feature/

git config gitflow.branch.release.type topic
git config gitflow.branch.release.parent main
git config gitflow.branch.release.startPoint develop
git config gitflow.branch.release.tag true

git config gitflow.branch.hotfix.type topic
git config gitflow.branch.hotfix.parent main
git config gitflow.branch.hotfix.tag true
```

### Custom Workflow

```bash
# Custom branch types with specific merge strategies
git config gitflow.branch.bugfix.type topic
git config gitflow.branch.bugfix.parent develop
git config gitflow.branch.bugfix.prefix bugfix/
git config gitflow.branch.bugfix.upstreamStrategy squash

# Custom finish behavior
git config gitflow.bugfix.finish.notag true
git config gitflow.bugfix.finish.keep true
```

## Configuration Management

### Viewing Configuration

```bash
# View all git-flow configuration
git config --get-regexp '^gitflow\.'

# View branch-specific configuration
git config --get-regexp '^gitflow\.branch\.feature\.'

# View command-specific configuration  
git config --get-regexp '^gitflow\.feature\.finish\.'
```

### Setting Configuration

```bash
# Set branch type configuration
git config gitflow.branch.feature.upstreamStrategy rebase

# Set command-specific configuration
git config gitflow.feature.finish.merge squash

# Set global configuration (all repositories)
git config --global gitflow.feature.finish.notag true
```

### Removing Configuration

```bash
# Remove specific setting
git config --unset gitflow.feature.finish.merge

# Remove all git-flow configuration (reset)
git config --remove-section gitflow
```

## Best Practices

1. **Start with defaults** - Use `git flow init --defaults` for standard configuration
2. **Customize gradually** - Override only what you need to change
3. **Document team conventions** - Use consistent configuration across team repositories
4. **Use command precedence** - Rely on command-line flags for one-off operations
5. **Test configuration** - Use `git flow overview` to verify your setup
6. **Backup configuration** - Export configuration before major changes:
   ```bash
   git config --get-regexp '^gitflow\.' > gitflow-backup.txt
   ```
