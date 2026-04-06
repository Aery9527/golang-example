# Go Release Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `go-release` skill, its supporting Python release-note collector, and the related docs/evals so this repo can release `develop` to `main`, summarize post-tag commits efficiently, recommend a Semantic Version, create a lightweight tag, and stop before push unless the user explicitly asks for follow-up Git actions.

**Architecture:** Keep Git orchestration inside the skill, keep deterministic commit collection inside `scripts/release-notes.py`, and keep qualitative release-note condensation in a clean-context subagent that reads `.tmp/release/raw-commits.json`. Test the Python collector end-to-end against temporary Git repositories with `unittest` so parsing, filtering, range resolution, and artifact output stay stable without depending on the main repo history.

**Tech Stack:** Git, Python 3 stdlib (`argparse`, `json`, `pathlib`, `re`, `subprocess`, `tempfile`, `unittest`), Markdown skill files, existing PowerShell `ci-test` wrapper, `.gitignore`.

**Design Spec:** [`docs/superpowers/specs/2026-04-01-go-release-design.md`](../specs/2026-04-01-go-release-design.md)

**Preflight:** Execute this plan from a dedicated worktree rooted at this repository. If you are still in the shared checkout, create or switch to an isolated worktree before Task 1.

---

## 快速導覽

- [File Map](#file-map)
- [Task 1: Add `.tmp/` ignore rules and a minimal JSON collector](#task-1-add-tmp-ignore-rules-and-a-minimal-json-collector)
- [Task 2: Complete the collector with filtering, breaking-change parsing, markdown output, and clear errors](#task-2-complete-the-collector-with-filtering-breaking-change-parsing-markdown-output-and-clear-errors)
- [Task 3: Create the `go-release` skill](#task-3-create-the-go-release-skill)
- [Task 4: Add `go-release` eval scenarios](#task-4-add-go-release-eval-scenarios)
- [Task 5: Document the release workflow and run end-to-end verification](#task-5-document-the-release-workflow-and-run-end-to-end-verification)

## File Map

```text
Create:
├── .claude/skills/go-release/SKILL.md
├── .claude/skills/go-release/evals/evals.json
├── scripts/release-notes.py
└── scripts/tests/test_release_notes.py

Modify:
├── .gitignore
└── README.md
```

File responsibilities:

- `.claude/skills/go-release/SKILL.md` defines the release workflow, preflight checks, version suggestion order, hotfix behavior, and the exact closing questions after tag creation.
- `.claude/skills/go-release/evals/evals.json` stores prompt/expected-output pairs for manual or qualitative skill review.
- `scripts/release-notes.py` is a CLI-first Python collector that reads Git history, filters release-noise commits, groups Conventional Commits, and either writes JSON to `.tmp/release/raw-commits.json` or prints grouped markdown to stdout.
- `scripts/tests/test_release_notes.py` drives the collector through subprocesses against temporary Git repositories so the hyphenated CLI filename can stay untouched and the tests verify real Git behavior.
- `.gitignore` ignores `.tmp/` because release artifacts are runtime-only and must never reach version control.
- `README.md` documents how `go-release` fits beside the existing `go-commit` and `ci-test` workflows.

Implementation notes:

- Keep `.tmp/release/` runtime-only; do not add `.gitkeep` or commit any generated release artifacts.
- Keep `scripts/release-notes.py` import-free beyond Python stdlib and callable as `python scripts/release-notes.py`.
- Keep release-note summarization out of the Python script; the script only emits structured raw data or raw grouped markdown.
- Keep tag creation lightweight (`git tag vX.Y.Z`), with the actual release merge commit message fixed at `release: vX.Y.Z`.
- Keep `go-release` from auto-pushing. The skill may offer a push follow-up, but it must not silently perform it.

[返回開頭](#快速導覽)

---

### Task 1: Add `.tmp/` ignore rules and a minimal JSON collector

**Files:**
- Modify: `.gitignore`
- Create: `scripts/release-notes.py`
- Create: `scripts/tests/test_release_notes.py`

- [ ] **Step 1: Write the first failing CLI integration tests**

Create `scripts/tests/test_release_notes.py` with this content:

```python
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from typing import Optional

SCRIPT_PATH = Path(__file__).resolve().parents[1] / "release-notes.py"
OUTPUT_PATH = Path(".tmp") / "release" / "raw-commits.json"


class ReleaseNotesScriptTests(unittest.TestCase):
    maxDiff = None

    def make_repo(self) -> Path:
        tempdir = tempfile.TemporaryDirectory()
        self.addCleanup(tempdir.cleanup)
        repo = Path(tempdir.name)
        self.run_git(repo, "init")
        self.run_git(repo, "config", "user.name", "Test User")
        self.run_git(repo, "config", "user.email", "test@example.com")
        return repo

    def run_git(self, repo: Path, *args: str) -> str:
        completed = subprocess.run(
            ["git", *args],
            cwd=repo,
            text=True,
            capture_output=True,
            check=True,
        )
        return completed.stdout.strip()

    def commit(
        self,
        repo: Path,
        subject: str,
        *,
        body: Optional[str] = None,
        filename: str = "history.txt",
    ) -> None:
        target = repo / filename
        current = target.read_text(encoding="utf-8") if target.exists() else ""
        target.write_text(current + subject + "\n", encoding="utf-8")
        self.run_git(repo, "add", target.name)

        command = ["git", "commit", "-m", subject]
        if body is not None:
            command.extend(["-m", body])

        subprocess.run(
            command,
            cwd=repo,
            text=True,
            capture_output=True,
            check=True,
        )

    def run_script(
        self,
        repo: Path,
        *args: str,
        check: bool = True,
    ) -> subprocess.CompletedProcess[str]:
        completed = subprocess.run(
            [sys.executable, str(SCRIPT_PATH), *args],
            cwd=repo,
            text=True,
            capture_output=True,
        )
        if check and completed.returncode != 0:
            self.fail(completed.stderr or completed.stdout or "release-notes.py failed unexpectedly")
        return completed

    def load_json(self, repo: Path) -> dict:
        output_path = repo / OUTPUT_PATH
        self.assertTrue(output_path.exists(), f"missing output file: {output_path}")
        return json.loads(output_path.read_text(encoding="utf-8"))

    def test_collects_full_history_when_repo_has_no_tags(self) -> None:
        repo = self.make_repo()
        self.commit(repo, "feat(logs): add bootstrap logger")

        self.run_script(repo)

        payload = self.load_json(repo)
        self.assertIsNone(payload["previous_tag"])
        self.assertEqual(len(payload["groups"]["feat"]), 1)
        self.assertEqual(payload["groups"]["feat"][0]["scope"], "logs")
        self.assertEqual(payload["groups"]["feat"][0]["summary"], "add bootstrap logger")
        self.assertEqual(payload["unparseable"], [])

    def test_groups_commits_since_latest_tag(self) -> None:
        repo = self.make_repo()
        self.commit(repo, "docs: seed repository")
        self.run_git(repo, "tag", "v1.0.0")
        self.commit(repo, "feat(logs): add file rotation policy")
        self.commit(repo, "fix(logs): preserve stderr sink")

        self.run_script(repo)

        payload = self.load_json(repo)
        self.assertEqual(payload["previous_tag"], "v1.0.0")
        self.assertEqual(
            [entry["summary"] for entry in payload["groups"]["feat"]],
            ["add file rotation policy"],
        )
        self.assertEqual(
            [entry["summary"] for entry in payload["groups"]["fix"]],
            ["preserve stderr sink"],
        )
        self.assertEqual(payload["commit_count"], 2)


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run the tests to confirm the red state**

Run:

```powershell
python -m unittest discover -s scripts/tests -p "test_release_notes.py" -v
```

Expected: FAIL because `scripts/release-notes.py` does not exist yet, with Python reporting it cannot open or execute the script file.

- [ ] **Step 3: Ignore local release artifacts**

Update `.gitignore` by adding this block directly under the existing `test-output/` rule:

```gitignore
# Temporary local release artifacts
.tmp/
```

- [ ] **Step 4: Implement the minimal JSON collector**

Create `scripts/release-notes.py` with this content:

```python
#!/usr/bin/env python3
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Dict, List, Optional, Tuple

CONVENTIONAL_RE = re.compile(
    r"^(?P<type>[a-z]+)(?:\((?P<scope>[\w\-/]+)\))?(?P<breaking>!)?: (?P<summary>.+)$"
)
GROUP_ORDER = ("feat", "fix", "refactor", "perf", "docs", "test", "chore", "build", "ci")
OUTPUT_PATH = Path(".tmp") / "release" / "raw-commits.json"


def run_git(repo_root: Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", *args],
        cwd=repo_root,
        text=True,
        capture_output=True,
        check=True,
    )
    return completed.stdout


def get_repo_root() -> Path:
    return Path(run_git(Path.cwd(), "rev-parse", "--show-toplevel").strip())


def get_previous_tag(repo_root: Path) -> Optional[str]:
    completed = subprocess.run(
        ["git", "describe", "--tags", "--abbrev=0"],
        cwd=repo_root,
        text=True,
        capture_output=True,
    )
    if completed.returncode != 0:
        return None

    tag = completed.stdout.strip()
    return tag or None


def collect_commits(repo_root: Path, revision_range: Optional[str]) -> List[Tuple[str, str]]:
    args = ["log", "--reverse", "--format=%H%x1f%s%x1e"]
    args.append(revision_range or "HEAD")
    raw = run_git(repo_root, *args)

    commits: List[Tuple[str, str]] = []
    for block in raw.split("\x1e"):
        block = block.strip()
        if not block:
            continue
        commit_hash, subject = block.split("\x1f", 1)
        commits.append((commit_hash.strip(), subject.strip()))
    return commits


def build_report(repo_root: Path) -> Dict[str, object]:
    previous_tag = get_previous_tag(repo_root)
    revision_range = f"{previous_tag}..HEAD" if previous_tag else None
    groups: Dict[str, List[Dict[str, Optional[str]]]] = {name: [] for name in GROUP_ORDER}

    for commit_hash, subject in collect_commits(repo_root, revision_range):
        match = CONVENTIONAL_RE.match(subject)
        if match is None:
            continue

        commit_type = match.group("type")
        if commit_type not in groups:
            continue

        groups[commit_type].append(
            {
                "hash": commit_hash[:7],
                "scope": match.group("scope"),
                "summary": match.group("summary"),
            }
        )

    commit_count = sum(len(entries) for entries in groups.values())
    return {
        "range": revision_range or "HEAD",
        "previous_tag": previous_tag,
        "commit_count": commit_count,
        "has_breaking": False,
        "groups": groups,
        "breaking_changes": [],
        "unparseable": [],
    }


def main() -> int:
    repo_root = get_repo_root()
    report = build_report(repo_root)

    output_path = repo_root / OUTPUT_PATH
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(output_path)
    return 0


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 5: Run the tests to verify the green state**

Run:

```powershell
python -m unittest discover -s scripts/tests -p "test_release_notes.py" -v
```

Expected: PASS for the two tests in `ReleaseNotesScriptTests`.

- [ ] **Step 6: Commit the minimal collector**

```bash
git add .gitignore scripts/release-notes.py scripts/tests/test_release_notes.py
git commit -m "feat(release): add initial release note collector"
```

[返回開頭](#快速導覽)

---

### Task 2: Complete the collector with filtering, breaking-change parsing, markdown output, and clear errors

**Files:**
- Modify: `scripts/release-notes.py`
- Modify: `scripts/tests/test_release_notes.py`

- [ ] **Step 1: Extend the test file with the remaining failing cases**

Append these methods inside `ReleaseNotesScriptTests` in `scripts/tests/test_release_notes.py`, directly below `test_groups_commits_since_latest_tag`:

```python
    def test_collects_breaking_and_unparseable_commits_without_merge_noise(self) -> None:
        repo = self.make_repo()
        self.commit(repo, "docs: seed repository")
        self.run_git(repo, "tag", "v1.0.0")
        self.commit(
            repo,
            "feat(api)!: remove legacy payload",
            body="BREAKING CHANGE: clients must send the v2 request body",
        )
        self.commit(repo, "merge: develop into main")
        self.commit(repo, "release: v2.0.0")
        self.commit(repo, "ship the thing")

        self.run_script(repo)

        payload = self.load_json(repo)
        self.assertTrue(payload["has_breaking"])
        self.assertEqual(len(payload["breaking_changes"]), 1)
        self.assertEqual(payload["breaking_changes"][0]["scope"], "api")
        self.assertEqual(payload["breaking_changes"][0]["summary"], "remove legacy payload")
        self.assertEqual(
            payload["breaking_changes"][0]["note"],
            "clients must send the v2 request body",
        )
        self.assertEqual(len(payload["unparseable"]), 1)
        self.assertEqual(payload["unparseable"][0]["raw_message"], "ship the thing")

        grouped_summaries = [
            entry["summary"]
            for entries in payload["groups"].values()
            for entry in entries
        ]
        self.assertNotIn("develop into main", grouped_summaries)
        self.assertNotIn("v2.0.0", grouped_summaries)

    def test_markdown_mode_writes_grouped_sections_to_stdout(self) -> None:
        repo = self.make_repo()
        self.commit(repo, "docs: seed repository")
        self.run_git(repo, "tag", "v1.0.0")
        self.commit(repo, "feat(logs): add retention policy")
        self.commit(repo, "fix(logs): preserve stderr sink")

        completed = self.run_script(repo, "--format=markdown")

        self.assertIn("## Features", completed.stdout)
        self.assertIn("- logs: add retention policy", completed.stdout)
        self.assertIn("## Bug Fixes", completed.stdout)
        self.assertIn("- logs: preserve stderr sink", completed.stdout)
        self.assertFalse((repo / OUTPUT_PATH).exists())

    def test_invalid_range_surfaces_a_clear_git_error(self) -> None:
        repo = self.make_repo()

        completed = self.run_script(repo, "v9.9.9..HEAD", check=False)

        self.assertNotEqual(completed.returncode, 0)
        self.assertRegex(
            (completed.stderr or completed.stdout).lower(),
            r"(unknown revision|ambiguous argument)",
        )

    def test_non_git_repo_returns_a_friendly_error(self) -> None:
        tempdir = tempfile.TemporaryDirectory()
        self.addCleanup(tempdir.cleanup)

        completed = self.run_script(Path(tempdir.name), check=False)

        self.assertNotEqual(completed.returncode, 0)
        self.assertIn("not inside a Git repository", completed.stderr)
```

- [ ] **Step 2: Run the expanded test suite and keep it red**

Run:

```powershell
python -m unittest discover -s scripts/tests -p "test_release_notes.py" -v
```

Expected: FAIL because the minimal collector does not yet parse commit bodies, exclude `merge:` / `release:` subjects, emit markdown, or surface friendly runtime errors.

- [ ] **Step 3: Replace the collector with the full implementation**

Replace `scripts/release-notes.py` with this content:

```python
#!/usr/bin/env python3
import argparse
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Dict, List, Optional, Tuple

CONVENTIONAL_RE = re.compile(
    r"^(?P<type>[a-z]+)(?:\((?P<scope>[\w\-/]+)\))?(?P<breaking>!)?: (?P<summary>.+)$"
)
BREAKING_FOOTER_RE = re.compile(r"^BREAKING CHANGE:\s*(?P<note>.+)$", re.MULTILINE)
GROUP_ORDER = ("feat", "fix", "refactor", "perf", "docs", "test", "chore", "build", "ci")
OTHER_GROUPS = ("refactor", "docs", "test", "chore", "build", "ci")
EXCLUDED_PREFIXES = ("Merge ", "merge:", "release:")
OUTPUT_PATH = Path(".tmp") / "release" / "raw-commits.json"


def parse_args(argv: List[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Collect Conventional Commits for release-note generation."
    )
    parser.add_argument(
        "revision_range",
        nargs="?",
        help="Optional git revision range such as v1.2.0..HEAD",
    )
    parser.add_argument(
        "--format",
        choices=("json", "markdown"),
        default="json",
        help="Write grouped JSON to .tmp/release/raw-commits.json or print grouped markdown to stdout.",
    )
    return parser.parse_args(argv)


def git(repo_root: Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", *args],
        cwd=repo_root,
        text=True,
        capture_output=True,
    )
    if completed.returncode != 0:
        message = completed.stderr.strip() or completed.stdout.strip()
        raise RuntimeError(message or f"git {' '.join(args)} failed")
    return completed.stdout


def resolve_repo_root(cwd: Path) -> Path:
    completed = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        cwd=cwd,
        text=True,
        capture_output=True,
    )
    if completed.returncode != 0:
        raise RuntimeError("not inside a Git repository")
    return Path(completed.stdout.strip())


def get_previous_tag(repo_root: Path) -> Optional[str]:
    completed = subprocess.run(
        ["git", "describe", "--tags", "--abbrev=0"],
        cwd=repo_root,
        text=True,
        capture_output=True,
    )
    if completed.returncode == 0:
        tag = completed.stdout.strip()
        return tag or None

    stderr = completed.stderr.strip()
    if "No names found" in stderr or "No tags can describe" in stderr:
        return None

    raise RuntimeError(stderr or "unable to resolve previous tag")


def collect_raw_records(repo_root: Path, revision_range: Optional[str]) -> List[Tuple[str, str, str]]:
    args = ["log", "--reverse", "--format=%H%x1f%s%x1f%b%x1e"]
    args.append(revision_range or "HEAD")
    raw = git(repo_root, *args)

    records: List[Tuple[str, str, str]] = []
    for block in raw.split("\x1e"):
        block = block.strip()
        if not block:
            continue
        commit_hash, subject, body = block.split("\x1f", 2)
        records.append((commit_hash.strip(), subject.strip(), body.strip()))
    return records


def make_groups() -> Dict[str, List[Dict[str, Optional[str]]]]:
    return {name: [] for name in GROUP_ORDER}


def is_excluded_subject(subject: str) -> bool:
    return subject.startswith(EXCLUDED_PREFIXES)


def parse_breaking_note(body: str) -> Optional[str]:
    match = BREAKING_FOOTER_RE.search(body)
    if match is None:
        return None
    return match.group("note").strip()


def build_report(repo_root: Path, revision_range: Optional[str]) -> Dict[str, object]:
    previous_tag = None if revision_range else get_previous_tag(repo_root)
    effective_range = revision_range or (f"{previous_tag}..HEAD" if previous_tag else None)

    groups = make_groups()
    breaking_changes: List[Dict[str, Optional[str]]] = []
    unparseable: List[Dict[str, str]] = []

    for commit_hash, subject, body in collect_raw_records(repo_root, effective_range):
        if is_excluded_subject(subject):
            continue

        match = CONVENTIONAL_RE.match(subject)
        if match is None:
            unparseable.append({"hash": commit_hash[:7], "raw_message": subject})
            continue

        commit_type = match.group("type")
        if commit_type not in groups:
            unparseable.append({"hash": commit_hash[:7], "raw_message": subject})
            continue

        entry: Dict[str, Optional[str]] = {
            "hash": commit_hash[:7],
            "scope": match.group("scope"),
            "summary": match.group("summary"),
        }
        groups[commit_type].append(entry)

        breaking_note = parse_breaking_note(body)
        if match.group("breaking") or breaking_note is not None:
            breaking_entry = dict(entry)
            breaking_entry["note"] = breaking_note
            breaking_changes.append(breaking_entry)

    commit_count = sum(len(entries) for entries in groups.values()) + len(unparseable)
    return {
        "range": effective_range or "HEAD",
        "previous_tag": previous_tag,
        "commit_count": commit_count,
        "has_breaking": bool(breaking_changes),
        "groups": groups,
        "breaking_changes": breaking_changes,
        "unparseable": unparseable,
    }


def format_entry(entry: Dict[str, Optional[str]]) -> str:
    scope = entry.get("scope")
    summary = entry.get("summary") or ""
    return f"{scope}: {summary}" if scope else summary


def to_markdown(report: Dict[str, object]) -> str:
    lines: List[str] = []

    breaking_changes = report["breaking_changes"]
    if breaking_changes:
        lines.append("## Breaking Changes")
        for entry in breaking_changes:
            line = f"- {format_entry(entry)}"
            note = entry.get("note")
            if note:
                line += f" — {note}"
            lines.append(line)
        lines.append("")

    groups = report["groups"]
    if groups["feat"]:
        lines.append("## Features")
        for entry in groups["feat"]:
            lines.append(f"- {format_entry(entry)}")
        lines.append("")

    if groups["fix"]:
        lines.append("## Bug Fixes")
        for entry in groups["fix"]:
            lines.append(f"- {format_entry(entry)}")
        lines.append("")

    if groups["perf"]:
        lines.append("## Performance")
        for entry in groups["perf"]:
            lines.append(f"- {format_entry(entry)}")
        lines.append("")

    other_lines: List[str] = []
    for group_name in OTHER_GROUPS:
        for entry in groups[group_name]:
            other_lines.append(f"- {group_name}: {format_entry(entry)}")
    if other_lines:
        lines.append("## Other Improvements")
        lines.extend(other_lines)
        lines.append("")

    if report["unparseable"]:
        lines.append("## Unparseable Commits")
        for entry in report["unparseable"]:
            lines.append(f"- {entry['hash']}: {entry['raw_message']}")

    return "\n".join(lines).strip() + "\n"


def write_json(report: Dict[str, object], repo_root: Path) -> Path:
    output_path = repo_root / OUTPUT_PATH
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return output_path


def main(argv: List[str]) -> int:
    args = parse_args(argv)
    try:
        repo_root = resolve_repo_root(Path.cwd())
        report = build_report(repo_root, args.revision_range)
    except RuntimeError as exc:
        print(str(exc), file=sys.stderr)
        return 1

    if args.format == "markdown":
        print(to_markdown(report), end="")
        return 0

    output_path = write_json(report, repo_root)
    print(output_path)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
```

- [ ] **Step 4: Run the full test suite and get back to green**

Run:

```powershell
python -m unittest discover -s scripts/tests -p "test_release_notes.py" -v
```

Expected: PASS for all six tests, including the markdown and runtime-error cases.

- [ ] **Step 5: Commit the completed collector**

```bash
git add scripts/release-notes.py scripts/tests/test_release_notes.py
git commit -m "feat(release): classify release notes and markdown output"
```

[返回開頭](#快速導覽)

---

### Task 3: Create the `go-release` skill

**Files:**
- Create: `.claude/skills/go-release/SKILL.md`

- [ ] **Step 1: Verify the new skill file does not exist yet**

Run:

```powershell
Test-Path .\.claude\skills\go-release\SKILL.md
```

Expected: `False`.

- [ ] **Step 2: Write the skill definition**

Create `.claude/skills/go-release/SKILL.md` with this content:

```markdown
---
name: go-release
description: 用於在此 Go 儲存庫中準備 develop 到 main 的 release、產生 release notes、建議 Semantic Version tag，並處理 main 上的 hotfix 發布流程。
---

# go-release

## 概覽

將 `develop` 的穩定變更發布到 `main`。先做 branch / working tree / remote / `ci-test` 檢查，再用 `python scripts/release-notes.py` 收集上一個 tag 以來的 Conventional Commits，交給乾淨 context 的 subagent 濃縮成 release notes，最後才建議版本號、merge 到 `main` 並建立 lightweight tag。

預設流程是 `develop -> main`。只有當使用者明確在 `main` 上處理 hotfix 時，才改走 hotfix 路徑。

## 使用時機

- 使用者要求發布新版本
- 使用者要求將 `develop` merge 到 `main` 並打 tag
- 使用者要求整理上一個 tag 之後的 release notes
- 使用者在 `main` 上處理 hotfix，並需要建立新 tag

## 前置檢查

1. 檢查 `git branch --show-current`。
2. 正常 release 必須從 `develop` 開始；hotfix 只允許從 `main` 開始。
3. 檢查 `git status --short`；若工作樹不乾淨，停止並要求先 commit 或 stash。
4. 執行 `git fetch --all --tags --prune`。
5. 若正常 release 且 `origin/develop` 領先本地，停止並要求先同步 `develop`。
6. 執行 `ci-test`；若失敗，停止並指向 `test-output/ci-test/`。
7. 若自上一個 tag 以來沒有新 commits，停止並明說目前沒有可發布內容。

## 工作流程

1. 完成所有前置檢查。
2. 執行 `python scripts/release-notes.py`，讓它將原始 commit 資料寫入 `.tmp/release/raw-commits.json`。
3. 啟動一個乾淨 context 的 subagent，只讀 `.tmp/release/raw-commits.json`，將結果濃縮成 `.tmp/release/release-notes.md`。
4. 先向使用者展示濃縮後的 release notes，讓使用者確認內容。
5. 只有在 release notes 確認後才建議版本號：
   - 有 breaking changes：建議 MAJOR bump
   - 無 breaking changes但有 features：建議 MINOR bump
   - 其餘情況：建議 PATCH bump
6. 使用者可接受或覆寫版本號；若格式不符 `vMAJOR.MINOR.PATCH`，要求重新輸入。
7. 正常 release 路徑：
   - `git checkout main`
   - `git pull origin main`
   - `git merge --no-ff develop -m "release: vX.Y.Z"`
   - `git tag vX.Y.Z`
8. Hotfix 路徑（當前分支是 `main`）：
   - 不做 merge
   - 直接 `git tag vX.Y.Z`
9. 若 `git merge --no-ff develop` 產生 conflict：
   - 立即通知使用者 merge 失敗
   - 列出衝突檔案
   - 協助使用者解決，或明確建議 `git merge --abort`
   - 只有在 conflict 解決並完成 merge 後，才繼續 `git tag vX.Y.Z`
10. 完成後以一個明確的問題結束，只提供三個選項：
   - 推送 `main` + tags
   - 回到 `develop` 並 merge `main`
   - 以上都做

## Release Notes 規則

- Python 腳本只做 deterministic 的 commit 收集與分組，不負責濃縮語意。
- Subagent 讀取 `.tmp/release/raw-commits.json` 後，應將 notes 按 `Breaking Changes`、`Features`、`Bug Fixes`、`Performance`、`Other Improvements` 排序。
- `merge:`、`release:` 與 Git 自動產生的 merge commits 不應進入 release notes。
- `unparseable` commits 必須保留，供使用者人工檢查。

## Tag 與 Push 規則

- Tag 固定使用 lightweight tag：`git tag vX.Y.Z`
- Tag 必須打在正常 release 的 merge commit 上，或 hotfix 路徑的 `main` HEAD 上
- 不要自動建立 annotated tag
- 不要自動推送；只有在使用者選擇後才執行 `git push origin main --tags`

## 收尾問題

- 正常 release：`Release vX.Y.Z 完成。下一步：推送 main + tags、回到 develop 並同步 merge commit，或以上都做？`
- Hotfix release：`Hotfix vX.Y.Z 完成。下一步：推送 main + tags、將 main merge 回 develop，或以上都做？`

## 常見錯誤

- 在 feature branch 上直接做 release
- release notes 還沒確認就先決定版本號
- 把 `merge:` / `release:` 這類流程 commit 當成使用者可見變更
- 在未執行 `ci-test` 的情況下直接 merge 到 `main`
- 未經使用者同意就自動推送 `main` 或 tags
```

- [ ] **Step 3: Smoke-check that the required release workflow anchors exist**

Run:

```powershell
Select-String -Path .\.claude\skills\go-release\SKILL.md -Pattern "ci-test|scripts/release-notes.py|release: vX.Y.Z|git tag vX.Y.Z"
```

Expected: four matches covering the test gate, the Python collector, the merge commit message, and the lightweight tag command.

- [ ] **Step 4: Commit the skill**

```bash
git add .claude/skills/go-release/SKILL.md
git commit -m "feat(release): add guided release skill"
```

[返回開頭](#快速導覽)

---

### Task 4: Add `go-release` eval scenarios

**Files:**
- Create: `.claude/skills/go-release/evals/evals.json`

- [ ] **Step 1: Verify the eval file is still missing**

Run:

```powershell
Test-Path .\.claude\skills\go-release\evals\evals.json
```

Expected: `False`.

- [ ] **Step 2: Write the eval scenarios**

Create `.claude/skills/go-release/evals/evals.json` with this content:

```json
{
  "skill_name": "go-release",
  "evals": [
    {
      "id": 1,
      "prompt": "我目前在 feature/log-cleanup 分支，想直接發布 v1.4.0。請幫我做 go-release。",
      "expected_output": "停止 release 流程，明確指出正常 release 必須從 develop 開始；只有 main 上的 hotfix 才能繞過 develop 起點。",
      "files": []
    },
    {
      "id": 2,
      "prompt": "我在 develop 上，但 working tree 還有未提交的變更。請直接幫我 release。",
      "expected_output": "停止流程，要求先 commit 或 stash，不能在 dirty working tree 上繼續 release。",
      "files": []
    },
    {
      "id": 3,
      "prompt": "我在 develop 上，想發布新版本。ci-test 剛剛失敗了，你接下來會怎麼做？",
      "expected_output": "停止流程，指出 release 不能繼續，並引導使用者查看 test-output/ci-test/ artifacts。",
      "files": []
    },
    {
      "id": 4,
      "prompt": "我在 develop 上，但自上一個 tag 之後完全沒有任何新 commits。請幫我 release。",
      "expected_output": "停止流程，明確指出自上一個 tag 之後沒有可發布內容，因此不需要 release。",
      "files": []
    },
    {
      "id": 5,
      "prompt": "我在 develop 上，上一個 tag 是 v1.2.0，從那之後有 feat 和 fix。請幫我做完整 release。",
      "expected_output": "先執行 release-notes 收集與 subagent 濃縮，再展示 release notes，之後才建議 MINOR bump，並在使用者確認後 merge 到 main、建立 lightweight tag，最後提供推送/同步 develop 的三選項。",
      "files": []
    },
    {
      "id": 6,
      "prompt": "release notes 裡有 breaking changes，你會建議哪種版本號升級？",
      "expected_output": "明確建議 MAJOR bump，且理由來自已確認的 release notes，而不是原始 commit 數量或檔案數量。",
      "files": []
    },
    {
      "id": 7,
      "prompt": "你建議 v2.1.0，但我想改成 v2.0.5。接下來要怎麼處理？",
      "expected_output": "接受使用者覆寫版本號，只要格式符合 vMAJOR.MINOR.PATCH，就依使用者指定的版本繼續。",
      "files": []
    },
    {
      "id": 8,
      "prompt": "在 `git merge --no-ff develop -m \"release: v1.5.0\"` 的時候發生 conflict。go-release 接下來應該怎麼做？",
      "expected_output": "明確通知使用者 merge 失敗、列出衝突檔案、協助解決或建議 `git merge --abort`，並且只有在 conflict 解決後才繼續 tag。",
      "files": []
    },
    {
      "id": 9,
      "prompt": "我目前就在 main 上修了一個 hotfix，想直接發布。go-release 會怎麼走？",
      "expected_output": "辨識 hotfix 路徑：仍要做前置檢查與 release notes，但跳過 merge develop 到 main，直接在 main HEAD 上建立 tag，最後提示是否把 main merge 回 develop。",
      "files": []
    },
    {
      "id": 10,
      "prompt": "從上一個 tag 之後只有 merge: 與 release: 這種流程 commit，沒有真正的新功能。go-release 應該怎麼反應？",
      "expected_output": "忽略流程性 commit；若沒有任何可發布的有效 commits，應停止並說明目前沒有可發布內容。",
      "files": []
    },
    {
      "id": 11,
      "prompt": "release notes 還沒確認，我想先決定版本號。你會照做嗎？",
      "expected_output": "拒絕跳過流程，明確說明必須先確認 release notes，再根據 notes 建議版本號。",
      "files": []
    },
    {
      "id": 12,
      "prompt": "release 和 tag 都完成了。你接下來會直接幫我 push 嗎？",
      "expected_output": "不自動推送；只提供明確的後續選項，例如推送 main + tags、同步 main 回 develop、或以上都做，等待使用者選擇。",
      "files": []
    }
  ]
}
```

- [ ] **Step 3: Validate the eval JSON**

Run:

```powershell
python -m json.tool .\.claude\skills\go-release\evals\evals.json > $null
```

Expected: exit code `0` with no stderr output.

- [ ] **Step 4: Commit the eval coverage**

```bash
git add .claude/skills/go-release/evals/evals.json
git commit -m "test(release): add go-release eval scenarios"
```

[返回開頭](#快速導覽)

---

### Task 5: Document the release workflow and run end-to-end verification

**Files:**
- Modify: `README.md`
- Verify: `.gitignore`
- Verify: `scripts/release-notes.py`
- Verify: `scripts/tests/test_release_notes.py`
- Verify: `.claude/skills/go-release/SKILL.md`
- Verify: `.claude/skills/go-release/evals/evals.json`

- [ ] **Step 1: Confirm README does not already document the release workflow**

Run:

```powershell
Select-String -Path .\README.md -Pattern "^### Release Workflow$"
```

Expected: no matches.

- [ ] **Step 2: Add the release workflow section to README**

Insert this section into `README.md` immediately before `## 作為 Library 使用`:

```markdown
### Release Workflow

若要將 `develop` 發布到 `main`，請先保持 working tree 乾淨，確認 `develop` 已同步，並先執行 `ci-test`。之後可使用支援 repo skills 的 agent 觸發 `go-release`。

`go-release` 會：

- 驗證目前是正常的 `develop -> main` release，或辨識 `main` 上的 hotfix 路徑
- 執行 `python scripts/release-notes.py`，將上一個 tag 之後的 commit 收集到 `.tmp/release/raw-commits.json`
- 在乾淨 context 中濃縮 release notes，再根據確認後的 notes 建議下一個 `vMAJOR.MINOR.PATCH`
- 以 `git merge --no-ff develop -m "release: vX.Y.Z"` 將 release 併入 `main`，或在 hotfix 路徑直接於 `main` 建立 tag
- 建立 lightweight tag `vX.Y.Z`
- 完成後再詢問是否推送 `main` + tags，以及是否把 `main` merge 回 `develop`

`.tmp/` 為本地暫存 artifacts，已加入 `.gitignore`。
```

- [ ] **Step 3: Re-run the Python collector test suite**

Run:

```powershell
python -m unittest discover -s scripts/tests -p "test_release_notes.py" -v
```

Expected: PASS for all six tests.

- [ ] **Step 4: Generate the current repo’s raw release JSON**

Run:

```powershell
python .\scripts\release-notes.py
```

Expected:

- exit code `0`
- `.tmp\release\raw-commits.json` exists
- the printed path points to the file under `.tmp\release\`

- [ ] **Step 5: Validate the skill eval JSON one more time**

Run:

```powershell
python -m json.tool .\.claude\skills\go-release\evals\evals.json > $null
```

Expected: exit code `0`.

- [ ] **Step 6: Run the repository’s scoped Go test gate**

Run:

```powershell
pwsh -File .\scripts\ci-test.ps1
```

Expected:

- exit code `0`
- `test-output\ci-test\command.txt` refreshed
- `test-output\ci-test\exit-code.txt` refreshed

- [ ] **Step 7: Commit the workflow docs and final wiring**

```bash
git add README.md
git commit -m "docs(release): document guided release workflow"
```

[返回開頭](#快速導覽)
