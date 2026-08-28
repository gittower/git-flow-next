# Releasing

This document describes the release process for git-flow-next.

## Automation

Two skills automate this process:

- **`/release`** — prepares the release locally: determines the version
  bump, updates CHANGELOG.md and both version files, verifies the build,
  and creates the version bump commit (steps 1–4 below).
- **`/full-release`** — runs the entire process end-to-end: `/release`
  prep, push + tag (after confirmation), CI verification, Homebrew tap
  update, WinGet manifest submission, and website sync (all steps below).

The manual steps are documented here as the source of truth; the skills
follow this document.

## Windows Code Signing

Windows executables are Authenticode-signed with Azure Artifact Signing
during the release run, before the archives are created. This is a one-time
setup; it needs no attention during a normal release.

Signing runs in the `release-signing` GitHub environment, which holds:

| Kind | Name | Value |
| --- | --- | --- |
| Secret | `AZURE_CLIENT_ID` | Entra app registration (client) ID |
| Secret | `AZURE_TENANT_ID` | Entra tenant ID |
| Secret | `AZURE_SUBSCRIPTION_ID` | Azure subscription ID |
| Variable | `AZURE_CODESIGN_ENDPOINT` | `https://weu.codesigning.azure.net/` |
| Variable | `AZURE_CODESIGN_ACCOUNT_NAME` | `Trusted` |
| Variable | `AZURE_CODESIGN_CERTIFICATE_PROFILE` | `TowerCertificate` |

Authentication uses GitHub Actions OIDC — there is no client secret to
rotate. The Entra app needs a federated credential for this subject:

```text
repo:gittower/git-flow-next:environment:release-signing
```

and the `Artifact Signing Certificate Profile Signer` role on the
certificate profile.

> The RFC3161 timestamp URL in the workflow is intentionally `http`. The
> service rejects `https` with `SignTool Error: Invalid Timestamp URL`.

### Verifying the signing setup

The Azure-side pieces — OIDC federation, the signer role, and the
account/profile/endpoint values — cannot be checked from the repository. Run
the workflow manually to exercise them without cutting a release:

```bash
gh workflow run release.yml --ref main
```

A `workflow_dispatch` run builds, signs and verifies, then stops: the
release job only runs on a tag push. A green run confirms the whole
credential path at once. If it fails, the `Verify Azure authentication` step
separates a federation problem from a missing role assignment.

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
2. Signs the Windows executables with Azure Artifact Signing
3. Verifies every Windows signature — a bad signature fails the run before
   anything is published
4. Creates the archives, so each one contains the signed executable
5. Generates checksums from the final archives
6. Extracts release notes for the tagged version from CHANGELOG.md
7. Creates GitHub release with artifacts
8. Marks as prerelease if tag contains `-alpha`, `-beta`, or `-rc`

Verify the release after the run completes:

```bash
gh run watch
gh release view vX.Y.Z
```

Confirm a published Windows binary is signed by extracting one and checking
it on Windows:

```powershell
Get-AuthenticodeSignature .\git-flow.exe
```

### 7. Stamp the Milestone

**Skip for preview releases** — an rc/alpha must not consume the `Next`
milestone name.

Issues and PRs are assigned to a rolling `Next` milestone as they merge,
via the `.github/workflows/milestone-on-merge.yml` workflow. Because we use
semantic versioning, the actual version is not known until the release is
cut — so instead of guessing a number upfront, we rename `Next` to the real
version once it ships. Renaming preserves every assignment, so each issue
then carries the version it shipped in on its milestone badge.

After the GitHub release is verified live:

```bash
NEXT_ID=$(gh api repos/:owner/:repo/milestones --jq '.[]|select(.title=="Next")|.number')
gh api --method PATCH repos/:owner/:repo/milestones/$NEXT_ID -f title="vX.Y.Z" -f state=closed
gh api --method POST  repos/:owner/:repo/milestones -f title="Next"
```

Users can then find which version fixed an issue directly on the issue, and
maintainers can list a release's scope with
`gh issue list --milestone vX.Y.Z --state all`.

### 8. Update Homebrew Tap

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

### 9. Submit WinGet Manifest

**Skip for preview releases** — the WinGet community repo is for stable
versions only.

This is an **interim manual step** until manifest submission is automated in
the release workflow. It publishes the new version to WinGet by opening a PR
against `microsoft/winget-pkgs` with
[komac](https://github.com/russellbanks/Komac), which derives the SHA-256
hashes and release notes automatically.

Prerequisites (one-time):

```bash
brew install komac                                # cross-platform manifest tool
gh repo fork microsoft/winget-pkgs --clone=false  # komac PRs from your fork
```

Submit (run only after the GitHub release from step 6 is live — komac
downloads the release zips to hash them):

```bash
export GITHUB_TOKEN=$(gh auth token)   # needs public_repo scope; gh's repo scope covers it
komac update GitTower.GitFlowNext \
  --version X.Y.Z \
  --urls \
    "https://github.com/gittower/git-flow-next/releases/download/vX.Y.Z/git-flow-next-vX.Y.Z-windows-amd64.zip" \
    "https://github.com/gittower/git-flow-next/releases/download/vX.Y.Z/git-flow-next-vX.Y.Z-windows-386.zip" \
  --submit
```

komac forks/updates `winget-pkgs`, pushes a branch, and opens the PR. Note
the PR URL it prints — Microsoft's validation bots run automatically and
normally auto-merge within a few hours, so no manual merge is needed. To
preview the manifests first, swap `--submit` for `--dry-run --output <dir>`.

The manifest covers x64 and x86; arm64 is shipped in the release but not yet
in the WinGet manifest. **Scoop needs no action** — its Main-bucket manifest
has `checkver`/`autoupdate`, so Scoop's excavator bot updates it
automatically after each release.

Repository: https://github.com/microsoft/winget-pkgs

### 10. Update Website

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
the Homebrew tap, the WinGet manifest, or the website for preview releases.

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
- [ ] Verified the Windows signing job succeeded
- [ ] Verified GitHub release: artifacts, checksums, non-empty release notes
- [ ] Stamped milestone: renamed `Next` → `vX.Y.Z`, closed it, opened new `Next` — skip for previews
- [ ] Updated Homebrew tap (`ruby update_formula.rb` + `git push`) — skip for previews
- [ ] Submitted WinGet manifest (`komac update ... --submit`) — skip for previews
- [ ] Updated website: version + changelog (every release), command/config docs (if changed) — skip for previews
