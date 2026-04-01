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
        parts = block.split("\x1f", 2)
        commit_hash = parts[0].strip()
        subject = parts[1].strip() if len(parts) > 1 else ""
        body = parts[2].strip() if len(parts) > 2 else ""
        records.append((commit_hash, subject, body))
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
