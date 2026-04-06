"""Template-only regression tests for init scaffolding.

This file exists only in the template repository. The init scripts prune it
from generated repositories, so post-init docs should describe only the script
tests that remain after initialization.
"""

import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
INIT_TEMPLATE_PATHS = (
    REPO_ROOT / "init.sh",
    REPO_ROOT / "init.ps1",
)
ERROR_FLOW_CONTINUES_PATTERN = re.compile(
    r'if err := h\.Handle\(\); err != nil \{\s+'
    r'logs\.ErrorWith\("application stopped", func\(\) \(error, \[\]any\) \{\s+'
    r'return err, \[\]any\{"component", "app"\}\s+'
    r'\}\)\s+'
    r'\}\s+'
    r'logs\.Info\("application finished", func\(\) \[\]any \{\s+'
    r'return \[\]any\{"component", "app"\}',
    re.MULTILINE,
)


class InitTemplateTests(unittest.TestCase):
    maxDiff = None

    def make_workspace(self) -> Path:
        tempdir = tempfile.TemporaryDirectory()
        self.addCleanup(tempdir.cleanup)
        workspace = Path(tempdir.name)

        tracked_files = subprocess.run(
            ["git", "ls-files"],
            cwd=REPO_ROOT,
            text=True,
            encoding="utf-8",
            capture_output=True,
            check=True,
        ).stdout.splitlines()

        for relative_path in tracked_files:
            source = REPO_ROOT / relative_path
            target = workspace / relative_path
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, target)

        tests_dir = workspace / "scripts" / "tests"
        tests_dir.mkdir(parents=True, exist_ok=True)
        (tests_dir / "template_only_test.py").write_text("placeholder\n", encoding="utf-8")
        pycache_dir = tests_dir / "__pycache__"
        pycache_dir.mkdir(exist_ok=True)
        (pycache_dir / "template_only_test.cpython-312.pyc").write_bytes(b"pyc")

        return workspace

    def run_init(self, workspace: Path, *command: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            list(command),
            cwd=workspace,
            text=True,
            encoding="utf-8",
            errors="replace",
            capture_output=True,
        )

    def assert_scripts_tests_pruned(self, workspace: Path) -> None:
        tests_dir = workspace / "scripts" / "tests"
        remaining_files = sorted(
            str(path.relative_to(tests_dir)).replace("\\", "/")
            for path in tests_dir.rglob("*")
            if path.is_file()
        )
        self.assertEqual(["test_release_notes.py"], remaining_files)

    def test_main_template_continues_to_finished_log_after_error_log(self) -> None:
        for template_path in INIT_TEMPLATE_PATHS:
            with self.subTest(template=template_path.name):
                content = template_path.read_text(encoding="utf-8-sig")
                self.assertRegex(content, ERROR_FLOW_CONTINUES_PATTERN)

    def test_powershell_init_succeeds_outside_git_repo_and_prunes_script_tests(self) -> None:
        workspace = self.make_workspace()
        completed = self.run_init(workspace, "powershell.exe", "-File", "init.ps1")
        self.assertEqual(
            completed.returncode,
            0,
            msg=completed.stderr or completed.stdout or "init.ps1 failed unexpectedly",
        )
        main_go = (workspace / "cmd" / "app" / "main.go").read_text(encoding="utf-8")
        self.assertRegex(main_go, ERROR_FLOW_CONTINUES_PATTERN)
        self.assert_scripts_tests_pruned(workspace)

    def test_bash_init_succeeds_outside_git_repo_and_prunes_script_tests(self) -> None:
        bash = shutil.which("bash")
        if bash is None:
            self.skipTest("bash is not available")

        workspace = self.make_workspace()
        completed = self.run_init(workspace, bash, "init.sh")
        self.assertEqual(
            completed.returncode,
            0,
            msg=completed.stderr or completed.stdout or "init.sh failed unexpectedly",
        )
        main_go = (workspace / "cmd" / "app" / "main.go").read_text(encoding="utf-8")
        self.assertRegex(main_go, ERROR_FLOW_CONTINUES_PATTERN)
        self.assert_scripts_tests_pruned(workspace)


if __name__ == "__main__":
    unittest.main()
