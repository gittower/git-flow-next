---
name: ship-issue
description: Fully solve a small issue end-to-end — resolve, publish the PR, run two Copilot review rounds, address them, and merge when confident. Escalates to the user on anything uncertain.
argument-hint: <issue-number> [--no-merge] [--rounds <n>]
allowed-tools: Task, Bash, Read, Write, Edit, Glob, Grep, mcp__github__get_issue, mcp__github__get_pull_request, mcp__github__get_pull_request_reviews, mcp__github__get_pull_request_comments, mcp__github__add_issue_comment
---

# Ship Issue

Take a small, well-scoped issue all the way to merged, autonomously. This
skill chains the existing pipeline — `/resolve-issue` to implement, publish,
then **two rounds** of GitHub Copilot code review, addressing each round, and
finally merging when the result is confidently correct.

It is the end-to-end counterpart to `/resolve-issue` (which stops at the
publish gate). Use it only for issues you are confident an AI can solve
unattended. The autonomy is real: it publishes, comments, and merges without
asking — **except** at the escalation points below, where it stops and asks
you.

## Arguments

`/ship-issue <issue-number> [--no-merge] [--rounds <n>]`

- `<issue-number>` — required, the spec issue to solve end-to-end.
- `--no-merge` — do everything through the review rounds, then stop and hand
  the merge decision to you (present the state and the merge command). Use
  when you want the final call.
- `--rounds <n>` — number of Copilot review rounds (default **2**). `0` skips
  Copilot review entirely and goes straight to the merge gate.

## When to use / not use

- **Use** for small, well-scoped spec issues — a contained bug fix, a
  single-command flag, a doc-plus-code change — where the spec is clear and
  the blast radius is small.
- **Do not use** for large or ambiguous work, anything touching merge/conflict
  semantics broadly, or an issue without a spec. Run `/resolve-issue`
  interactively instead. If in doubt about size, run `/resolve-issue` and stay
  in the loop.

## Autonomy & Escalation Contract

Proceed without asking through the happy path. **Stop and ask the user** the
moment any of these holds — describe what happened, what you tried, and the
options, then wait:

1. **Not a small/clear issue** — `/resolve-issue`'s precondition fails
   (non-spec issue, or a parent with a Breakdown), or the spec turns out to be
   larger/more ambiguous than "small" once analyzed.
2. **Resolve aborted** — `/resolve-issue` hits its 3-revision abort, or
   `go build ./...` / `go test ./...` fails and a straightforward fix isn't
   obvious.
3. **Copilot can't be attached** — neither the REST nor the GraphQL request
   attaches the reviewer (see recipe below). Ask the user to add Copilot in
   the web UI, then continue.
4. **Copilot doesn't respond** — no review within the poll timeout (~10 min).
5. **A review finding you can't confidently resolve** — a must-fix that needs
   a design decision, contradicts the spec, or that your fix doesn't fully
   satisfy. Don't guess on correctness.
6. **Rounds didn't converge** — after the final round you still have an
   unresolved must-fix, or each round keeps surfacing new must-fix issues.
7. **Base update conflicts** — the branch fell behind `main` and the merge
   from base produces conflicts that aren't trivially and unambiguously
   resolvable, or resolving them breaks the build/tests.
8. **CI red** — required checks fail after fixes and the cause isn't a
   trivial, obvious fix.
9. **Anything unexpected** — an API error, a force-push seemingly required, a
   merge that reports not-merged, or any state you don't understand. Default
   to asking, not improvising.

The bar for merging is: build + tests green, CI green, both review rounds
addressed with no unresolved must-fix, base up to date, and no lingering
doubt. If all of that holds, merge. If any of it wobbles, escalate.

---

## Workflow

### Step 1: Resolve the issue

Run the `/resolve-issue` workflow (read
`.claude/skills/resolve-issue/SKILL.md`) for issue #`<n>` **in this context**,
executing its Steps 1–9. It analyzes the spec, creates the feature branch in
its own worktree (`../git-flow-next.worktrees/<n>-<slug>`), plans test-first,
implements, runs local + Codex review, applies fixes, and writes
`pr_summary.md`.

**Do not stop at its Publish Gate (Step 10)** — in this autonomous flow you
perform the publish yourself (Step 2). But honor every one of its abort
conditions: if the precondition fails, the 3-revision abort fires, or
build/tests fail, **stop and ask the user** (Escalation 1–2).

After it completes, capture the **worktree path**, **branch name**
(`feature/<n>-<slug>`), and the **`.ai/` folder** — you need them for the rest
of the flow. Confirm you are in the worktree and the branch is checked out.

### Step 2: Publish the PR

Verify the branch is truly ready before going public: `go build ./...` and
`go test ./...` are green, and `pr_summary.md` passes the `/pr-summary`
**Validate Format** checklist (the PR body is a GitHub-only artifact).

Push with an explicit refspec and set tracking explicitly — a worktree branch
mis-tracks with `git push -u` (it can target `main`):

```bash
git push origin feature/<n>-<slug>:refs/heads/feature/<n>-<slug>
git branch --set-upstream-to=origin/feature/<n>-<slug> feature/<n>-<slug>
gh pr create --title "<type(scope): Subject>" --body "$(cat <ai-folder>/pr_summary.md)"
```

`<title>` follows the PR-title format from `/pr-summary` / COMMIT_GUIDELINES.md
(`type(scope): Subject`, no issue number). Verify the posted body with
`gh pr view <pr>`; if it drifted, fix it via REST (see the note on PR-body
edits in the recipe section — `gh pr edit --body` fails on this repo). Record
the PR number.

### Step 3: Copilot review rounds

Repeat for each round `1..rounds` (default 2; skip this whole step if
`--rounds 0`):

1. **Request** a Copilot review on the PR — see
   [Requesting a Copilot review](#requesting-a-copilot-review). For round ≥ 2,
   request only after the previous round's fixes are pushed.
2. **Poll** for the new review (a Copilot review with `submitted_at` newer than
   this round's request). Timeout ~10 min → Escalation 4.
3. **Address** the round — see [Addressing a round](#addressing-a-round).
   Accepted fixes are implemented, committed, pushed, and each thread replied
   to and resolved. If a finding needs a decision you can't confidently make,
   Escalation 5.

After the final round, if an unresolved must-fix remains or rounds aren't
converging, Escalation 6.

### Step 4: Update from base

Before merging, make sure the branch isn't stale:

```bash
git fetch origin
git log --oneline origin/main ^HEAD   # any base commits the branch lacks?
```

If the branch is behind `main`, bring it up to date by **merging** the base
(do not rebase a published branch):

```bash
git merge origin/main
```

- **Clean merge** → run `go build ./...` && `go test ./...`, then push.
- **Conflicts** → resolve only if trivial and unambiguous, then verify
  build + tests still pass and push. If the conflicts are non-trivial, or the
  resolution breaks anything, `git merge --abort` and **Escalation 7**.

Also confirm CI is green (`gh pr checks <pr>`); red required checks →
Escalation 8.

### Step 5: Merge decision

If `--no-merge` was given: present the full state (commits, review outcomes,
CI status, base freshness) and the merge command, and stop — the call is
yours.

Otherwise apply the merge bar from the contract. If everything holds, merge:

```bash
gh pr merge <pr> --merge --delete-branch
```

`gh pr merge` emits a `projectCards` deprecation **warning** on this repo but
still succeeds — confirm `merged: true` / exit 0 and ignore the warning. If it
reports not-merged, Escalation 9.

### Step 6: Close the parent report

`Resolves #<spec>` closes the **spec** issue automatically. GitHub does not
cascade that to the spec's parent, so a user report that the spec was written
for stays open unless you close it — shipped work then sits in the open-issue
list looking unstarted. See
[ISSUE_GUIDELINES.md](../../../ISSUE_GUIDELINES.md) §Relation to User Reports.

Check whether the spec has a parent: the spec body usually opens with
"Refs #<report>", and the report may list the spec as a native sub-issue.

```bash
gh issue view <spec> --json body --jq '.body' | head -5   # look for "Refs #N"
```

If a parent exists and the spec fully completes it, comment what shipped —
naming the spec, the PR and the merge commit — and close it:

```bash
gh issue comment <report> --body "$(cat <<'EOF'
Shipped. The spec for this was #<spec>, implemented in #<pr> and merged to `main` in <sha>.

<2-4 sentences on the user-visible behavior that now exists, written for the reporter rather than the implementer.>

Closing as completed by #<spec>.
EOF
)"
gh issue close <report>
```

Do **not** close the parent when the spec only partly completes it — several
specs may hang off one report. In that case comment what shipped and leave it
open. Repeat up the chain if the report itself has a parent.

### Step 7: Clean up and report

After a successful merge, remove the worktree and local branch:

```bash
cd ../git-flow-next
git worktree remove ../git-flow-next.worktrees/<n>-<slug>
git branch -D feature/<n>-<slug>   # if it still exists locally
git fetch --prune
```

Report: the merged PR URL, the issue it closed, any parent report closed (or
why it was left open), a one-line-per-round summary of Copilot findings
addressed, whether a base update was needed, and confirmation the
branch/worktree were cleaned up.

---

## Requesting a Copilot review

Requesting Copilot on this repo is finicky — several obvious methods silently
no-op. Use this order:

**Fast path — REST with the full bot login** (confirmed reliable):

```bash
gh api repos/gittower/git-flow-next/pulls/<pr>/requested_reviewers \
  -X POST -f "reviewers[]=copilot-pull-request-reviewer[bot]"
```

Success = the response's `requested_reviewers` contains `Copilot`. An empty
array (a 200/201 with no reviewer) is a **silent no-op** — fall through.

**Fallback — GraphQL `requestReviews` with the reviewer bot id:**

```bash
# PR node id:
gh api graphql -f query='query { repository(owner:"gittower",name:"git-flow-next"){ pullRequest(number:<pr>){ id } } }' --jq '.data.repository.pullRequest.id'
# Request — use the REVIEWER bot id BOT_kgDOCnlnWA (NOT the swe-agent BOT_kgDOC9w8XQ):
gh api graphql -f query='mutation($pr:ID!,$bot:ID!){ requestReviews(input:{pullRequestId:$pr, botIds:[$bot], union:true}){ pullRequest { reviewRequests(first:10){ nodes { requestedReviewer { __typename ... on Bot { login } } } } } } }' -f pr="<PR_NODE_ID>" -f bot="BOT_kgDOCnlnWA"
```

`botIds` (not `userIds`); the reviewer bot is `BOT_kgDOCnlnWA`. The
`copilot-swe-agent` id from `suggestedActors` (`BOT_kgDOC9w8XQ`) is the
**coding** bot — passing it here is a silent no-op. Success = `reviewRequests`
lists `Copilot`.

**Human fallback** — if neither attaches the reviewer, ask the user to add
Copilot as a reviewer in the web UI (one click, reliable), then continue.
Escalation 3.

**Polling for completion:**

```bash
gh api repos/gittower/git-flow-next/pulls/<pr>/reviews \
  --jq '[.[] | select(.user.login=="copilot-pull-request-reviewer[bot]")] | last | {state, submitted_at}'
```

Copilot's review objects appear under `user.login ==
"copilot-pull-request-reviewer[bot]"` (inline comments show author `Copilot`).
It usually posts 1–3 min after the request. Poll periodically (e.g. via the
Monitor tool or repeated checks) until a review with a `submitted_at` newer
than this round's request appears; for round 2, that means a **second** review
newer than round 1. Timeout ~10 min → Escalation 4.

## Addressing a round

Delegate to `/address-review` (read `.claude/skills/address-review/SKILL.md`)
for the PR. It fetches the round's comments, evaluates each against the
project's pragmatic philosophy (accept / dismiss / partial), implements
accepted fixes, commits, and prepares inline replies + thread resolutions.

`/address-review` has its own "Push and post?" gate. In this autonomous flow
**you are the approver** for that gate: proceed automatically when the accepted
changes are clear-cut and build + tests pass. But if evaluating a comment
requires a judgment call you're not confident in — a must-fix touching
correctness or the spec, or a dismissal you're unsure the user would agree
with — surface it to the user instead of auto-approving (Escalation 5). Never
auto-dismiss a must-fix to keep the flow moving.

After it runs, the round's fixes are committed, pushed, and its threads
replied to and resolved.

### PR-body edits

If you need to change the PR title/body, `gh pr edit --body` fails on this repo
(projectCards deprecation). Use REST:

```bash
gh api -X PATCH repos/gittower/git-flow-next/pulls/<pr> -f body="$(cat body.md)"
```

## Notes

- This skill orchestrates existing skills — `/resolve-issue`, `/address-review`
  — rather than reimplementing them. Keep it thin; when their behavior changes,
  this inherits it.
- All public writes (PR body, review replies) still go through the
  [posting checklist](../_shared/POSTING_CHECKLIST.md) via the delegated
  skills.
- The autonomy is deliberately asymmetric: bias toward **asking** whenever
  correctness, scope, or an unexpected state is in play. A merged wrong change
  costs far more than a question.
- `--no-merge` and `--rounds 0` let you dial the autonomy down without leaving
  the pipeline — full auto is the default because that's the point.
