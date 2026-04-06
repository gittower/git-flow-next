# GIT-FLOW-DELETE(1)

## NAME

git-flow-delete - Delete topic branches

## SYNOPSIS

**git-flow** *topic* **delete** *name* [*options*]

**git-flow delete** [*name*] [*options*]

## DESCRIPTION

Delete a topic branch locally and optionally on the remote. This command works with any topic branch type (feature, release, hotfix, support, or custom types).

The delete operation removes the specified topic branch from the local repository and can also remove the corresponding remote tracking branch.

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

*name*
: Name of the topic branch to delete. When using the shorthand **git-flow delete**, if omitted, the current branch is used.

## OPTIONS

**--force**, **-f**
: Force delete the branch even if it has unmerged changes

**--no-force**
: Don't force delete the branch even if configured (overrides config)

**--remote**, **-r**
: Delete the remote tracking branch in addition to the local branch. Requires a configured remote; the command will fail with a clear error if no remote is available.

**--no-remote**
: Don't delete the remote tracking branch (default behavior)

## SAFETY CHECKS

By default, Git prevents deletion of branches with unmerged changes. The delete command:

- **Checks for unmerged commits** - Prevents accidental loss of work
- **Warns about remote branches** - Shows if remote tracking branch exists
- **Validates branch exists** - Fails gracefully if branch not found

Use **--force** to override safety checks when you're certain the branch should be deleted.

## EXAMPLES

### Basic Usage

Delete a feature branch:
```bash
git flow feature delete user-authentication
```

Delete current topic branch (shorthand):
```bash
git flow delete
```

Delete with remote cleanup:
```bash
git flow feature delete my-feature --remote
```

### Force Deletion

Delete branch with unmerged changes:
```bash
git flow feature delete experimental-feature --force
```

Delete current branch forcibly:
```bash
git flow delete --force
```

### Remote Management

Delete only remote tracking branch:
```bash
git flow feature delete published-feature --remote --no-local
```

Clean up both local and remote:
```bash
git flow feature delete completed-feature --remote
```

## BRANCH NAME RESOLUTION

The delete command accepts branch names with or without prefixes:

```bash
# These are equivalent:
git flow feature delete user-auth
git flow feature delete feature/user-auth
```

## WORKFLOW INTEGRATION

### After Finishing
```bash
# Typical workflow - finish then clean up
git flow feature finish my-feature
git flow feature delete my-feature  # Clean up local branch
```

### Abandoned Features
```bash
# Delete abandoned work
git flow feature delete abandoned-idea --force
```

### Remote Collaboration
```bash
# Delete feature after team review
git flow feature delete reviewed-feature --remote
```

## CONFIGURATION

Delete behavior can be influenced by Git configuration:

### Force Delete Settings
```bash
# Enable force delete by default for feature branches
git config gitflow.feature.delete.force true

# Enable force delete for release branches
git config gitflow.release.delete.force true

# Enable force delete for hotfix branches
git config gitflow.hotfix.delete.force true
```

### Remote Deletion Settings
```bash
# Enable remote deletion by default for feature branches
git config gitflow.branch.feature.deleteRemote true
```

## SAFETY CONSIDERATIONS

**Unmerged Changes**
: By default, Git prevents deletion of branches with unmerged commits

**Remote Synchronization**
: Deleting local branch doesn't automatically delete remote - use **--remote**

**Team Coordination**
: Communicate with team before deleting shared feature branches

**Backup Strategy**
: Consider creating tags or patches for important experimental work before deletion

## RECOVERY

If you accidentally delete a branch:

```bash
# Find the commit hash from reflog
git reflog

# Recreate the branch
git checkout -b recovered-branch <commit-hash>
```

## EXIT STATUS

**0**
: Successful deletion

**1**
: Branch not found

**2**
: Branch has unmerged changes (use --force to override)

**3**
: Invalid branch name

**4**
: Git operation failed

**5**
: Cannot delete current branch (checkout another branch first)

## SEE ALSO

**git-flow**(1), **git-flow-finish**(1), **git-flow-list**(1), **git-branch**(1), **git-push**(1)

## NOTES

- Deletion is permanent - use **git reflog** for recovery if needed
- Cannot delete the currently checked out branch
- Remote deletion requires push permissions to the remote repository
- **--force** bypasses Git's safety checks - use with caution
- Branch prefixes are automatically handled during name resolution
- Consider using **git flow list** to see available branches before deletion