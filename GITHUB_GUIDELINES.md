# GitHub Posting Guidelines

How we write prose posted to GitHub: issue bodies, pull request descriptions, PR reviews and inline diff comments, and issue/PR comments and discussion posts. These are the tone and formatting rules common to all of them. Type-specific structure lives in its own place — issue bodies in [ISSUE_GUIDELINES.md](ISSUE_GUIDELINES.md), PR descriptions in the [pull request template](.github/PULL_REQUEST_TEMPLATE.md), and PR reviews and inline comments in the review skills — and each references this doc rather than restating the shared rules.

Not covered here: commit messages (see [COMMIT_GUIDELINES.md](COMMIT_GUIDELINES.md) — they follow git's own conventions, including wrapping the body at ~72 columns), and short structured fields such as labels, milestones, and release notes.

## Applies To

- Issue bodies and comments (bug, enhancement, spec, epic)
- Pull request descriptions
- PR reviews (the summary body), inline diff comments, and threaded review replies
- Issue and PR comments (triage replies, follow-ups, close notes)
- Discussion posts and replies

## Tone & Voice

- **Lead with a summary** — open with 1–2 sentences stating what the post is about before any sections or detail.
- **Be brief and concise** — get to the point; assume the reader is skimming.
- **Professional and neutral** — welcoming, matter-of-fact language; no emojis.
- **Acknowledge briefly, don't gush** — a short thanks for the report is fine, then confirm the outcome plainly ("Confirmed, this is a bug"). Skip effusive praise ("detailed report and clean reproduction"). Don't restate analysis the reporter already gave back to them — add an analysis summary only when they didn't provide one.
- **Read the whole thread before commenting** — read every prior comment first, including earlier ones posted under the maintainer's own GitHub account (which an AI may have written on their behalf via that access). Take the full conversation into account and don't repeat information anyone has already stated. Successive comments should read as one natural, continuous conversation — a closing comment need only add what's new (e.g. "Spec is #131, closing in favor of it"), not re-explain what's already above.
- **Direct and honest** — state verdicts plainly (call a bug a bug), name scope boundaries, and explain the reasoning rather than hand-waving.
- **Reviews focus on what needs to change** — do not open with generic praise ("solid change") or a re-summary of what the PR does; both are filler and the summary is often wrong. Get straight to the required changes, organized by severity, each specific and actionable. If nothing blocks merge, say that plainly. Direct is not harsh — critique the code, not the author.

## Formatting

- **No hard line wrapping** — write each paragraph and list item as a single line; never wrap prose at a fixed column. GitHub renders single newlines inside a paragraph as line breaks, so hard-wrapped text shows up with ragged mid-sentence breaks. Reserve newlines for paragraph breaks, list items, and code blocks. This applies to local `.ai/` sources too, since skills post them to GitHub verbatim. (Commit messages are the exception — they are not web-rendered prose and follow COMMIT_GUIDELINES.md.)
- **Minimal formatting** — plain text where possible; don't over-format with bold, blockquotes, or horizontal rules.
- **Use h3 (`###`) for sections** — not h1 or h2, which compete with the issue/PR title.
- **Tables and code blocks are fine** — when they make things clearer, not for decoration.
- **No checkboxes for decoration** — use a task list only when it is genuinely actionable (e.g. a spec Breakdown), not to style a plain list.

## Attribution

- Credit AI assistance with a `Co-Authored-By:` trailer on the commit only.
- Never add "Generated with Claude" (or any similar line) to an issue body, PR description, review, or comment.

## What to Avoid

- Hard-wrapping body text at a fixed column — renders as ragged breaks on GitHub
- Any emoji
- h1/h2 headings in the body
- Walls of text with no structure
- Restating the title in the first line

## Examples

**Reply to a bug reporter** — brief thanks, plain confirmation, no restating what the reporter already diagnosed.

Avoid:

> Thanks for the detailed report and clean reproduction — this is a real bug. On a fresh repo without a configured git identity, `git flow init` fails while creating the initial commit and leaves the config half-written, which is why the second run reports "already initialized." I've turned it into a spec (#131) and split the two follow-ups into #125 and #126.

Prefer:

> Thanks for the report — confirmed, this is a bug. I've turned it into a spec (#131) and split the two follow-ups into #125 and #126. Closing this in favor of #131 so there's a single source of truth.

If the reporter had not already diagnosed the cause, add a one-line analysis summary before the next steps — otherwise don't repeat theirs back to them.

**PR review summary** — get straight to the required changes; don't pad with praise or a re-summary.

Avoid:

> Solid change — the rollback approach is the right call and the tests cover the failure path well. Two things before merge: the `UnsetConfigSection` call should run before the error return, otherwise the orphan can still persist on an early exit; and scenario 3 needs an assertion that the retry actually creates the branch. Nits inline.

Prefer:

> Two changes needed before merge:
>
> - `UnsetConfigSection` runs after the error return, so an early-exit path still leaves the orphaned `gitflow.branch.qa.*` section — move it before the return.
> - Scenario 3 asserts the retry exits 0 but not that the branch was created — add that assertion.
>
> Nits inline.

**Inline diff comment** (concise, one finding, actionable):

> This writes the config before `CreateBranch`, so a creation failure here leaves an orphaned `gitflow.branch.qa.*` section. Roll back the write on failure so a failed add leaves nothing behind.

## See Also

- [ISSUE_GUIDELINES.md](ISSUE_GUIDELINES.md) — issue titles, labels, and body structure by type
- [COMMIT_GUIDELINES.md](COMMIT_GUIDELINES.md) — commit message format (separate conventions)
- [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md) — PR description structure
