---
name: full-release
description: Run the full release process end-to-end - prep, tag, CI verification, Homebrew tap, and website sync
allowed-tools: Bash, Read, Edit, Grep, Glob
---

# Full Release

Orchestrate a complete release: version prep, push + tag, GitHub Actions
verification, Homebrew tap update, and website documentation sync. Follows
the process defined in `RELEASING.md` — read it first; this skill sequences
it, it does not replace it.

One gate: explicit user confirmation before pushing the tag (step 5).
Everything before it is local and reversible; everything after it is
published.

## Arguments

`/full-release` — determine version bump automatically from commits
`/full-release 1.2.0` — use the given version instead

## Instructions

### 1. Preflight

Verify before doing anything:

```bash
git branch --show-current          # must be main
git status --porcelain             # must be clean
git fetch origin && git status -sb # must not be behind origin/main
gh auth status                     # gh must be authenticated
ls ../homebrew-tap ../git-flow-next-website  # sibling repos must exist
```

If a sibling repo is missing, continue but note that the corresponding step
will be skipped and must be done manually later.

### 2. Prepare the Release

Follow `.claude/skills/release/SKILL.md` to update CHANGELOG.md and both
version files, verify the build, and create the `chore: Bump version to
X.Y.Z` commit.

If HEAD is already an untagged `chore: Bump version` commit, prep was done
in a prior run — verify its version and changelog are still correct and
reuse it.

### 2a. Sync Contributors

Regenerate the `## Contributors` section of `CONTRIBUTORS.md` from git
history so anyone whose PR merged since the last release is listed. Anyone
with a merged commit qualifies — there is no change-size threshold.

1. Collect every distinct author from commit history:

   ```bash
   git shortlog -sne HEAD | sed -E 's/^ *[0-9]+\t//'
   ```

2. Exclude:
   - Anyone already in `## Project Maintainers` (e.g. Alexander Rinass).
   - AI/bot co-authors — the `Co-authored-by: Claude ...
     <noreply@anthropic.com>` trailers are attribution, not contributors,
     and never go in this list.
3. Collapse duplicate identities (same person, multiple emails) into one
   entry, and resolve each to a GitHub profile link. A
   `<id+handle@users.noreply.github.com>` email encodes the handle directly;
   otherwise resolve via the commit author on GitHub:

   ```bash
   gh api "repos/gittower/git-flow-next/commits/<sha>/pulls" \
     --jq '.[].user.login'
   ```

   If a handle can't be resolved, list the plain name without a link rather
   than guessing.
4. Write the entries alphabetically by name as `- [Name](profile-url)`,
   preserving the explanatory comment at the top of the section. Do not add
   per-contributor descriptions — authorship is the record.

Only touch the `## Contributors` section; leave Maintainers, Original
git-flow Authors, and the rest untouched. If nothing changed, leave the file
as-is. This edit is part of the release prep commit from step 2 (amend it in
if the bump commit is already made).

### 3. Verify Changelog/Tag Coupling

The release workflow extracts the GitHub release notes from CHANGELOG.md by
matching the version header against the tag. Verify the header for this
release is exactly `## [X.Y.Z]` for tag `vX.Y.Z` — a mismatch publishes a
release with an empty body.

Also determine whether this is a preview release (version contains
`-alpha`, `-beta`, or `-rc`). Preview releases skip steps 7 and 8.

### 4. Confirm with User

**GATE**: Show the user:

- Previous version → new version, and why (which commits drove the bump)
- The new changelog section verbatim
- Whether it is a preview release
- What will happen next (push, tag, CI, Homebrew, website)

Wait for explicit confirmation. Do not push anything without it.

### 5. Push and Tag

```bash
git push origin main
git tag vX.Y.Z
git push origin vX.Y.Z
```

### 6. Watch CI and Verify the Release

```bash
gh run list --workflow=release.yml --limit 1   # get the run for the tag
gh run watch <run-id> --exit-status
gh release view vX.Y.Z
```

Verify the release has the platform archives (`.tar.gz`, `.zip`), the
checksums file, and a non-empty body with the changelog content.

If the run fails: stop, report the failure with log excerpts. The tag is
already pushed — after the cause is fixed, the workflow can be re-run from
the same tag (`gh run rerun <run-id>`); do not delete/re-push the tag
unless the fix requires a code change.

### 7. Update Homebrew Tap

**Skip for preview releases** — `update_formula.rb` picks the newest
non-draft release and does not filter prereleases, so running it after a
preview tag would ship the preview to brew users.

```bash
cd ../homebrew-tap
git pull
ruby update_formula.rb    # updates formula AND commits itself
git push
```

The script fetches the release checksums and creates the commit — do not
add a manual commit on top. Verify afterwards that
`Formula/git-flow-next.rb` contains the new version.

### 8. Sync Website

**Skip for preview releases.**

Work in `../git-flow-next-website` and follow its
`.claude/commands/sync-docs.md`:

- **Every release**: version number in `src/components/Hero.astro`,
  changelog in `src/pages/changelog.astro` (synced from this repo's
  CHANGELOG.md)
- **Only if commands/options/config changed**: `src/content/docs/commands.md`
  and `src/content/docs/configuration.md` from this repo's `docs/` manpages

Verify with `npm run build`. **Leave the changes uncommitted** for user
review — the website deploys automatically when pushed to main, so pushing
is publishing.

### 9. Report

Summarize:

- Version released, link to the GitHub release
- CI run status
- Homebrew tap: pushed formula version (or skipped + why)
- Website: files changed, build status, and the remaining manual action —
  review the diff in `../git-flow-next-website`, then commit and push to
  deploy
- Any step that was skipped or failed, stated plainly
