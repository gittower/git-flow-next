# GIT-FLOW-WORKTREE(1)

## NAME

git-flow-worktree - Manage worktrees for branches

## SYNOPSIS

**git-flow worktree add** *branch* [**--path** *path*] [**--no-cd**] [**--quiet**]

**git-flow worktree remove** *branch* [**--force**] [**--no-cd**]

**git-flow worktree list**

**git-flow worktree prune**

**git-flow worktree path** *branch*

## DESCRIPTION

Manage Git worktrees by branch name, independently of the branch lifecycle.

Subcommands address worktrees by **full branch name** (e.g. `feature/user-auth`), not by topic type plus short name. Paths are computed from the **gitflow.worktreePath** template unless **--path** overrides them, and the computed path is always absolute.

git-flow records the worktrees it creates by writing a provenance marker in Git config, so **list** can tell them apart from worktrees created with plain **git worktree add**. Provenance is never inferred by comparing a worktree's path against the template: **--path** and any later change to the template both break that correspondence.

## SUBCOMMANDS

**add** *branch*
: Create a worktree for an existing branch at the computed path (or at **--path**). Intermediate directories of a nested path such as `feature/user-auth` are created. The branch must exist and must not be checked out in another worktree — Git allows a branch in only one worktree at a time. Writes the provenance marker for the branch.

**remove** *branch*
: Remove the worktree that has *branch* checked out. The branch itself is kept. Removal refuses a worktree with uncommitted or untracked changes unless **--force** is given, and never removes the main worktree. Because the command names its target explicitly, it removes the worktree whatever its origin — an unmanaged worktree is removed just like a git-flow-created one. Clears the provenance marker.

**list**
: List the linked worktrees with their branch and path, one row each. The main worktree is excluded. A worktree git-flow did not create is tagged `(unmanaged)`; a worktree whose HEAD is detached shows `(detached)` in place of a branch name and is reported as unmanaged, since there is no branch to key provenance on. A stale entry — one whose directory was deleted by hand — is listed as-is until **prune** runs. Prints `No linked worktrees found` when there are none.

**prune**
: Drop the administrative entries of worktrees whose directories no longer exist, then drop every provenance marker whose branch has no live worktree, so markers never outlive what they describe. A worktree that is still live but detached from its branch keeps its directory and loses its marker.

**path** *branch*
: Print the path the template computes for *branch* and nothing else. Creates nothing and changes nothing.

## OPTIONS

**--path** *path*
: Create the worktree at *path* instead of the computed one (**add** only). A relative value resolves against the **invocation directory** — the directory the command was run from — which is what a hand-typed path means. Note the deliberate asymmetry with a relative **gitflow.worktreePath** template, which resolves against the main worktree root because it is a repository-wide setting rather than one command's argument.

**--force**
: Remove a worktree even when it has uncommitted or untracked changes, discarding them (**remove** only).

**--no-cd**
: Do not write a navigation destination for the calling shell, even when **GIT_FLOW_CD_FILE** is set (**add** and **remove**).

**--quiet**, **-q**
: Do not print the tip naming **git flow shell-init** (**add** only).

## ENVIRONMENT

**GIT_FLOW_CD_FILE**
: When set to a writable path, a command that navigates writes its absolute destination there. git-flow runs as a subprocess and cannot change its parent shell's working directory, so this variable is how a caller asks for that destination in a form it can act on: point the variable at a file, run the command, then read the file and change directory yourself. The variable is an input git-flow reads, never one it sets, which is why it is not a Git config key: it describes one invocation's calling environment, not a repository preference.

: **add** writes the new worktree's path once it exists. **remove** writes only when it would otherwise strand the user: if the current directory is inside the worktree being removed, it writes the main worktree's path, so the shell is never left in a deleted directory. Run from anywhere else, **remove** writes nothing.

: The destination is written **only after** the operation it follows has succeeded, so a refused or failed command leaves the file empty. **--no-cd** suppresses the write even when the variable is set. A failure to write the destination is **not fatal**: the operation still succeeds and a warning is printed to standard error.

: When the variable is unset — an ordinary shell, a script, CI — nothing is written and the command prints the human `cd` hint it would print anyway. Nothing machine-readable ever goes to standard output, so a caller that does not use the channel never sees a protocol line.

: **add** also prints a tip naming **git flow shell-init**, but only in that same state: a caller that set the variable has the wrapper installed already, and the advice would be noise. **--quiet** suppresses the tip outright. See **git-flow-shell-init**(1).

## EXAMPLES

Create a worktree for an existing branch:
```bash
git flow worktree add feature/user-auth
```

Create it somewhere else:
```bash
git flow worktree add feature/user-auth --path ../review-copy
```

See where a branch's worktree would go, without creating it:
```bash
git flow worktree path feature/user-auth
```

List the linked worktrees:
```bash
$ git flow worktree list
feature/user-auth  /home/you/my-project-worktrees/feature/user-auth
feature/api-v2     /home/you/elsewhere/api-v2  (unmanaged)
```

Remove a worktree, keeping the branch:
```bash
git flow worktree remove feature/user-auth
```

Remove one with uncommitted changes:
```bash
git flow worktree remove feature/user-auth --force
```

Clean up after deleting a worktree directory by hand:
```bash
git flow worktree prune
```

Change the path template for this repository:
```bash
git config gitflow.worktreePath '../{{ repo }}-worktrees/{{ branch }}'
git config gitflow.worktreePath '~/worktrees/{{ topicType }}/{{ branchName }}'
```

## CONFIGURATION

**gitflow.worktreePath**
: Template for the path of a branch's worktree. Supports **{{ repo }}** (the main worktree's directory name), **{{ branch }}** (the full branch name), **{{ branchName }}** (the branch without its topic prefix) and **{{ topicType }}** (the topic branch type, empty for a non-topic branch), in the spaced (`{{ branch }}`) and unspaced (`{{branch}}`) form alike. A leading `~` expands to the home directory; a relative template resolves against the main worktree root. Defaults to `../{{ repo }}-worktrees/{{ branch }}`. See **gitflow-config**(5).

**gitflow.worktree.*branch*.managed**
: The provenance marker **add** writes and **remove** clears. This is repository-local state written by git-flow, not a setting to configure by hand: it is what **list** reads to decide whether a worktree is tagged `(unmanaged)`. It is read from and written to **local** config only, and is excluded from the shared-config set, so it never reaches a committed `.gitflow`.

## EXIT STATUS

**0**
: Success

**1**
: git-flow is not initialized in this repository

**3**
: A Git operation failed

**4**
: A worktree for the branch already exists

**5**
: The branch does not exist, or has no worktree to remove

**6**
: A refusal: the target path is occupied, the branch is checked out in another worktree, the target is the main worktree, or the worktree has uncommitted or untracked changes

## SEE ALSO

**git-flow**(1), **git-flow-checkout**(1), **git-flow-shell-init**(1), **gitflow-config**(5), **git-worktree**(1)

## NOTES

- **Requires Git 2.17 or newer.** `git worktree remove`, which **remove** builds on, was added in 2.17; the other operations used here go back further, but the command as a whole needs 2.17. There is no runtime version check — on an older Git you see Git's own error.
- A relative **--path** is invocation-directory-relative, while a relative **gitflow.worktreePath** template is main-worktree-relative. See **OPTIONS**.
- Provenance comes from the marker only, never from the shape of the path.
- The main worktree is never removed and never detached.
- A **--path** inside the repository work tree is allowed, matching Git's own behavior; the worktree simply shows up as an untracked directory.
- An existing **empty** directory is accepted as the target of **add**, again matching Git's own behavior; only a file or a directory with content in it counts as occupied and is refused.
- A worktree whose directory is already gone counts as clean, so **remove** does not demand **--force** for it.
