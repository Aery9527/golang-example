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
