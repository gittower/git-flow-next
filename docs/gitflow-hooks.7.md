# GITFLOW-HOOKS(7)

## NAME

gitflow-hooks - Git-flow hooks and filters for customizing workflow behavior

## SYNOPSIS

Custom scripts to extend and customize git-flow operations, located in a configurable hooks directory.

## DESCRIPTION

Git-flow supports two types of extension points:

**Filters** transform input/output values during operations (e.g., modifying version numbers or tag messages).

**Hooks** execute scripts before and after operations (e.g., running CI checks before starting a release).

All scripts are located in the hooks directory following specific naming conventions.

## HOOKS DIRECTORY

By default, git-flow looks for hook and filter scripts in `.git/hooks/`. This can be overridden using configuration, following a three-level precedence:

1. **gitflow.path.hooks** — git-flow-specific override. Set this to use a dedicated directory for git-flow hooks, independent of Git's own hooks. Compatible with git-flow-avh.

2. **core.hooksPath** — Git's native hooks path configuration. If set and `gitflow.path.hooks` is not, git-flow respects this setting.

3. **.git/hooks** — Default location when neither configuration is set. Worktree-aware: in a git worktree, hooks are resolved from the main repository's `.git/hooks` directory.

Both absolute and relative paths are supported. Relative paths are resolved from the repository root.

### Examples

```bash
# Use a tracked directory for git-flow hooks (absolute)
git config gitflow.path.hooks /shared/team-hooks

# Use a tracked directory for git-flow hooks (relative to repo root)
git config gitflow.path.hooks .githooks

# Use Git's native hooks path (also respected by git-flow)
git config core.hooksPath .githooks
```

## FILTERS

Filters transform values passed to git-flow commands. They receive input as arguments or via environment variables and output the transformed value to stdout.

### Filter Naming Convention

```
filter-flow-{type}-{action}-{target}
```

Where:
- `{type}` is the branch type (release, hotfix)
- `{action}` is the git-flow action (start, finish)
- `{target}` is what is being filtered (version, tag-message)

### Available Filters

| Filter Name | Command | Purpose |
|-------------|---------|---------|
| `filter-flow-release-start-version` | `git flow release start` | Modify version number |
| `filter-flow-hotfix-start-version` | `git flow hotfix start` | Modify version number |
| `filter-flow-release-finish-tag-message` | `git flow release finish` | Customize tag message |
| `filter-flow-hotfix-finish-tag-message` | `git flow hotfix finish` | Customize tag message |

### Version Filters

Version filters receive the version as the first argument (`$1`) and should output the modified version to stdout.

**Example: Add 'v' prefix to versions**

```bash
#!/bin/sh
# .git/hooks/filter-flow-release-start-version
VERSION="$1"
if [ "${VERSION#v}" = "$VERSION" ]; then
    echo "v$VERSION"
else
    echo "$VERSION"
fi
```

**Example: Enforce semantic versioning**

```bash
#!/bin/sh
# .git/hooks/filter-flow-release-start-version
VERSION="$1"

# Validate semver format
if ! echo "$VERSION" | grep -qE '^v?[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'; then
    echo "Error: Version must follow semantic versioning (e.g., 1.0.0)" >&2
    exit 1
fi

echo "$VERSION"
```

### Tag Message Filters

Tag message filters receive:
- `$1` - Version number
- `$2` - Original tag message

Environment variables:
- `BRANCH_TYPE` - The branch type (release, hotfix)
- `BRANCH_NAME` - The short branch name
- `BASE_BRANCH` - The parent/base branch
- `VERSION` - The version number

**Example: Append changelog to tag message**

```bash
#!/bin/sh
# .git/hooks/filter-flow-release-finish-tag-message
VERSION="$1"
MESSAGE="$2"

# Get recent commits since last tag
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -n "$LAST_TAG" ]; then
    CHANGELOG=$(git log --oneline "${LAST_TAG}..HEAD" 2>/dev/null)
else
    CHANGELOG=$(git log --oneline -10 2>/dev/null)
fi

echo "${MESSAGE}

Changes in this release:
${CHANGELOG}"
```

### Filter Behavior

- If a filter does not exist, the original value is used
- If a filter is not executable, it is skipped silently
- If a filter exits with non-zero status, the operation fails with an error
- If a filter outputs nothing, the original value is used
- Filter output is trimmed of leading/trailing whitespace

## HOOKS

Hooks are scripts that execute before (pre) or after (post) git-flow operations. Pre-hooks can prevent operations from proceeding.

### Hook Naming Convention

```
{pre,post}-flow-{type}-{action}
```

Where:
- `{pre,post}` indicates when the hook runs
- `{type}` is the branch type (feature, release, hotfix, support)
- `{action}` is the git-flow action (start, finish, publish, track, delete, update)

### Available Hooks

| Hook Pattern | Operations |
|--------------|------------|
| `{pre,post}-flow-feature-{action}` | start, finish, publish, track, delete, update |
| `{pre,post}-flow-release-{action}` | start, finish, publish, track, delete, update |
| `{pre,post}-flow-hotfix-{action}` | start, finish, publish, delete, update |
| `{pre,post}-flow-support-{action}` | start, finish, publish, delete, update |

### Hook Input

Hooks receive context through both **positional arguments** (for git-flow-avh compatibility) and **environment variables** (for enhanced context). You can use either method or both.

#### Positional Arguments

For compatibility with git-flow-avh scripts, hooks receive positional arguments:

| Action | Arguments |
|--------|-----------|
| start | `$1=name` `$2=origin` `$3=branch` `$4=base` |
| finish | `$1=name` `$2=origin` `$3=branch` |
| publish | `$1=name` `$2=origin` `$3=branch` |
| track | `$1=name` `$2=origin` `$3=branch` |
| delete | `$1=name` `$2=origin` `$3=branch` |
| update | `$1=name` `$2=origin` `$3=branch` `$4=base` |

Where:
- `name` - Short branch name (e.g., `my-feature`)
- `origin` - Remote name (e.g., `origin`)
- `branch` - Full branch name (e.g., `feature/my-feature`)
- `base` - Parent/base branch (e.g., `develop`)

**Note:** The `update` action is a git-flow-next extension and follows the same pattern as `start`.

#### Environment Variables

Hooks also receive context via environment variables:

| Variable | Description |
|----------|-------------|
| `BRANCH` | Full branch name (e.g., `feature/my-feature`) |
| `BRANCH_NAME` | Short name (e.g., `my-feature`) |
| `BRANCH_TYPE` | Type (e.g., `feature`) |
| `BASE_BRANCH` | Parent branch (e.g., `develop`) |
| `ORIGIN` | Remote name |
| `VERSION` | Version (for release/hotfix) |
| `EXIT_CODE` | Post-hooks only: exit code of the operation |

#### Compatibility Note

Both methods provide the same core information. Existing git-flow-avh hook scripts using positional arguments (`$1`, `$2`, etc.) will work without modification. New scripts can use either method or both for maximum flexibility.

### Pre-hooks

Pre-hooks run before the operation. If a pre-hook exits with a non-zero status, the operation is aborted.

**Example: Verify CI is passing before release**

```bash
#!/bin/sh
# .git/hooks/pre-flow-release-start

if command -v gh &> /dev/null; then
    STATUS=$(gh run list --branch "${BASE_BRANCH:-develop}" --limit 1 --json conclusion -q '.[0].conclusion')
    if [ "$STATUS" != "success" ]; then
        echo "Error: CI is not passing on ${BASE_BRANCH:-develop}"
        exit 1
    fi
fi

exit 0
```

**Example: Check for uncommitted changes**

```bash
#!/bin/sh
# .git/hooks/pre-flow-feature-start

if ! git diff-index --quiet HEAD --; then
    echo "Error: You have uncommitted changes. Please commit or stash them first."
    exit 1
fi

exit 0
```

### Post-hooks

Post-hooks run after the operation completes, regardless of success or failure. The `EXIT_CODE` environment variable indicates the operation result.

**Example: Send notification on release completion**

```bash
#!/bin/sh
# .git/hooks/post-flow-release-finish

if [ "$EXIT_CODE" -eq 0 ]; then
    echo "Release $VERSION completed successfully!"
    # Send Slack notification, update ticket system, etc.
fi
```

**Example: Update documentation after feature finish**

```bash
#!/bin/sh
# .git/hooks/post-flow-feature-finish

if [ "$EXIT_CODE" -eq 0 ]; then
    echo "Feature $BRANCH_NAME merged. Consider updating documentation."
fi
```

**Example: Run tests after updating branch**

```bash
#!/bin/sh
# .git/hooks/post-flow-feature-update

if [ "$EXIT_CODE" -eq 0 ]; then
    echo "Branch $BRANCH updated from $BASE_BRANCH. Running tests..."
    # npm test || yarn test || make test
fi
```

### Hook Behavior

- If a hook does not exist, the operation proceeds normally
- If a hook is not executable, it is skipped silently
- Pre-hooks that exit non-zero abort the operation
- Post-hooks always run (success or failure), their exit codes are ignored
- Hook output is displayed to the user

## CREATING HOOK SCRIPTS

1. Create the script in the hooks directory (default: `.git/hooks/`) with the appropriate name
2. Make the script executable: `chmod +x <hooks-dir>/<script-name>`
3. Test the script manually before relying on it

### Tips

- Always start scripts with a shebang (`#!/bin/sh` or `#!/bin/bash`)
- Use `exit 0` for success, `exit 1` (or any non-zero) for failure
- Write error messages to stderr: `echo "Error: ..." >&2`
- Test scripts in isolation before enabling them
- Consider using version control for your hook scripts (see below)

## SHARING HOOKS

Since `.git/hooks/` is not tracked by Git, consider these approaches for sharing hooks:

### Using gitflow.path.hooks (recommended)

```bash
# Store hooks in a tracked directory
mkdir .githooks

# Configure git-flow to use this directory
git config gitflow.path.hooks .githooks
```

This only affects git-flow hooks and filters, leaving Git's own hooks unaffected.

### Using core.hooksPath

```bash
# Store hooks in a tracked directory
mkdir .githooks

# Configure Git (and git-flow) to use this directory
git config core.hooksPath .githooks
```

Note: This also affects Git's own hooks (pre-commit, commit-msg, etc.).

### Using global or system configuration

```bash
# Set a shared hooks directory for all repositories (user-wide)
git config --global gitflow.path.hooks /shared/team-hooks

# Set a shared hooks directory system-wide
sudo git config --system gitflow.path.hooks /etc/git-flow-hooks
```

### Using symbolic links

```bash
# Create symbolic links in .git/hooks/
ln -s ../../.githooks/pre-flow-release-start .git/hooks/
```

## EXAMPLES

### Complete Version Filter

```bash
#!/bin/sh
# .git/hooks/filter-flow-release-start-version
# Validates and normalizes version numbers

VERSION="$1"

# Remove 'v' prefix for validation
CLEAN_VERSION="${VERSION#v}"

# Validate semver format (major.minor.patch)
if ! echo "$CLEAN_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "Error: Invalid version format. Use MAJOR.MINOR.PATCH" >&2
    exit 1
fi

# Always output with 'v' prefix
echo "v$CLEAN_VERSION"
```

### Complete Pre-hook (using positional arguments)

```bash
#!/bin/sh
# .git/hooks/pre-flow-feature-start
# Ensures feature names follow conventions
# Compatible with git-flow-avh

# Positional arguments: $1=name, $2=origin, $3=branch, $4=base
NAME="$1"
ORIGIN="$2"
BRANCH="$3"
BASE="$4"

# Check for valid characters
if ! echo "$NAME" | grep -qE '^[a-z0-9-]+$'; then
    echo "Error: Feature name must contain only lowercase letters, numbers, and hyphens" >&2
    exit 1
fi

# Check minimum length
if [ ${#NAME} -lt 3 ]; then
    echo "Error: Feature name must be at least 3 characters long" >&2
    exit 1
fi

echo "Creating feature '$NAME' from '$BASE'"
exit 0
```

### Complete Pre-hook (using environment variables)

```bash
#!/bin/sh
# .git/hooks/pre-flow-feature-start
# Ensures feature names follow conventions
# Uses environment variables (git-flow-next style)

# Check for valid characters
if ! echo "$BRANCH_NAME" | grep -qE '^[a-z0-9-]+$'; then
    echo "Error: Feature name must contain only lowercase letters, numbers, and hyphens" >&2
    exit 1
fi

# Check minimum length
if [ ${#BRANCH_NAME} -lt 3 ]; then
    echo "Error: Feature name must be at least 3 characters long" >&2
    exit 1
fi

echo "Creating $BRANCH_TYPE '$BRANCH_NAME' from '$BASE_BRANCH'"
exit 0
```

## SEE ALSO

**git-flow**(1), **git-flow-start**(1), **git-flow-finish**(1), **gitflow-config**(5), **githooks**(5)

## REFERENCES

- [git-flow-avh Hooks and Filters](https://github.com/petervanderdoes/gitflow-avh/wiki/Reference:-Hooks-and-Filters)
- [Git Hooks Documentation](https://git-scm.com/docs/githooks)
