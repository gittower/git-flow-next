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

**--fetch**
: Fetch from remote before deleting. This updates local refs so that Git can correctly detect whether the branch has been merged remotely (e.g., via a GitHub PR merge). The parent branch is fast-forwarded from its remote only when it is the branch currently checked out (`git merge --ff-only` acts on HEAD); delete auto-checks-out the parent when you delete the branch you are on, which is the common case. If the remote is unreachable the fetch failure is a non-fatal note and deletion is not blocked by the fetch itself, but the topic sync check still runs against existing local tracking data and can still abort (behind/diverged) unless **--force** is given.

**--no-fetch**
: Don't fetch from remote before deleting (overrides config). This skips only the fetch; the topic sync check still runs against existing local tracking data.

### Worktree Cleanup

A branch checked out in a linked worktree cannot be deleted while it is checked out there, so delete frees the worktree first. What "freeing" means depends on who created it: git-flow removes the ones it created; a worktree created by hand (`git worktree add`) is kept, with its HEAD detached from the branch instead — the directory and every file in it, including uncommitted work, stay exactly as they were. Neither flag has a git config equivalent; both are CLI-only, like **git-flow-checkout**(1)'s **--worktree**.

**--keep-worktree**
: Keep the branch's worktree instead of removing it, even when git-flow created it. The directory survives on a detached HEAD, and the branch is still deleted. Has no additional effect on a worktree git-flow did not create, which is always detached rather than removed.

**--force-worktree**, **-W**
: Remove a git-flow-created worktree even if it has uncommitted or untracked changes, discarding them. Only applies to the removal path — detaching never needs it, since detaching changes no files. If the worktree has a merge, rebase, bisect, cherry-pick, or revert in progress, deletion is refused regardless of **--force-worktree**: an in-progress operation cannot be abandoned by either freeing path.

If you are standing inside the worktree being removed, the main worktree's path — not the one being removed — is written to **GIT_FLOW_CD_FILE** (see **git-flow-worktree**(1)) as the destination; delete always offers the main worktree, unlike **finish**, which has no merge target of its own to prefer instead. Detaching never navigates: the directory stays exactly where it is.

A branch with no worktree, or one checked out in the main worktree, is unaffected by either flag.

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

### Fetch Before Delete

Fetch and fast-forward the parent branch before deleting, so Git can detect branches merged remotely (e.g., via a GitHub PR merge):
```bash
git flow feature delete my-feature --fetch
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

Delete a branch and its remote tracking branch:
```bash
git flow feature delete completed-feature --remote
```

### Worktree Cleanup

Delete a branch with a git-flow-created worktree (the worktree is removed automatically):
```bash
git flow feature delete my-feature
```

Delete the branch but keep its worktree, detached:
```bash
git flow feature delete my-feature --keep-worktree
```

Delete a branch whose git-flow-created worktree has uncommitted changes:
```bash
git flow feature delete my-feature --force-worktree
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

### Fetch Settings
```bash
# Always fetch before deleting feature branches
git config gitflow.feature.delete.fetch true
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

**git-flow**(1), **git-flow-finish**(1), **git-flow-list**(1), **git-flow-worktree**(1), **git-branch**(1), **git-push**(1)

## NOTES

- Deletion is permanent - use **git reflog** for recovery if needed
- Cannot delete the currently checked out branch
- Remote deletion requires push permissions to the remote repository
- **--force** bypasses Git's safety checks - use with caution
- Branch prefixes are automatically handled during name resolution
- Consider using **git flow list** to see available branches before deletion