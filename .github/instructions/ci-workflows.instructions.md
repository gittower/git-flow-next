---
applyTo: ".github/workflows/**/*.yml"
---

# CI Workflow Review Instructions

## Security

- MUST: Use minimal permissions — only request the permissions the workflow actually needs
- MUST: Reference every action by a full commit SHA or at least a major version tag — never `@latest`, never a branch name
- MUST: Prefer calling a tool directly over adding an action owned outside GitHub — see Action pinning below
- MUST: Never expose secrets in logs — do not echo or print secret values

### Action pinning

Only a full commit SHA is an immutable reference. Every version tag floats: whoever can write to the action's repository can repoint it at new code, and that code then runs here with the job's permissions and secrets without any change to this repository. So the question for a `uses:` is who is allowed to move its tag, and what the job would hand them.

- SHOULD: Replace an action owned outside GitHub with a direct call to the underlying tool where one exists. `gh` and `curl` are on the runner, and a `run:` step changes only through a PR against this repository. This is how the release and WinGet jobs publish (#239, #241).
- SHOULD: SHA-pin an outside action that cannot be avoided, with the version in a trailing comment so Dependabot can bump the pin and the comment together. Weigh it against what the job can reach — a job holding `contents: write` or a signing credential is not a build job on a read-only token.
- SHOULD: Leave `actions/*` and `azure/*` on a major tag. A SHA pin would defend against those tags being repointed, so this is a judgment about likelihood, not an argument that pinning is useless: both namespaces are maintained by GitHub and Microsoft under organizational controls, GitHub already supplies the runner and the token, and the pins cost a hand-bump per release. Revisit it for the signing job first if the balance ever changes — `azure/login` federates into the code-signing account, which is the strongest credential any job here holds.
- MUST: Verify anything a `run:` step downloads against a digest recorded in this repository, not against one published alongside the download. See the komac pin in `release.yml`.

Major bumps do not arrive on their own — `@v4` follows v4.x forever. `.github/dependabot.yml` is what surfaces them.

## Concurrency

- SHOULD: Set concurrency groups to prevent duplicate runs for the same branch or PR
- SHOULD: Use `cancel-in-progress: true` for PR-triggered workflows to avoid wasting resources on superseded commits

## Reliability

- SHOULD: Use explicit `timeout-minutes` on jobs to prevent hung workflows
- SHOULD: Fail fast on critical steps — do not continue if build or lint fails
- NIT: Add descriptive step names that explain what each step does, not just the action being called

## Maintainability

- SHOULD: Keep workflow files focused on a single purpose — separate CI, release, and review workflows
- NIT: Use consistent formatting and indentation across all workflow files
