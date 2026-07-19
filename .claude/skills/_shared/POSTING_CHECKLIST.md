# GitHub Posting Checklist

A pre-post self-verification for anything written to GitHub. Skills reference
this document instead of restating the rules, so they stay in one place. The
source of truth for *why* is [GITHUB_GUIDELINES.md](../../../GITHUB_GUIDELINES.md);
this is the operational checklist a skill runs before it posts.

## When to Run

Immediately before any GitHub write — issue body or comment, PR description,
PR review body or inline comment, discussion post or reply. Run it against the
exact text about to be sent, not an earlier draft. If the text lives in an
`.ai/` file that gets posted verbatim, verify the file.

## The Check

Read the drafted body once and confirm every item. If any fails, fix the text
and re-check before posting — never post first and fix after.

**Mechanical (unambiguous — a violation is always wrong):**

- [ ] No emoji anywhere.
- [ ] No h1 (`#`) or h2 (`##`) headings — sections use h3 (`###`).
- [ ] No hard-wrapped prose — each paragraph and list item is a single line, no fixed-column breaks. (Fenced code blocks are exempt.)
- [ ] No AI-attribution line in the body ("Generated with Claude", robot emoji, `Co-Authored-By`) — credit goes on the commit trailer only.
- [ ] The first line does not restate the title.

**Tone (judgment — read the draft as the recipient):**

- [ ] Leads with a 1–2 sentence summary of what the post is about.
- [ ] Brief and concise; no walls of text.
- [ ] Professional and neutral; no gushing praise ("detailed report", "solid change") and no effusive thanks.
- [ ] Does not restate analysis the other person already gave back to them.
- [ ] For a reply in a thread: read every prior comment first (including ones under the maintainer's own account); it reads as one continuous conversation and adds only what is new.
- [ ] For a review: opens with the required changes organized by severity, not praise or a re-summary; each point specific and actionable; says so plainly if nothing blocks merge.

**Structure (type-specific — confirm the right one):**

- [ ] Issue body follows [ISSUE_GUIDELINES.md](../../../ISSUE_GUIDELINES.md) (title has no type prefix, correct label, right section structure for its type).
- [ ] PR description follows the [pull request template](../../../.github/PULL_REQUEST_TEMPLATE.md).

## After Posting

For posts that matter (a spec issue, a PR description, a review), re-read what
actually rendered on GitHub — occasionally the API mangles formatting or a
paste drops a section. Fix in place if it did not come out as intended.
