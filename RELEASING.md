# Releasing

This document describes the release process for git-flow-next.

## Automation

Two skills automate this process:

- **`/release`** — prepares the release locally: determines the version
  bump, updates CHANGELOG.md and both version files, verifies the build,
  and creates the version bump commit (steps 1–4 below).
- **`/full-release`** — runs the entire process end-to-end: `/release`
  prep, push + tag (after confirmation), CI verification, Homebrew tap
  update, and website sync (all steps below).

The manual steps are documented here as the source of truth; the skills
follow this document.

## Version Locations

Update version in **both** files (must match):

- `version/version.go` — Core version constant
- `cmd/version.go` — Version variable for CLI

## Semantic Versioning

We follow [Semantic Versioning](https://semver.org/):

| Change Type | Version Bump | Example |
|-------------|--------------|---------|
| Breaking changes | Major | 1.0.0 → 2.0.0 |
| New features (`feat:`) | Minor | 1.0.0 → 1.1.0 |
| Bug fixes (`fix:`) | Patch | 1.0.0 → 1.0.1 |

## Changelog Convention

We follow [Keep a Changelog](https://keepachangelog.com/). Only **user-facing changes** go in CHANGELOG.md:

| Commit Type | Changelog Section | Include? |
|-------------|-------------------|----------|
| `feat:` | Added | ✅ Yes |
| `fix:` | Fixed | ✅ Yes |
| `BREAKING CHANGE` | Changed | ✅ Yes |
| `perf:` | Changed | ✅ Yes |
| `build:` | Changed | ⚠️ If user-facing |
| `docs:` | — | ❌ No |
| `refactor:` | — | ❌ No |
| `test:` | — | ❌ No |
| `chore:` | — | ❌ No |
| `ci:` | — | ❌ No |
| `style:` | — | ❌ No |

## Release Process

### 1. Determine Version Bump

Review commits since last release:

```bash
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```

- Any `feat:` → **minor** bump
- Only `fix:` → **patch** bump
- Any `BREAKING CHANGE` or `!` suffix → **major** bump

### 2. Update Changelog

Edit `CHANGELOG.md`:

1. Move items from `[Unreleased]` to new version section
2. Add release date
3. Categorize changes: Added, Changed, Fixed, Removed, Security

### 3. Update Version

Update version in both files:

```bash
# Edit version/version.go
# Edit cmd/version.go
```

Verify:

```bash
go build -o /tmp/git-flow main.go && /tmp/git-flow version
```

### 4. Commit Version Bump

```bash
git add CHANGELOG.md version/version.go cmd/version.go
git commit -m "chore: Bump version to X.Y.Z"
```

### 5. Push and Tag

```bash
git push origin main
git tag vX.Y.Z
git push origin vX.Y.Z
```

> **⚠️ Important**: The release workflow extracts the GitHub release notes
> from CHANGELOG.md by matching the version header against the tag. The
> header must be exactly `## [X.Y.Z]` for tag `vX.Y.Z` — a mismatch
> publishes a release with an empty body.

### 6. GitHub Actions

The `.github/workflows/release.yml` workflow automatically:

1. Builds binaries for all platforms (darwin, linux, windows)
2. Extracts release notes for the tagged version from CHANGELOG.md
3. Creates GitHub release with artifacts
4. Generates checksums
5. Marks as prerelease if tag contains `-alpha`, `-beta`, or `-rc`

Verify the release after the run completes:

```bash
gh run watch
gh release view vX.Y.Z
```

### 7. Update Homebrew Tap

**Skip for preview releases** — the update script picks the newest
non-draft release and does not filter prereleases.

After the GitHub release is published, update the Homebrew formula:

```bash
cd /path/to/homebrew-tap
ruby update_formula.rb
git push
```

The script fetches the latest release and its checksums, updates
`Formula/git-flow-next.rb` from the template, **and commits itself** — no
manual `git add`/`git commit` needed, only the push.

Repository: https://github.com/gittower/homebrew-tap

### 8. Update Website

**Skip for preview releases.**

The website repo has a `/sync-docs` command that performs this step (see
`.claude/commands/sync-docs.md` there). Two parts are needed on **every**
release, one only sometimes:

1. **Every release**: update the version number in
   `src/components/Hero.astro` and sync the changelog page
   (`src/pages/changelog.astro`) from CHANGELOG.md
2. **Only if commands, options, or configuration changed**: update
   `src/content/docs/commands.md` and `src/content/docs/configuration.md`
   from the `docs/*.md` manpages

Verify with `npm run build`, review the diff, then commit and push.
**Pushing to main deploys automatically** via GitHub Pages — there is no
separate deploy step.

Repository: https://github.com/gittower/git-flow-next-website

## Preview Releases

For preview releases, use suffixes:

- Alpha: `v1.0.0-alpha.1`
- Beta: `v1.0.0-beta.1`
- Release candidate: `v1.0.0-rc.1`

These are automatically marked as prereleases on GitHub. Do **not** update
the Homebrew tap or the website for preview releases.

## Checklist

- [ ] Reviewed commits since last release
- [ ] Determined correct version bump (major/minor/patch)
- [ ] Updated CHANGELOG.md with user-facing changes
- [ ] Changelog header `## [X.Y.Z]` exactly matches the planned tag `vX.Y.Z`
- [ ] Updated version in `version/version.go`
- [ ] Updated version in `cmd/version.go`
- [ ] Verified build: `go build && ./git-flow version`
- [ ] Committed: `chore: Bump version to X.Y.Z`
- [ ] Pushed to main
- [ ] Created and pushed tag
- [ ] Verified GitHub release: artifacts, checksums, non-empty release notes
- [ ] Updated Homebrew tap (`ruby update_formula.rb` + `git push`) — skip for previews
- [ ] Updated website: version + changelog (every release), command/config docs (if changed) — skip for previews
