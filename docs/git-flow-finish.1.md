# GIT-FLOW-FINISH(1)

## NAME

git-flow-finish - Complete and merge topic branches

## SYNOPSIS

**git-flow** *topic* **finish** [*name*] [*options*]

**git-flow finish** [*options*]

## DESCRIPTION

Complete a topic branch by merging it to its parent branch according to the configured merge strategy. This command works with any topic branch type (feature, release, hotfix, support, or custom types).

The finish operation follows a strict state machine with these steps:
1. **Merge**: Merges the topic branch to its parent branch using the configured upstream strategy
2. **Create Tag**: Optionally creates and signs tags (if configured)
3. **Update Children**: Updates any child branches that have `autoUpdate=true` using their downstream strategies
4. **Delete Branch**: Optionally deletes the topic branch (local and/or remote)

The operation maintains a persistent state file that allows it to resume after conflicts. If conflicts occur during any merge operation (main merge or child updates), the state is saved and the operation can be continued with **--continue** or aborted with **--abort**.

## ARGUMENTS

*topic*
: The topic branch type (feature, release, hotfix, support, or any configured custom type)

*name*
: Name of the topic branch to finish. If omitted when using the shorthand **git-flow finish**, the current branch is used.

## OPTIONS

### Operation Control

**--continue**, **-c**
: Continue the finish operation after resolving merge conflicts. This acts only on an in-progress finish. If an **update** or **integrate** operation is in progress instead, finish refuses non-destructively, names the owning operation, prints its resume/abort commands, and exits 3 without touching it.

**--abort**, **-a**
: Abort the finish operation and return to the original state. When no finish operation is in progress, **--abort** is a no-op and exits successfully. Like **--continue**, it acts only on a finish operation: a foreign in-progress update or integrate is refused (exit 3) rather than aborted.

**--force**, **-f**
: Force finish: ignore fetch failures, skip the remote sync check, and allow finishing non-standard branches. The fetch is still attempted, but any failure is ignored rather than fatal, and the safety check that prevents finishing when the local branch is behind or diverged from its remote tracking branch is bypassed.

### Tag Creation

**--tag**
: Create a tag for the finished branch (overrides configuration)

**--notag**, **-n**
: Don't create a tag for the finished branch (overrides configuration)

**--sign**, **-s**
: Sign the tag cryptographically with GPG

**--no-sign**
: Don't sign the tag cryptographically

**--signingkey**, **-u** *keyid*
: Use the given GPG key for the digital signature

**--message**, **-m** *message*
: Use the given message for the tag

**--messagefile** *file*
: Use contents of the given file as tag message

**--tagname**, **-T** *name*
: Use the given tag name instead of the default

### Branch Retention

**--keep**, **-k**
: Keep the topic branch after finishing (don't delete)

**--no-keep**
: Delete the topic branch after finishing (default behavior)

**--keepremote**
: Keep the remote tracking branch after finishing

**--no-keepremote**
: Delete the remote tracking branch after finishing

**--keeplocal**
: Keep the local branch after finishing

**--no-keeplocal**
: Delete the local branch after finishing

**--force-delete**, **-D**
: Force delete the branch even if not fully merged

**--no-force-delete**
: Don't force delete the branch (default)

### Merge Strategy Control

**--rebase**
: Rebase topic branch before merging (overrides configured strategy)

**--no-rebase**
: Don't rebase topic branch (use configured strategy)

**--squash**, **-S**
: Squash all commits into single commit (overrides configured strategy)

**--no-squash**
: Keep individual commits (don't squash)

**--squash-message** *message*
: Custom commit message for squash merge. This is a CLI-only option with no git config equivalent, as squash messages are specific to each branch being finished.

**--merge-message**, **-M** *message*
: Custom commit message for the upstream merge operation (topic branch to parent). Useful for teams using commit message validation hooks (e.g., conventional commits) where auto-generated messages like "Merge branch 'feature/foo'" would be rejected. Supports placeholders (see MESSAGE PLACEHOLDERS below). Can be configured as default via `gitflow.<type>.finish.mergemessage`.

**--update-message** *message*
: Custom commit message for child branch update operations (parent to child branches). When finishing a release or hotfix, child branches like develop are automatically updated from the parent. This option allows customizing those merge commit messages. Supports placeholders (see MESSAGE PLACEHOLDERS below). Can be configured as default via `gitflow.<type>.finish.updatemessage`.

**--preserve-merges**, **-p**
: Preserve merges during rebase operations

**--no-preserve-merges**
: Flatten merges during rebase (default)

**--no-ff**
: Create merge commit even for fast-forward cases

**--ff**
: Allow fast-forward merge when possible (default)

**--ff-only**
: Require that the merge into the parent branch be a fast-forward. This is a precondition, not a merge strategy: if the parent branch carries any commit the topic branch does not — whether the two have truly diverged or the parent is merely ahead — finish runs the configured fetch and then aborts before touching any local branch, tag or the working tree, so what lands on the parent is exactly the tested topic tip. Integrate the parent into the topic branch, re-test, and finish again. The condition is re-checked immediately before the merge, so a parent branch advanced in the meantime — by the pre-finish hook or by a concurrent process — is caught as well; that later abort still performs no merge, creates no tag and deletes no branch, but it is not a no-op, because the pre-finish hook has already run and whatever it did stands. The merge itself is run with Git's own `--ff-only`, so a repository that sets `merge.ff = false` still gets a fast-forward rather than a merge commit. Overrides git config setting `gitflow.<type>.finish.ff-only`.

### Remote Fetch Options

**--fetch**
: Fetch from remote before finishing the branch (default). This fetches both the base branch and the topic branch to ensure the latest remote changes are known before merging. Overrides git config setting `gitflow.<type>.finish.fetch`. If no remote is configured, the fetch is automatically skipped. The base (parent) branch is fetched best-effort — a failure there is a non-fatal note. A failure fetching the **topic** branch against a reachable-but-failing remote is **fatal**: finish aborts and names the cause, suggesting `--no-fetch` or `--force`. A remote that simply has no ref for the topic (never pushed, or deleted after a remote merge) is treated as benign and the topic's sync check is skipped.

**--no-fetch**
: Don't fetch from remote before finishing. Disables the default fetch behavior. Overrides git config setting `gitflow.<type>.finish.fetch`. The fetch is skipped, but the remote sync check still runs against existing (local) tracking data.

### Remote Push Options

By default, finishing a branch performs only local work; nothing is pushed. These options opt in to pushing the results of a completed finish to the configured remote (`gitflow.origin`, default `origin`) as a final stage, after all merges, tags, child updates, and branch deletion are done. The finished topic branch itself is never pushed by these options.

When pushing branches, finish pushes the target (parent) branch first, followed by each auto-updated child base branch (for example `main` then `develop` on a release or hotfix finish). Each branch is pushed explicitly to the configured remote with a plain `git push <remote> <branch>`, so it does not depend on or alter the branch's upstream tracking configuration.

**--push**
: Push the target branch and each auto-updated child base branch (and, by default, the created tag) after finishing. Overrides git config setting `gitflow.<type>.finish.push`.

**--no-push**
: Don't push branches after finishing. Because the tag-push default derives from the branch-push decision, `--no-push` also suppresses the inherited tag push unless `--pushtag` is given explicitly. Overrides git config setting `gitflow.<type>.finish.push`.

**--pushtag**
: Push the created tag after finishing. The default for pushing the tag follows the resolved branch-push setting, so a bare `--push` already pushes the tag; use `--pushtag` to push the tag without pushing branches, or to re-enable the tag push when branch pushing is disabled. Has no effect when no tag was created. Overrides git config setting `gitflow.<type>.finish.pushtag`.

**--no-pushtag**
: Don't push the created tag after finishing, even when branches are pushed. Overrides git config setting `gitflow.<type>.finish.pushtag`.

If no remote is configured, the push stage is skipped with a note and finish still exits successfully. If a push is rejected (for example a non-fast-forward update because the remote has diverged), finish exits with an error; the local finish is already complete and nothing is rolled back — re-run the push manually after reconciling.

CLI push flags are not persisted across `--continue`. Options are re-resolved on continue, so to enable a push that must survive a conflict-and-continue, set the `gitflow.<type>.finish.push` config key rather than relying on the flag.

### Hook Control

**--no-verify**
: Bypass pre-commit and commit-msg hooks during merge and commit operations. This passes the `--no-verify` flag to the underlying `git merge` and `git commit` commands. Useful when hooks would interfere with automated finishing workflows or when you want to temporarily skip validation. The setting is persisted through `--continue` operations after conflict resolution. Overrides git config setting `gitflow.<type>.finish.noverify`.

## REMOTE SYNC CHECK

Before performing the merge operation, the finish command checks if the local topic branch is in sync with its remote tracking branch. This safety check prevents accidental data loss when the remote has commits that are not present locally (behind or diverged). Being ahead of the remote is tolerated with a note, since finish merges the local commits into the parent and (by default) deletes the topic branch. The parent (merge-target) branch is checked too — see *Parent Branch Sync Check* below.

### Sync Status Behavior

**Equal**: Local and remote are at the same commit. Finish proceeds normally.

**Ahead**: Local has commits not pushed to remote. Finish **proceeds** and prints a note. The unpushed commits are merged into the parent branch and the topic branch is then deleted (by default), so requiring a push first would preserve nothing.

**Behind**: Remote has commits not present locally. Finish **aborts with an error** to prevent discarding those changes.

**Diverged**: Both local and remote have unique commits. Finish **aborts with an error** with a diverged-specific message, since finishing would discard the remote-only commits.

**No Tracking**: Branch has no remote tracking branch configured. Finish proceeds normally (no remote to compare against).

### Parent Branch Sync Check

In addition to the topic branch, finish checks that the parent (merge-target) branch — for example `develop` when finishing a feature, or `main` when finishing a release — is in sync with its remote before merging into it. This prevents finish from writing a merge onto a stale base that would have to be reconciled (or force-pushed) later.

The parent must not be **behind** or **diverged** from its remote, but being **equal** or **ahead** is accepted. A locally-ahead parent is the normal state right after a previous finish that has not been pushed yet, so it must not block work. Like the topic check in `finish`, an ahead parent is tolerated (behind/diverged remain fatal) — the parent simply proceeds silently rather than printing an ahead note.

**Equal**: Parent matches its remote. Finish proceeds.

**Ahead**: Parent has local commits not on its remote (e.g. an earlier unpushed finish). Finish **proceeds** silently.

**Behind** / **Diverged**: The parent's remote has commits the local parent lacks. Finish **aborts with an error** naming the parent branch; update the parent (for example `git checkout develop && git pull`) or pass **--force**.

The parent check is gated the same way as the topic check: it is skipped when no remote is configured, when the parent has no remote tracking branch, or when the parent's remote ref is gone (never pushed or deleted after a remote merge). With **--no-fetch** the parent is not fetched but the check still runs against existing tracking data. **--force** bypasses it along with the topic check.

### Bypassing the Check

Use the **--force** flag to bypass the remote sync check:

```bash
git flow feature finish --force my-feature
```

This is useful when you intentionally want to discard remote changes, such as when the remote commits were experimental or should not be included in the merge.

### Resolving Sync Issues

When finish aborts due to being behind the remote, you have several options:

```bash
# Option 1: Update your branch with remote changes first
git flow feature update my-feature

# Option 2: Pull changes directly
git pull

# Option 3: Force finish (discards remote changes)
git flow feature finish --force my-feature
```

## MERGE STRATEGIES

The merge strategy used when finishing follows a three-layer precedence system:

**merge**
: Standard Git merge creating a merge commit (preserves branch history)

**rebase**
: Rebase topic branch onto parent creating linear history

**squash**
: Combine all topic branch commits into a single commit

### Strategy Precedence

The merge strategy follows a three-layer precedence system (highest to lowest):

1. **Command-line flags**: `--rebase`, `--squash`, `--no-rebase`, `--no-squash` (highest priority)
2. **Command-specific config**: `gitflow.<type>.finish.rebase`, `gitflow.<type>.finish.squash`
3. **Branch defaults**: `gitflow.branch.<type>.upstreamstrategy` (lowest priority)

### Additional Options

The following options modify strategy behavior:

- **--preserve-merges**: Only valid with rebase operations
- **--no-ff**: Forces creation of merge commits, even for fast-forward cases
- **--ff**: Allows fast-forward merges when possible (default)
- **--ff-only**: Requires the merge into the parent to be a fast-forward. `--ff`, `--no-ff` and `--ff-only` are three values of one setting, so combining `--ff-only` with either of the others is rejected, as is combining it with a squash strategy (squashing always creates a new commit). It constrains the upstream merge only — the automatic update of child base branches still creates merge commits as usual.

## MESSAGE PLACEHOLDERS

Custom merge and update messages can include placeholders that are automatically expanded with branch names. This allows creating dynamic commit messages without hardcoding branch names.

### Supported Placeholders

| Placeholder | Description | Example Value |
|-------------|-------------|---------------|
| **%b** | Branch name | `feature/my-feature` |
| **%B** | Full refname | `refs/heads/feature/my-feature` |
| **%p** | Parent branch name | `develop` |
| **%P** | Full parent refname | `refs/heads/develop` |
| **%%** | Literal percent sign | `%` |

### Context for Placeholders

**For --merge-message** (topic branch to parent):
- `%b` = the topic branch being merged (e.g., `feature/my-feature`)
- `%p` = the parent branch receiving the merge (e.g., `develop`)

**For --update-message** (parent to child branches):
- `%b` = the child branch being updated (e.g., `develop`)
- `%p` = the source branch (e.g., `main`)

### Examples

```bash
# Simple branch name placeholder
git flow feature finish my-feature --merge-message "feat: merge %b"
# Result: "feat: merge feature/my-feature"

# Branch and parent placeholders
git flow feature finish my-feature --merge-message "feat: merge %b into %p"
# Result: "feat: merge feature/my-feature into develop"

# Child updates with placeholders
git flow release finish 1.0 --update-message "chore: sync %b from %p"
# Result on develop: "chore: sync develop from main"

# Escaped percent sign
git flow feature finish my-feature --merge-message "100%% complete: %b"
# Result: "100% complete: feature/my-feature"
```

## EXAMPLES

### Basic Usage

Finish current topic branch (shorthand):
```bash
git flow finish
```

Finish specific feature branch:
```bash
git flow feature finish user-authentication
```

Finish release branch with tag:
```bash
git flow release finish 1.2.0 --tag
```

### Handling Conflicts

When conflicts occur during finish:
```bash
# Start finish operation
git flow feature finish my-feature

# Conflicts occur, resolve them manually
git add conflicted-files
git commit

# Continue the finish operation
git flow feature finish my-feature --continue
```

Or abort if needed:
```bash
git flow feature finish my-feature --abort
```

### Tag Management

Create signed tag with custom message:
```bash
git flow release finish 1.2.0 --sign --message "Release version 1.2.0"
```

Use specific GPG key:
```bash
git flow release finish 1.2.0 --sign --signingkey ABC123DEF
```

### Merge Strategy Examples

Force rebase strategy regardless of configuration:
```bash
git flow feature finish my-feature --rebase
```

Squash all commits into single commit:
```bash
git flow feature finish my-feature --squash
```

Rebase with preserved merges:
```bash
git flow feature finish my-feature --rebase --preserve-merges
```

Create merge commit even for fast-forward:
```bash
git flow feature finish my-feature --no-ff
```

Refuse to finish unless the parent can be fast-forwarded:
```bash
git flow feature finish my-feature --ff-only
```

Override configured squash with regular merge:
```bash
git flow feature finish my-feature --no-squash
```

Squash with custom commit message:
```bash
git flow feature finish my-feature --squash --squash-message "feat: add login functionality"
```

### Custom Merge Commit Messages

Use custom merge message for conventional commits:
```bash
git flow feature finish my-feature --merge-message "feat(auth): add user authentication"
```

Custom messages for both merge and child updates:
```bash
git flow release finish 1.2.0 \
  --merge-message "release: version 1.2.0" \
  --update-message "chore: sync develop with main after release 1.2.0"
```

### Branch Retention

Keep branch for backporting:
```bash
git flow hotfix finish 1.1.1 --keep
```

Clean up both local and remote:
```bash
git flow feature finish my-feature --no-keeplocal --no-keepremote
```

### Bypassing Hooks

Skip pre-commit and commit-msg hooks during finish:
```bash
git flow feature finish my-feature --no-verify
```

Useful in CI/CD environments where hooks might interfere:
```bash
git flow release finish 1.2.0 --no-verify --tag
```

### Pushing After Finish

Push the target branch (and any auto-updated child branches) plus the created tag after finishing:
```bash
git flow release finish 1.2.0 --push
```

Push the updated branches but not the tag:
```bash
git flow release finish 1.2.0 --push --no-pushtag
```

Push only the created tag, leaving branches local:
```bash
git flow release finish 1.2.0 --pushtag
```

Always push a branch type on finish by enabling it in config:
```bash
git config gitflow.feature.finish.push true
git flow feature finish my-feature
```

## WORKFLOW BEHAVIOR

### Dual Merge Pattern

For release and hotfix branches, finish typically performs dual merges:
1. **Topic → Parent**: Merge the topic branch to its parent (e.g., release → main)
2. **Parent → Develop**: Merge parent back to develop to synchronize branches

### Child Branch Updates

After merging to the parent, any child branches configured with `autoUpdate=true` will be automatically updated with the new changes from their parent branch. Each child branch uses its own configured downstream strategy for receiving updates.

Child branches are updated in alphabetical order by branch name. The order is stable across runs, so it also determines where a finish interrupted by a conflict resumes.

The update process:
1. Checks out each child branch in sequence
2. Merges the parent branch using the child's downstream strategy
3. If conflicts occur, saves state and prompts for resolution
4. Continues with next child branch after successful update

Only branches with `autoUpdate=true` are updated:
```bash
git config gitflow.branch.develop.autoUpdate true
```

## CONFIGURATION

Finish behavior is controlled by these configuration keys:

### Branch-Level Defaults
```bash
git config gitflow.branch.<type>.upstreamStrategy merge
git config gitflow.branch.<type>.tag true
```

### Command-Level Overrides
```bash
# Merge strategy overrides
git config gitflow.<type>.finish.rebase true
git config gitflow.<type>.finish.squash false
git config gitflow.<type>.finish.preserve-merges true
git config gitflow.<type>.finish.no-ff true
git config gitflow.<type>.finish.ff-only true

# Tag creation overrides
git config gitflow.<type>.finish.sign true
git config gitflow.<type>.finish.signingkey ABC123DEF

# Remote fetch options
git config gitflow.<type>.finish.fetch true

# Custom merge commit messages (with placeholder support)
git config gitflow.<type>.finish.mergemessage "feat: merge %b into %p"
git config gitflow.<type>.finish.updatemessage "chore: sync %b from %p"

# Hook control
git config gitflow.<type>.finish.noverify true
```

## EXIT STATUS

**0**
: Success.

**1**
: git-flow is not initialized.

**2**
: Invalid input (an unknown branch type, an invalid branch name, or conflicting options such as `--ff-only` combined with `--no-ff`, `--ff`, or a squash strategy).

**3**
: A Git operation failed, a merge or rebase is already in progress, unresolved conflicts remain, a required fetch failed, or there is no in-progress operation to continue.

**5**
: A required branch (the topic branch or its parent) does not exist.

**6**
: A validation error (the topic or parent branch is not in sync with its remote, or the `--ff-only` precondition failed because the parent carries commits the topic branch does not).

## SEE ALSO

**git-flow**(1), **git-flow-start**(1), **git-flow-config**(1), **git-flow-update**(1), **gitflow-config**(5)

## NOTES

- Merge strategy follows three-layer precedence: CLI flags > config > branch defaults
- Command-line flags always override any configuration settings
- **--preserve-merges** flag only applies to rebase operations
- **--squash** and **--rebase** flags are mutually exclusive when both set explicitly
- Use **--continue** and **--abort** for conflict resolution
- Tag creation behavior varies by topic branch type configuration
- The **git-flow finish** shorthand automatically detects current topic branch type
- Child branches are automatically updated when their parent changes
- Some topic branch types (like releases and hotfixes) may create tags by default
- **--merge-message** and **--update-message** can be configured as defaults via `gitflow.<type>.finish.mergemessage` and `gitflow.<type>.finish.updatemessage`
- **--squash-message** is CLI-only with no git config equivalent, as squash messages are specific to each branch being finished
- Custom messages are preserved in merge state and survive conflict resolution