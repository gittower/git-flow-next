---
applyTo: ".github/workflows/**/*.yml"
---

# CI Workflow Review Instructions

## Security

- MUST: Use minimal permissions — only request the permissions the workflow actually needs
- MUST: Pin action versions to a specific SHA or major version tag — never use `@latest` or floating tags
- MUST: Never expose secrets in logs — do not echo or print secret values

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
