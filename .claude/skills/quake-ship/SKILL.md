---
name: quake:ship
description: Use this skill when the user wants to push changes and create a pull request. Triggers on phrases like "create a PR", "push this", "ship it", "open a pull request", "quake:ship", or after quake:review has passed and the user is ready to ship. This skill pushes the branch and creates a clean, concise PR from the git history. It does NOT handle commits — those happen during task implementation or manually.
---

# Ship

> **Recommended model: Sonnet** — reading git history and writing a PR description is straightforward work. Sonnet handles this well without the overhead of Opus.

You handle the final step: pushing reviewed changes and opening a pull request. You do NOT commit — that already happened during task implementation.

## Process

### 1. Check the state

```bash
git status                              # anything uncommitted?
git log --oneline origin/main..HEAD     # commits ready to push
```

**If there are uncommitted changes**, don't silently commit them. Warn the user:
- "There are uncommitted changes. Commit them first, then run quake:ship again."

This keeps the responsibility clear: commits happen during implementation or
manually. quake:ship only pushes and creates the PR.

**If there are no commits ahead of main**, there's nothing to ship:
- "No new commits to push. Nothing to ship."

### 2. Check for PR template

```bash
cat .github/pull_request_template.md 2>/dev/null || cat .github/PULL_REQUEST_TEMPLATE.md 2>/dev/null || echo "no template"
```

If a template exists, fill it in rather than using the default format below.

### 3. Generate the PR

**PR Title:** Read the commit history to determine the right title.
- Single commit: use the commit message as the title
- Multiple commits: write a higher-level summary of what the branch does

**PR Body Format:**
```
## What
One paragraph: what this PR does. No fluff.

## Why
One paragraph: the motivation. Skip if obvious from "What".

## Changes
- Bullet list of logical changes (not file-by-file)
- For task-based work, summarize each task as one bullet

## Testing
How this was tested: "all tests passing" / "manual testing" / specific details.
```

**PR Rules:**
- Keep the whole body under 20 lines
- Don't repeat the title in the body
- Don't add sections like "Screenshots" or "Checklist" unless the project template requires them
- The commit history IS the detailed changelog — the PR body is high-level

### 4. Push and create

Show the PR title + body to the user and ask for confirmation:
- "Here's the PR. **Ship it?** (or edit first)"

On confirmation:
```bash
git push origin HEAD
gh pr create --title "<PR title>" --body "<PR body>"
```

## Token Efficiency Rules

1. **Read the git log, don't ask the user to describe changes.** The history is right there.
2. **One confirmation prompt.** Show PR title + body together, ask once.
3. **Skip the PR body for trivial changes.** A single-commit fix doesn't need "What/Why/Changes."
4. **Total output should be under 20 lines** for a typical PR flow.

## Flags

- `--draft`: Create the PR as a draft
- `--no-confirm`: Skip the confirmation prompt and ship directly
