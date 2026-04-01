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
