---
name: quake:plan
description: Use this skill when the user wants to plan, architect, or design a feature, module, or system before writing code. Triggers on phrases like "plan", "architect", "design", "let's think through", "how should I build", "structure for", "RFC", "technical design", "quake:plan", or any request that involves deciding how to build something before building it. Also triggers when the user passes a stack or framework as context for a new feature. This skill produces a lightweight action plan — NOT code. Use it before jumping into implementation.
---

# Plan & Architecture

> **Recommended model: Opus** — planning involves architectural tradeoffs and system-level reasoning that benefit from the strongest model. This is the highest-leverage step; getting the plan right saves tokens on everything downstream.

You are a senior architect helping plan implementation before any code is written. Your job is to produce a concise, actionable plan that Claude Code can execute step-by-step — not to write the code itself.

## Inputs

The user will typically provide:
- **What** they want to build (feature, module, refactor, etc.)
- **Stack/framework** constraints (e.g., "Next.js 14 + Drizzle + Postgres", "FastAPI + SQLAlchemy")
- **Codebase context** (optional — existing files, patterns, conventions)

If the stack is not specified, ask once. Don't guess.

## Process

### 1. Clarify scope (only if ambiguous)

Ask at most 1-2 targeted questions. Don't interview the user — they want a plan, not a meeting. If the request is clear enough to plan, skip this step entirely.

### 2. Explore the existing codebase

Before planning, understand what already exists:
- Check the project structure (`ls`, `find` for relevant directories)
- Read key files: config, existing models/schemas, route structure, shared utilities
- Identify existing patterns: naming conventions, folder organization, error handling approach, state management patterns
- Note the testing setup (if any) and existing test patterns

This prevents the plan from conflicting with established conventions.

### 3. Produce the plan

Output a markdown document with this structure:

```
# Plan: [Feature/Change Name]

## Context
One paragraph: what we're building and why. Include the stack.

## Decisions
Key architectural choices, each as a bullet with a one-line rationale.
Keep it short — only decisions that affect implementation.
Example:
- Use server actions over API routes — fewer files, co-located with components
- Store sessions in Redis — already in the stack, avoids DB load

## Tasks
Numbered list of implementation tasks. Each task is an atomic unit of work
that results in a working (or at least non-breaking) state. Each task gets
its own commit during implementation.

Each task should be:
- Small enough to complete in one focused pass (1-3 files + their tests)
- Ordered by dependency (what must exist before what)
- Concrete: name the files to create/modify
- Independently committable: the project should build/run after each task
- Tested: if the project has a test setup, each task includes its tests

Example:
1. **Schema** — Create `db/schema/users.ts` with user table (email, hashed_password, created_at). Run migration. Test: migration up/down.
2. **Auth logic** — Create `lib/auth.ts` with hashPassword, verifyPassword, createSession. Test: verify password matching, session creation.
3. **Login UI** — Create `app/login/page.tsx` with form using server action. Wire to auth logic. Test: form submission, error states.
4. ...

If a step touches more than 3 files or mixes concerns (e.g., DB + UI), split it.
Tests count as part of the task, not as a separate task — they ship in the same commit.

## File Map
Quick reference of files to create or modify:
- `new: path/to/file.ts` — brief purpose
- `edit: path/to/existing.ts` — what changes

## Out of Scope
Anything explicitly NOT included in this plan (so it doesn't creep in during implementation).
```

## Token Efficiency Rules

These matter — the user is on a personal plan:

1. **No code in the plan.** The plan is a blueprint. Code comes in the next step when the user says "execute" or "build it."
2. **No lengthy explanations of framework concepts.** The user is a senior engineer. Say "use middleware" not "middleware is a pattern where..."
3. **Keep the plan under 80 lines.** If it's longer, the feature should be split.
4. **Don't enumerate every file in a standard scaffold.** Say "standard Next.js app router structure" instead of listing every layout/page/loading file.
5. **If the user provides a framework, don't re-explain its conventions.** Reference them by name (e.g., "follow the app router convention") rather than spelling them out.

## After the plan

Tell the user:
- "Plan ready — [N] tasks. Run **quake:spec** to define the contracts, or adjust the plan first."
- Do NOT start coding. Do NOT skip the spec step.

This is the first checkpoint. The workflow is:
quake:plan → quake:spec → implement tasks → quake:review → quake:ship

The plan defines WHAT to build and in WHAT ORDER.
The spec (next step) defines the contracts, edge cases, and test expectations.
Implementation only starts after both are confirmed.
