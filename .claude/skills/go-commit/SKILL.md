---
name: go-commit
description: Use when preparing a git commit in this Go repository, especially when the user asks to commit changes, choose commit boundaries, stage files, or write a Conventional Commit message that will read well in release notes.
---

# go-commit

## Overview

Turn the current working tree into one high-signal commit. Prioritize release-note-ready history: subjects should describe capability, behavior, or architectural meaning, not file edits.

Local commits stay fast. Do not run `ci-test` or `dev-test` just to create a local commit unless the user explicitly asks. Push-time verification is handled by the repo's `pre-push` hook.

## When to Use

- The user asks to commit current changes
- The user wants a commit message or help staging the right files
- The user wants to split mixed changes into separate commits
- The user wants commit history that can be summarized into release notes

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
10. If the user also plans to push, remind them that `pre-push` will run `ci-test`.

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

## Message Rules

- Keep `scope` when the affected area is clear.
- Optimize the subject for future release notes.
- Prefer describing what changed for users or operators, not which files moved.
- Avoid generic summaries like `update`, `tweak`, `misc`, or `fix stuff`.
- Avoid file-driven subjects like `update writer.go and rotate.go`.
- If the intent is ambiguous, prefer one focused clarification over inventing a misleading message.

## Common Mistakes

- Mixing unrelated goals into one commit because of time pressure
- Writing subjects that name files instead of behavior
- Using low-signal messages like `chore: update files`
- Running tests before a local commit when the user did not ask for it

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
