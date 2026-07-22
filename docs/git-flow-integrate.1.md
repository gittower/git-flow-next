# GIT-FLOW-INTEGRATE(1)

## NAME

git-flow-integrate - Integrate a base branch into its parent

## SYNOPSIS

**git-flow integrate** [*branch*] [*options*]

## DESCRIPTION

Integrate merges a **base** branch upstream into its configured parent branch (for example, **develop** into **main**), honoring the branch type's upstream merge strategy, optionally creating a tag on the parent, and auto-updating the parent's `autoUpdate` children.

Unlike **git-flow-finish**(1), which completes and then *deletes* a topic branch, integrate operates on permanent base branches: it **never deletes, creates, or renames** a branch. The integrated base branch remains after the operation.

If *branch* is omitted, the current branch is integrated into its parent.

The integrate operation follows the same conflict-resumable state machine as finish, minus the delete step:

1. **Merge**: Merges the base branch into its parent using the configured upstream strategy.
2. **Create Tag**: Optionally creates and signs a tag on the parent (off by default; see **--tag**).
3. **Update Children**: Updates the parent's `autoUpdate=true` child branches using their downstream strategies.

A persistent state file lets the operation resume after conflicts. If a conflict occurs during any merge (the upstream merge or a child update), the state is saved and the operation can be continued with **--continue** or aborted with **--abort**. The **--continue**/**--abort** flags act only on an in-progress integrate; an in-progress finish or update is never affected.

## ARGUMENTS

*branch*
: Name of the base branch to integrate into its parent. Must be a configured base branch (`gitflow.branch.<name>.type = base`). If omitted, the current branch is used.

## OPTIONS

### Operation Control

**--continue**, **-c**
: Continue the integrate operation after resolving merge conflicts. This acts only on an in-progress integrate. If a **finish** or **update** operation is in progress instead, integrate refuses non-destructively, names the owning operation, prints its resume/abort commands, and exits 3 without touching it.

**--abort**, **-a**
: Abort the integrate operation and return to the original state. When no integrate operation is in progress, **--abort** is a no-op and exits successfully. A completed upstream merge and tag are preserved when aborting during a child update. Like **--continue**, it acts only on an integrate operation: a foreign in-progress finish or update is refused (exit 3) rather than aborted.

### Tag Creation

**--tag** *name*
: Create an annotated tag *name* on the parent branch after the merge. Unlike finish, this is a single string flag that both enables tagging and supplies the tag name — base branches have no version from which to derive a default name.

**--notag**, **-n**
: Do not create a tag (overrides a configured tag default).

**--sign**, **-s**
: Sign the tag cryptographically with GPG.

**--no-sign**
: Don't sign the tag cryptographically.

**--signingkey** *keyid*, **-u** *keyid*
: Use the given GPG key for the signature.

**--message** *msg*, **-m** *msg*
: Use *msg* as the tag message.

**--messagefile** *file*
: Read the tag message from *file*. Takes precedence over **--message**.

### Merge Strategy

**--rebase**, **-r**
: Rebase the base branch onto its parent before merging. **Caution:** this rewrites the base branch's history, which is disruptive for a shared, permanent branch — use only when the base branch is not shared.

**--no-rebase**
: Do not rebase (use the configured strategy).

**--squash**, **-S**
: Squash all commits into a single commit on the parent.

**--no-squash**
: Keep individual commits (don't squash).

**--preserve-merges**, **-p**
: Preserve merges during rebase.

**--no-preserve-merges**
: Flatten merges during rebase.

**--no-ff**
: Create a merge commit even when a fast-forward is possible.

**--ff**
: Allow a fast-forward merge when possible.

**--merge-message** *msg*, **-M** *msg*
: Custom commit message for the upstream merge.

**--update-message** *msg*
: Custom commit message for child branch updates.

**--squash-message** *msg*
: Custom commit message for a squash merge.

### Remote Fetch

**--fetch**
: Fetch from the remote and fast-forward the local parent before integrating. Fetching is **off by default** (opt-in).

**--no-fetch**
: Do not fetch from the remote (overrides a configured fetch default).

## CONFIGURATION

Operational defaults are read from the `gitflow.<branch>.integrate.*` namespace, keyed by the base-branch name. See **gitflow-config**(5) for the full list. Note the integrate-specific Layer-1 defaults differ from finish:

- **Tagging is off by default** (base branches have no version-derived tag name).
- **Fetching is off by default** (opt-in).

The upstream merge strategy resolves through the standard three layers: `gitflow.branch.<name>.upstreamStrategy` (Layer 1), `gitflow.<branch>.integrate.*` (Layer 2), and command-line flags (Layer 3, highest priority).

## EXAMPLES

Integrate develop into main:
```bash
git flow integrate develop
```

Integrate the current branch into its parent:
```bash
git flow integrate
```

Integrate and tag the parent:
```bash
git flow integrate develop --tag v2.0.0
```

Resume after resolving a conflict:
```bash
git flow integrate --continue
```

Abort an in-progress integrate:
```bash
git flow integrate --abort
```

## EXIT STATUS

**0**
: Success.

**1**
: git-flow is not initialized.

**2**
: Invalid input (for example, tagging enabled without a resolvable name).

**3**
: A Git operation failed, a merge or rebase is already in progress, unresolved conflicts remain, or there is no in-progress operation to continue.

**5**
: A required branch (the parent, or the named branch) does not exist.

**6**
: A validation error (not a base branch, no parent, self parent, or no upstream strategy).

## NOTES

- Integrate applies only to base branches. For topic branches (feature, release, hotfix, and custom types), use **git-flow-finish**(1).
- The **--rebase** strategy rewrites the integrated branch's history. Avoid it on branches that are shared with other collaborators.
- When aborting during a child update, the already-completed upstream merge and tag on the parent are preserved; only the in-progress child update is rolled back.

## SEE ALSO

**git-flow-finish**(1), **git-flow-update**(1), **git-flow-config**(1), **gitflow-config**(5), **git-flow**(1)

## AUTHORS

git-flow-next is maintained by the Tower team at GitTower GmbH.

## BUGS

Report bugs at https://github.com/gittower/git-flow-next/issues

## COPYRIGHT

This is free software; see the source for copying conditions.
