# GIT-FLOW-LIST(1)

## NAME

git-flow-list - List topic branches

## SYNOPSIS

**git-flow** *topic* **list**

## DESCRIPTION

List all topic branches of the specified type. This command works with any topic branch type (feature, release, hotfix, support, or custom types defined in your configuration).

The list command displays all local branches that match the topic branch type's prefix.

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

## OUTPUT FORMAT

The list command prints a heading and one branch per line, indented by two spaces:
```
$ git flow feature list
Feature branches:
  api-v2
  user-auth
```

Branch names are shown without the prefix for readability.

## EXAMPLES

### Basic Usage

List all feature branches:
```bash
git flow feature list
```

List all release branches:
```bash
git flow release list
```

List all hotfix branches:
```bash
git flow hotfix list
```

### Custom Branch Types

List custom branch type:
```bash
git flow bugfix list
git flow epic list
```

## WORKFLOW INTEGRATION

### Before Starting Work
```bash
# Check existing features before starting new one
git flow feature list
git flow feature start new-feature
```

### During Development
```bash
# See all active releases
git flow release list
```

### Cleanup Planning
```bash
# List all features to identify candidates for deletion
git flow feature list
git flow feature delete old-feature
```

## EMPTY RESULTS

When no branches of the type exist:
```bash
$ git flow feature list
No feature branches found
```

## CONFIGURATION

List behavior is controlled by branch type configuration:

### Prefix Configuration
```bash
git config gitflow.branch.feature.prefix feature/
git config gitflow.branch.release.prefix release/
git config gitflow.branch.hotfix.prefix hotfix/
```

The list command automatically uses the configured prefix to identify branches of each type.

## REMOTE BRANCHES

The list command shows only local branches by default. To see remote branches:

```bash
# Show remote feature branches
git branch -r | grep feature/

# Or use Git directly
git ls-remote --heads origin | grep feature/
```

## EXIT STATUS

**0**
: Successful listing (even if no branches found)

**1**
: git-flow is not initialized in this repository, or the command line could not be parsed — an unknown flag, or an argument where the command takes none

**2**
: Invalid topic branch type

**3**
: A Git operation failed

## SEE ALSO

**git-flow**(1), **git-flow-start**(1), **git-flow-delete**(1), **git-branch**(1), **git-ls-remote**(1)

## NOTES

- Only shows local branches - use **git branch -r** for remote branches
- Branch names are displayed without the prefix for readability
- The command takes no arguments; there is no name-pattern filter
- Empty results are not considered an error condition
- Custom topic branch types work exactly like built-in types