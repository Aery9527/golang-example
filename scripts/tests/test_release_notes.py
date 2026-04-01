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

    def test_markdown_mode_includes_breaking_changes_section(self) -> None:
        repo = self.make_repo()
        self.commit(repo, "docs: seed repository")
        self.run_git(repo, "tag", "v1.0.0")
        self.commit(
            repo,
            "feat(api)!: remove legacy payload",
            body="BREAKING CHANGE: clients must send the v2 request body",
        )
        self.commit(repo, "feat(logs): add retention policy")

        completed = self.run_script(repo, "--format=markdown")

        self.assertIn("## Breaking Changes", completed.stdout)
        self.assertIn("- api: remove legacy payload — clients must send the v2 request body", completed.stdout)
        self.assertIn("## Features", completed.stdout)
        self.assertIn("- logs: add retention policy", completed.stdout)
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


if __name__ == "__main__":
    unittest.main()
