#!/bin/bash
# PreToolUse hook — re-surface the GitHub posting checklist before any GitHub
# write goes out. Non-blocking: it injects a reminder the model self-verifies
# the body against, then the post proceeds under the normal permission flow
# (no permissionDecision field, so existing prompts are untouched).
#
# Source of truth for the rules: .claude/skills/_shared/POSTING_CHECKLIST.md
# and GITHUB_GUIDELINES.md. Keep the reminder below in sync with the checklist.
#
# Wired in .claude/settings.json for the mcp__github__ write tools.

cat <<'JSON'
{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"Posting-checklist reminder (see .claude/skills/_shared/POSTING_CHECKLIST.md). Verify this body before it posts. Mechanical, never correct to violate: no emoji; sections use h3 (###), never h1/h2; no hard-wrapped prose — one line per paragraph and list item, fenced code exempt; no AI-attribution line in the body (Co-Authored-By / 'Generated with Claude' belong on the commit trailer only); the first line does not restate the title. Tone: lead with a 1-2 sentence summary; concise, no walls of text; neutral, no gushing praise ('detailed report', 'solid change') or effusive thanks; don't restate analysis the other person already gave; a reply reads as one continuous conversation and adds only what is new; a review opens with required changes by severity, not praise or a re-summary. If any item fails, fix the body and re-post, then re-read what rendered on GitHub."}}
JSON
exit 0
