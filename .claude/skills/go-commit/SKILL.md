---
name: go-commit
description: Use when preparing a git commit in this Go repository, especially when the user asks to commit changes, choose commit boundaries, stage files, write a Conventional Commit message, or wrap up a linked worktree commit flow.
---

# go-commit

## Overview

Turn the current working tree into one high-signal commit. Prioritize release-note-ready history: subjects should describe capability, behavior, or architectural meaning, not file edits.

Local commits stay fast. Do not run `ci-test` or `dev-test` just to create a local commit unless the user explicitly asks. Push-time verification is handled by the repo's `pre-push` hook.

If the current checkout is a linked worktree, do not stop at the local commit. Finish by asking how that worktree branch should be integrated next.

## When to Use

- The user asks to commit current changes
- The user wants a commit message or help staging the right files
- The user wants to split mixed changes into separate commits
- The user wants commit history that can be summarized into release notes
- The user is finishing work in a linked git worktree and needs the next Git step after the local commit

Do not use this skill for hook installation or push-debugging; use the repo scripts and hook workflow instead.

## Workflow

1. Inspect `git status --short`, staged diff, and unstaged diff.
2. Decide whether the changes represent one high-level intent. If not, stop and propose separate commits.
3. Stage only the files for one intent.
4. If one file contains hunks for multiple intents, use `git add -p` so each commit stays coherent.
5. Choose `type` and optional `scope`.
6. Write `type(scope): summary` in English, imperative mood, and high-level wording.
7. If the diff reveals only low-level implementation details and the high-level intent cannot be reliably inferred, ask one focused question before finalizing the message.
8. Add body or footer only for migration notes, breaking changes, or issue references.
9. Create the local commit.
10. Detect whether the current checkout is a linked worktree before suggesting the next Git step. Use `.git` indirection or `git worktree list --porcelain`; do not guess from directory naming alone.
11. If the user also plans to push, remind them that `pre-push` will run `ci-test`.
12. Git can confirm that the current checkout is a linked worktree and which branch is checked out there, but it does not reliably preserve the branch the worktree was originally created from. Treat the source branch as known only when the user or session context already established it.
13. If the current checkout is a linked worktree, end with one focused question offering exactly three next-step choices: rebase onto the known source branch, rebase onto another branch, or push this branch to its remote.
14. If the user chooses the source-branch rebase path and the source branch is not already known, ask one focused follow-up for that branch name before suggesting commands.
15. If the current checkout is not a linked worktree, do not add the rebase-or-push menu unless the user explicitly asks for post-commit integration help.
16. If the user explicitly told you the source branch, use that exact branch name in the follow-up question.
17. Never infer the source branch from the primary checkout branch, another sibling worktree, or whatever branch happens to appear first in `git worktree list`; that data shows current checkouts, not the worktree's creation origin.
18. Do not replace the three-choice worktree question with an immediate request for the source branch or base commit. Ask for the source branch only after the user picks the source-branch rebase path.
19. Treat the repository root checkout as the primary checkout unless `.git` points into `.git/worktrees/...` or `git worktree list --porcelain` shows the current path as a non-primary worktree entry.

## Quick Reference

| Use `type` | When |
| --- | --- |
| `feat` | new capability or externally visible behavior |
| `fix` | bug fix or corrected behavior |
| `refactor` | internal reshaping without behavior change |
| `docs` | documentation-only change |
| `test` | test coverage or test workflow change |
| `chore` | maintenance work that is not feature or bug fix |
| `perf` | performance improvement |
| `build` | dependency, packaging, or build-system change |
| `ci` | hook, automation, or CI workflow change |

## Worktree Follow-up

When the current checkout is a linked worktree, use this closing question after the local commit:

- Known source branch: `Next step for this worktree branch: rebase onto <source-branch>, rebase onto another branch, or push this branch to remote?`
- Unknown source branch: `Next step for this worktree branch: rebase onto its source branch, rebase onto another branch, or push this branch to remote?`

Only turn the first option into a concrete branch name when that source branch is already known from the user or session context.

If the user already named the source branch, prefer that exact branch name over any guess from repository layout.

Follow this order exactly:

| Situation | Next question |
| --- | --- |
| Primary checkout or non-worktree branch | No worktree follow-up menu unless the user asked for post-commit integration help |
| Linked worktree, source branch known | Ask the three-choice menu and name that exact source branch |
| Linked worktree, source branch unknown | Ask the same three-choice menu, but keep the source-branch option generic |
| User picked source-branch rebase and source branch is still unknown | Ask one focused follow-up for the source branch name |

## Message Rules

- Keep `scope` when the affected area is clear.
- Optimize the subject for future release notes.
- Prefer describing what changed for users or operators, not which files moved.
- Avoid generic summaries like `update`, `tweak`, `misc`, or `fix stuff`.
- Avoid file-driven subjects like `update writer.go and rotate.go`.
- If the intent is ambiguous, prefer one focused clarification over inventing a misleading message.
- In linked worktrees, treat the post-commit branch decision as a separate focused question instead of silently assuming rebase or push.
- If the user already named the source branch, prefer their stated branch over any repository heuristic.
- Keep the three-choice worktree menu first; ask for the source branch only after the user chooses that path.

## Common Mistakes

- Mixing unrelated goals into one commit because of time pressure
- Writing subjects that name files instead of behavior
- Using low-signal messages like `chore: update files`
- Running tests before a local commit when the user did not ask for it
- Guessing a worktree's source branch from naming conventions or wishful thinking
- Showing the worktree rebase-or-push menu even when the current checkout is not a linked worktree
- Treating the primary checkout branch in `git worktree list` as proof of the linked worktree's source branch
- Replacing the worktree three-choice question with `which branch did it come from?` before the user even chose a rebase path

## Examples

Good:

- `ci(git): enforce quick scoped tests before push`
- `fix(logs): clean up old rotated files automatically`
- `feat(errc): add stable error code constants for callers`

Bad:

- `fix: improve writer rotation handling`
- `feat: add new constants`
- `docs: update README`
- `fix: update writer.go and rotate.go`
