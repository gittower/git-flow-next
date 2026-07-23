# Git-Flow-AVH Configuration Reference

This document provides a complete reference of all git-flow-avh configuration options for compatibility analysis with git-flow-next.

## Global Configuration Settings

### Remote Repository
- **`gitflow.origin`** - Set different remote repository name
  - Default: `origin`
  - Example: `git config gitflow.origin myorigin`

### Environment Variables
- Can be set in `.gitflow_export` file in home directory
- **`GIT_MERGE_AUTOEDIT`** - Control merge commit editing
  - Example: `export GIT_MERGE_AUTOEDIT=no`

### Working Tree State
- **`gitflow.allowdirty`** - Allow starting branches with dirty working tree
  - Type: Boolean
  - Default: `false`
  - Example: `git config --bool gitflow.allowdirty true`

## Branch Prefixes (Legacy Format)

### Core Branch Names
- **`gitflow.branch.master`** - Production branch name
  - Default: `master`
- **`gitflow.branch.develop`** - Development branch name
  - Default: `develop`

### Topic Branch Prefixes
- **`gitflow.prefix.feature`** - Feature branch prefix
  - Default: `feature/`
- **`gitflow.prefix.bugfix`** - Bugfix branch prefix
  - Default: `bugfix/`
- **`gitflow.prefix.release`** - Release branch prefix
  - Default: `release/`
- **`gitflow.prefix.hotfix`** - Hotfix branch prefix
  - Default: `hotfix/`
- **`gitflow.prefix.support`** - Support branch prefix
  - Default: `support/`
- **`gitflow.prefix.versiontag`** - Version tag prefix
  - Default: empty string

## Command-Specific Configuration

### Configuration Variable Convention
- **Format**: `gitflow.<branchtype>.<command>.<option>`
- **Boolean Values**:
  - Affirmative: `Y`, `YES`, `T`, `TRUE`
  - Negative: `N`, `NO`, `F`, `FALSE`

### Feature Branch Options

#### Start Command
- **`gitflow.feature.start.fetch`** - Fetch from remote before starting
  - Type: Boolean
  - Default: `false`

#### Finish Command
- **`gitflow.feature.finish.fetch`** - Fetch from remote before finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.feature.finish.rebase`** - Use rebase instead of merge
  - Type: Boolean
  - Default: `false`
- **`gitflow.feature.finish.preserve-merges`** - Preserve merges during rebase
  - Type: Boolean
  - Default: `false`
- **`gitflow.feature.finish.push`** - Push to remote after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.feature.finish.keep`** - Keep branch after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.feature.finish.force-delete`** - Force delete branch after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.feature.finish.squash`** - Squash commits when finishing
  - Type: Boolean
  - Default: `false`

### Bugfix Branch Options

#### Start Command
- **`gitflow.bugfix.start.fetch`** - Fetch from remote before starting
  - Type: Boolean
  - Default: `false`

#### Finish Command
- **`gitflow.bugfix.finish.fetch`** - Fetch from remote before finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.bugfix.finish.rebase`** - Use rebase instead of merge
  - Type: Boolean
  - Default: `false`
- **`gitflow.bugfix.finish.preserve-merges`** - Preserve merges during rebase
  - Type: Boolean
  - Default: `false`
- **`gitflow.bugfix.finish.push`** - Push to remote after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.bugfix.finish.keep`** - Keep branch after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.bugfix.finish.force-delete`** - Force delete branch after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.bugfix.finish.squash`** - Squash commits when finishing
  - Type: Boolean
  - Default: `false`

### Release Branch Options

#### Start Command
- **`gitflow.release.start.fetch`** - Fetch from remote before starting
  - Type: Boolean
  - Default: `false`

#### Finish Command
- **`gitflow.release.finish.fetch`** - Fetch from remote before finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.release.finish.sign`** - Sign the release tag
  - Type: Boolean
  - Default: `false`
- **`gitflow.release.finish.signingkey`** - GPG key for signing
  - Type: String
  - Default: Git configuration default
- **`gitflow.release.finish.push`** - Push to remote after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.release.finish.keep`** - Keep branch after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.release.finish.force-delete`** - Force delete branch after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.release.finish.squash`** - Squash commits when finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.release.finish.notag`** - Skip tag creation
  - Type: Boolean
  - Default: `false`

### Hotfix Branch Options

#### Start Command
- **`gitflow.hotfix.start.fetch`** - Fetch from remote before starting
  - Type: Boolean
  - Default: `false`

#### Finish Command
- **`gitflow.hotfix.finish.fetch`** - Fetch from remote before finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.hotfix.finish.sign`** - Sign the hotfix tag
  - Type: Boolean
  - Default: `false`
- **`gitflow.hotfix.finish.signingkey`** - GPG key for signing
  - Type: String
  - Default: Git configuration default
- **`gitflow.hotfix.finish.push`** - Push to remote after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.hotfix.finish.keep`** - Keep branch after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.hotfix.finish.force-delete`** - Force delete branch after finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.hotfix.finish.squash`** - Squash commits when finishing
  - Type: Boolean
  - Default: `false`
- **`gitflow.hotfix.finish.notag`** - Skip tag creation
  - Type: Boolean
  - Default: `false`

### Support Branch Options

#### Start Command
- **`gitflow.support.start.fetch`** - Fetch from remote before starting
  - Type: Boolean
  - Default: `false`

#### Rebase Command
- **`gitflow.support.rebase.interactive`** - Use interactive rebase
  - Type: Boolean
  - Default: `false`

## Configuration Examples

### Setting Global Configuration
```bash
# Enable fetching before operations
git config --global gitflow.feature.start.fetch true
git config --global gitflow.feature.finish.fetch true

# Use rebase for feature finishing
git config gitflow.feature.finish.rebase true

# Sign release and hotfix tags
git config gitflow.release.finish.sign true
git config gitflow.hotfix.finish.sign true

# Set custom remote origin
git config gitflow.origin upstream
```

### Repository-Specific Configuration
```bash
# Allow dirty working tree for this repository
git config --bool gitflow.allowdirty true

# Custom branch prefixes
git config gitflow.prefix.feature "feat/"
git config gitflow.prefix.release "rel/"
```

## Compatibility Notes

### Branch Types
- git-flow-avh includes `bugfix` branches not in original git-flow
- Support branches are available but less commonly used
- Master branch is default (not main as in newer systems)

### Key Differences from git-flow-next
1. **Configuration Format**: AVH uses `gitflow.prefix.*` vs git-flow-next's `gitflow.branch.*.prefix`
2. **Branch Names**: AVH defaults to `master` vs git-flow-next's `main`
3. **Merge Strategies**: git-flow-next reads the same `finish.rebase`/`finish.squash` flags AVH uses, and adds branch-level `upstreamStrategy`/`downstreamStrategy` configuration on top
4. **Tag Management**: AVH has `notag` flags vs git-flow-next's `tag` boolean configuration

### Migration Path
When migrating from git-flow-avh to git-flow-next, the following mappings apply:

| AVH Configuration | git-flow-next Equivalent |
|------------------|--------------------------|
| `gitflow.branch.master` | `gitflow.branch.main` |
| `gitflow.prefix.feature` | `gitflow.branch.feature.prefix` |
| `gitflow.prefix.release` | `gitflow.branch.release.prefix` |
| `gitflow.prefix.hotfix` | `gitflow.branch.hotfix.prefix` |
| `gitflow.prefix.versiontag` | `gitflow.branch.*.tagprefix` |
| `gitflow.feature.finish.rebase` | `gitflow.feature.finish.rebase` (unchanged — read directly) |
| `gitflow.release.finish.squash` | `gitflow.release.finish.squash` (unchanged — read directly) |
| `gitflow.release.finish.notag` | `gitflow.release.finish.notag` (unchanged — read directly; or `gitflow.branch.release.tag=false`) |

---

*This reference is based on git-flow-avh wiki documentation and is maintained for compatibility analysis with git-flow-next.*