# GIT-FLOW-CHECKOUT(1)

## NAME

git-flow-checkout - Switch to topic branches

## SYNOPSIS

**git-flow** *topic* **checkout** *name*|*nameprefix*

## DESCRIPTION

Switch to an existing topic branch of the specified type. This command works with any topic branch type (feature, release, hotfix, support, or custom types) and supports partial name matching for convenience.

The checkout command is a convenient wrapper around **git checkout** that automatically handles branch prefixes and provides partial name matching.

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

*name*|*nameprefix*
: Full name or partial name prefix of the topic branch to checkout. Supports partial matching for convenience.

## OPTIONS

**--showcommands**
: Show the underlying git commands as they are executed.

## PARTIAL NAME MATCHING

The checkout command supports partial name matching for convenience:

**Exact Match** (preferred)
: If a branch matches the exact name, it's selected

**Prefix Match**
: If no exact match, looks for branches starting with the given prefix

**Ambiguous Match**
: If multiple branches match the prefix, shows options and fails

## EXAMPLES

### Basic Usage

Checkout a specific feature branch:
```bash
git flow feature checkout user-authentication
```

Checkout using partial name:
```bash
# Assuming you have feature/user-authentication
git flow feature checkout user
```

Checkout a release branch:
```bash
git flow release checkout 1.2.0
```

### With and Without Prefixes

These commands are equivalent:
```bash
git flow feature checkout user-auth
git flow feature checkout feature/user-auth
```

### Partial Matching Examples

If you have these feature branches:
- `feature/user-authentication`
- `feature/user-profile`  
- `feature/api-endpoints`

```bash
git flow feature checkout user    # Ambiguous - shows options
git flow feature checkout user-a  # Matches user-authentication
git flow feature checkout api     # Matches api-endpoints
```

## AMBIGUOUS MATCHES

When multiple branches match a prefix:

```bash
$ git flow feature checkout user
Error: Ambiguous branch name 'user'. Matches:
  feature/user-authentication
  feature/user-profile
  
Please specify a more complete name.
```

## WORKFLOW INTEGRATION

### Daily Development
```bash
# Quick switch between features
git flow feature checkout api
git flow feature checkout user-a
```

### Context Switching
```bash
# Switch to hotfix for urgent work
git flow hotfix checkout critical-fix

# Return to feature work
git flow feature checkout main-feature
```

### Team Collaboration
```bash
# Checkout teammate's branch for review
git flow feature checkout teammate-feature
```

## BRANCH VALIDATION

The checkout command validates:

**Branch Exists**
: Verifies the target branch exists locally

**Branch Type Match**
: Ensures branch has the correct prefix for the topic type

**Unique Resolution**
: Handles partial name matching and ambiguity

## REMOTE BRANCHES

The checkout command works only with local branches. For remote branches:

```bash
# Fetch remote branches first
git fetch origin

# Create local tracking branch
git checkout -b feature/remote-feature origin/feature/remote-feature

# Then use git-flow commands normally
git flow feature checkout remote-feature
```

## CONFIGURATION

Checkout behavior respects branch type configuration:

### Prefix Configuration
```bash
git config gitflow.branch.feature.prefix feature/
git config gitflow.branch.release.prefix release/
git config gitflow.branch.hotfix.prefix hotfix/
```

The checkout command uses these prefixes to identify and validate branches.

## ERROR HANDLING

Common error scenarios:

**Branch not found**:
```bash
$ git flow feature checkout nonexistent
Error: Branch 'feature/nonexistent' not found
```

**Ambiguous match**:
```bash
$ git flow feature checkout user
Error: Ambiguous branch name 'user'. Multiple matches found.
```

**Wrong branch type**:
```bash
$ git flow feature checkout release/1.2.0
Error: Branch 'release/1.2.0' is not a feature branch
```

## ALTERNATIVES

For more advanced checkout operations, use Git directly:

```bash
# Checkout and create tracking branch
git checkout -b feature/new-branch origin/feature/new-branch

# Checkout specific commit
git checkout feature/branch-name~3

# Checkout with path specification
git checkout feature/branch -- specific-file.txt
```

## EXIT STATUS

**0**
: Successful checkout

**1**
: Branch not found

**2**
: Ambiguous branch name

**3**
: Invalid topic branch type

**4**
: Git checkout failed

## SEE ALSO

**git-flow**(1), **git-flow-list**(1), **git-flow-start**(1), **git-checkout**(1)

## NOTES

- Only works with local branches - fetch remote branches first if needed
- Supports partial name matching for convenience
- Branch prefixes are automatically handled
- Use **git flow list** to see available branches before checkout
- For complex checkout operations, use **git checkout** directly
- Partial matching prioritizes exact matches over prefix matches