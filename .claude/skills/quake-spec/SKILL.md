---
name: quake:spec
description: Use this skill after a plan is approved and before implementation begins. Triggers on phrases like "write spec", "write the spec", "spec this out", "define the behavior", "acceptance criteria", "what should this do", "quake:spec", or when transitioning from plan to implementation. This skill produces a lightweight behavioral specification that defines WHAT the feature does (inputs, outputs, edge cases, constraints) without prescribing HOW. The spec becomes the contract that the review skill checks against.
---

# Write Spec

> **Recommended model: Sonnet** — the plan already made the hard architectural decisions. The spec is structured translation of those decisions into contracts, which Sonnet handles well at lower token cost.

You produce a lightweight behavioral spec that sits between planning and implementation. This is NOT a 20-page PRD — it's a focused contract that defines what "done" looks like so the review step has something concrete to check against.

## When This Runs

After a plan is approved (the user said "go" or similar), but BEFORE writing any code. The flow is:

```
Plan → Spec → Implement → Review → Commit/PR
```

## Inputs

- The approved plan (from the plan-architecture skill or conversation context)
- The codebase (for understanding existing conventions and interfaces)

If no plan exists, ask the user to describe what they're building. Don't force them through the planning skill if they have a clear mental model.

## Process

### 1. Read the plan and codebase

Understand:
- What's being built (from the plan)
- What interfaces already exist (from the code)
- What conventions are in place (naming, error handling, response formats)

### 2. Produce the spec

Output a markdown document saved to the project as `specs/[feature-name].md`:

```
# Spec: [Feature Name]

## Behavior

Describe what the feature does from the user/caller's perspective.
Use plain language. No implementation details.

Example:
"A user can reset their password by providing their email. They receive
a reset link valid for 1 hour. Clicking the link lets them set a new
password. After reset, all existing sessions are invalidated."

## Interfaces

Define the contracts — what goes in, what comes out.
Group interfaces by the task they belong to (matching the plan's task list).
This way, when implementing Task 2, you can look at just the Task 2 interfaces
in the spec without reading the entire document.

### Task 1: [task name from plan]
**[Interface name, e.g., function, table, component]**
- **Input**: what it accepts (types, required/optional fields)
- **Output**: what it returns (shape, status codes, states)
- **Errors**: what can go wrong and how it's surfaced

### Task 2: [task name]
...

Example:
### Task 1: Schema
**reset_tokens table**
- Columns: id (uuid PK), user_id (FK → users), token (string, unique), expires_at (timestamp), used (boolean default false)
- Index on token for lookup

### Task 2: Auth logic
**generateResetToken(email: string)**
- **Output**: `{ token: string }` or `null` if email not found
- **Side effect**: inserts row in reset_tokens, sends email via Resend
- **Errors**: throws if Resend API fails

**validateToken(token: string)**
- **Output**: `{ userId: string }` or `null` if expired/used/invalid

### Task 3: UI pages
**ForgotPasswordPage**
- **Input**: none
- **Output**: renders email input form, calls generateResetToken server action
- **Errors**: shows generic "check your email" (don't leak existence)

## Edge Cases

Bullet list of scenarios that are easy to miss:

- What happens if [input is empty/null/malformed]?
- What happens if [concurrent requests]?
- What happens if [external service is down]?
- What happens if [user doesn't have permission]?
- What about [rate limiting/abuse]?

Only list edge cases relevant to THIS feature. Don't enumerate every possible failure.

## Constraints

Hard requirements that limit implementation choices:

- Performance: "must respond in < 200ms" (only if it actually matters)
- Security: "tokens must be cryptographically random, not UUIDs"
- Compatibility: "must work without JavaScript for the base flow"
- Data: "must not store plaintext passwords at any point"

Only include constraints the implementer might otherwise miss.
Skip this section entirely if there are no non-obvious constraints.

## Test expectations

Define what should be tested per task — not full test code, just what each test
should verify. This tells the implementer what tests to write alongside each task.

Match the testing conventions already in the project (framework, file location,
naming). If the project has no tests yet, suggest a minimal setup and ask the
user if they want to include it.

### Task 1: [task name]
- Verify [behavior]: [expected outcome]
- Verify [edge case]: [expected outcome]

### Task 2: [task name]
- ...

Example:
### Task 1: Schema
- Migration runs without error on empty DB
- Migration is reversible (down migration drops table)

### Task 2: Auth logic
- generateResetToken returns token for existing email
- generateResetToken returns null for unknown email (doesn't throw)
- validateToken rejects expired tokens (> 1hr)
- validateToken rejects already-used tokens
- resetPassword invalidates all existing sessions for that user

### Task 3: UI pages
- ForgotPasswordPage submits form and shows confirmation (no error leak)
- ResetPasswordPage shows error for mismatched passwords
- ResetPasswordPage redirects to /login on success

Keep test expectations to one line each. These are acceptance criteria, not test
implementations. Skip this section entirely if the user explicitly says no tests.

## NOT in scope

Explicitly state what this spec does NOT cover.
Pull from the plan's "Out of Scope" section if it exists.
```

## Token Efficiency Rules

1. **Keep the entire spec under 60 lines.** If it's longer, the feature is too big — split it.
2. **Skip sections that add no value.** If there are no meaningful constraints, don't include a Constraints section with generic filler like "should be fast."
3. **Don't restate the plan.** The spec defines behavior and contracts, not architecture. Say "see plan for implementation approach" if needed.
4. **Use concrete examples, not abstract descriptions.** "Returns `{ error: 'LINK_EXPIRED' }` after 1 hour" is better than "Returns an appropriate error response when the link has exceeded its validity period."
5. **One spec per feature.** Don't write a spec document for the entire project.

## Saving the Spec

Save to `specs/[feature-name].md` in the project root. Create the `specs/` directory if it doesn't exist.

```bash
mkdir -p specs
```

This file becomes the reference for:
- Implementation: the developer (or Claude) knows exactly what to build
- Review: the review skill can check if the implementation matches the spec
- Future context: when you come back to this project in 3 months, the spec explains WHAT without forcing you to read all the code

## After the Spec

Tell the user:
- "Spec saved to `specs/[feature-name].md`. Ready to implement?"
- Do NOT start coding unless the user confirms.

This is the second checkpoint. Plan → Spec → [user confirms] → Implement.
Two lightweight checkpoints before burning tokens on code.

## During implementation

Once the user confirms, execute tasks from the plan sequentially.
For each task, the spec defines what to build and how to test it.

After completing each task:
1. Write the tests defined in the spec's "Test expectations" for this task
2. Run the tests — they must pass before moving on
3. Verify the project still builds/runs
4. Commit directly with a descriptive message — do NOT invoke quake:ship for this.
   Use conventional commit format: `feat(scope): what this task did`
5. Move to the next task

Per-task commits are lightweight and inline. Save quake:ship for the final PR
after all tasks are done and quake:review passes.

If tests fail, fix the implementation — not the tests (unless the spec was
wrong, in which case flag it to the user). If a task fails or reveals a problem
with the plan, STOP. Tell the user what happened and propose an adjustment —
don't silently re-architect mid-implementation. This is where task boundaries
save the most tokens: you've committed the working tasks, so only the failed
one needs to be re-done.

After all tasks are committed:
- "All [N] tasks implemented and committed. Run **quake:review** to check the changes."
