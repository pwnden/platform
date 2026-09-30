"""Check bootstrap activation and path boundaries without a Docker daemon."""

import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest

SOURCE = Path(__file__).resolve().parent.parent


class BootstrapTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.entry_directory = tempfile.TemporaryDirectory(prefix="pwnden-entry-tool-")
        cls.entry = Path(cls.entry_directory.name) / "pwnden-entry"
        subprocess.run(["go", "build", "-o", str(cls.entry), "./tools/checkout"], cwd=SOURCE, check=True)

    @classmethod
    def tearDownClass(cls):
        cls.entry_directory.cleanup()

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="pwnden-bootstrap-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name) / "checkout with spaces"
        self.root.mkdir()
        for name in ("pwnden",):
            shutil.copy2(SOURCE / name, self.root / name)
        self.tools = Path(self.temporary.name) / "path"
        self.tools.mkdir()
        for name in ("sh", "dirname", "uname", "mkdir", "mktemp", "rm", "mv"):
            (self.tools / name).symlink_to(shutil.which(name))
        self.log = Path(self.temporary.name) / "calls.jsonl"
        fake = self.tools / "docker"
        fake.write_text(f"#!{sys.executable}\n" + '''
import json, os, pathlib, shutil, sys, time
args = sys.argv[1:]
with open(os.environ["BOOTSTRAP_TEST_LOG"], "a") as log:
    log.write(json.dumps(args) + "\\n")
if args[:1] == ["info"]:
    print(os.environ.get("BOOTSTRAP_TEST_OS", "linux"))
elif args[:2] == ["compose", "version"]:
    print("fixture")
elif args[:2] == ["buildx", "version"]:
    print("fixture")
elif args[:2] == ["buildx", "build"]:
    if os.environ.get("BOOTSTRAP_TEST_BLOCK") and args[args.index("--target") + 1] == os.environ.get("BOOTSTRAP_TEST_BLOCK_TARGET", "entry"):
        pathlib.Path(os.environ["BOOTSTRAP_TEST_CHILD_PID"]).write_text(str(os.getpid()))
        time.sleep(60)
    target = args[args.index("--target") + 1]
    code = int(os.environ.get("BOOTSTRAP_TEST_BUILD_CODE", "0")) if target != "entry" else 0
    if code: sys.exit(code)
    output = pathlib.Path(args[args.index("--output") + 1])
    output.mkdir()
    if target == "entry":
        shutil.copy2(os.environ["BOOTSTRAP_TEST_ENTRY"], output / "pwnden-entry")
        sys.exit(0)
    binary = output / "pwnden"
    binary.write_text('#!/bin/sh\\nif [ "$1" = setup ]; then exit "${BOOTSTRAP_TEST_SETUP_CODE:-0}"; fi\\nif [ "$1" = stdin ]; then IFS= read -r line; printf "%s\\\\n" "$line"; exit; fi\\nprintf "%s\\\\n" "$@"\\n')
    binary.chmod(0o755)
else:
    sys.exit(125)
''')
        fake.chmod(0o755)
        self.environment = dict(os.environ, PATH=str(self.tools),
                                BOOTSTRAP_TEST_LOG=str(self.log), BOOTSTRAP_TEST_ENTRY=str(self.entry))
        self.assertIsNone(shutil.which("go", path=str(self.tools)))
        self.assertIsNone(shutil.which("git", path=str(self.tools)))

    def invoke(self, name, *args, input=None):
        return subprocess.run([str(self.root / name), *args],
                              env=self.environment, cwd=self.temporary.name,
                              capture_output=True, text=True, input=input)

    def test_setup_and_repeat_preserve_previous_build(self):
        self.assertEqual(self.invoke("pwnden", "setup").returncode, 0)
        active = self.root / "dist/active"
        first = active.read_text().strip()
        self.assertEqual(self.invoke("pwnden", "setup").returncode, 0)
        self.assertNotEqual(active.read_text().strip(), first)
        self.assertTrue((self.root / "dist" / first / "payload/pwnden").exists())
        result = self.invoke("pwnden", "exec", "example", "--", "printf", "a b")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "exec\nexample\n--\nprintf\na b\n")
        result = self.invoke("pwnden", "stdin", input="piped input with spaces\n")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "piped input with spaces\n")
        calls = [json.loads(line) for line in self.log.read_text().splitlines()]
        for call in calls:
            if call[:2] == ["buildx", "build"]:
                self.assertEqual(call[-1], str(self.root))
                self.assertTrue(Path(call[call.index("--output") + 1]).is_relative_to(self.root))
                self.assertNotIn("--allow", call)
                self.assertNotIn("--push", call)

    def test_failure_preserves_active_build_and_cleans_new_output(self):
        self.assertEqual(self.invoke("pwnden", "setup").returncode, 0)
        before = (self.root / "dist/active").read_text()
        paths = sorted(p.name for p in (self.root / "dist").iterdir())
        for variable in ("BOOTSTRAP_TEST_BUILD_CODE", "BOOTSTRAP_TEST_SETUP_CODE"):
            with self.subTest(variable=variable):
                self.environment[variable] = "42"
                self.assertEqual(self.invoke("pwnden", "setup").returncode, 42)
                self.assertEqual((self.root / "dist/active").read_text(), before)
                self.assertEqual(sorted(p.name for p in (self.root / "dist").iterdir()), paths)
                del self.environment[variable]

    def test_external_output_symlink_is_rejected(self):
        outside = Path(self.temporary.name) / "outside"
        outside.mkdir()
        (self.root / "dist").symlink_to(outside, target_is_directory=True)
        result = self.invoke("pwnden", "setup")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("inside this checkout", result.stderr)
        self.assertEqual(list(outside.iterdir()), [])

    def test_launcher_rejects_external_build_symlink(self):
        dist = self.root / "dist"
        dist.mkdir()
        (dist / "active").write_text("build.fixture\n")
        (dist / "build.fixture").symlink_to(Path(self.temporary.name), target_is_directory=True)
        result = self.invoke("pwnden", "list")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("inside this checkout", result.stderr)

    def test_invalid_inputs_fail_before_build(self):
        self.assertNotEqual(self.invoke("pwnden", "setup", "extra").returncode, 0)
        calls = [json.loads(line) for line in self.log.read_text().splitlines()]
        self.assertTrue(all(call[call.index("--target") + 1] == "entry" for call in calls))
        self.assertIn("run ./pwnden setup first", self.invoke("pwnden", "list").stderr)
        self.environment["BOOTSTRAP_TEST_OS"] = "windows"
        self.assertNotEqual(self.invoke("pwnden", "setup").returncode, 0)
        self.assertEqual(list((self.root / "dist").iterdir()), [])

    def test_interrupt_stops_build_child_and_removes_new_output(self):
        pid_file = Path(self.temporary.name) / "child.pid"
        self.environment.update(BOOTSTRAP_TEST_BLOCK="1", BOOTSTRAP_TEST_CHILD_PID=str(pid_file), BOOTSTRAP_TEST_BLOCK_TARGET="export")
        process = subprocess.Popen([str(self.root / "pwnden"), "setup"], env=self.environment,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        try:
            deadline = time.monotonic() + 5
            while not pid_file.exists() and time.monotonic() < deadline:
                time.sleep(0.02)
            self.assertTrue(pid_file.exists(), "build child did not start")
            child_pid = int(pid_file.read_text())
            process.send_signal(signal.SIGTERM)
            process.communicate(timeout=5)
            self.assertEqual(process.returncode, 143)
            self.assertEqual(list((self.root / "dist").iterdir()), [])
            with self.assertRaises(ProcessLookupError):
                os.kill(child_pid, 0)
        finally:
            if process.poll() is None:
                process.kill()
                process.communicate()
            if pid_file.exists():
                try:
                    os.kill(int(pid_file.read_text()), signal.SIGTERM)
                except ProcessLookupError:
                    pass

    def test_development_uses_local_checkout_and_separate_reference(self):
        challenges = self.root.parent / "challenges"
        challenges.mkdir()
        (challenges / "contract.toml").write_text("version=1\n")
        self.assertEqual(self.invoke("pwnden", "setup").returncode, 0)
        original = (self.root / "dist/active").read_text()
        result = self.invoke("pwnden", "dev")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("--repo\n" + str(challenges) + "\ndev\n", result.stdout)
        self.assertEqual((self.root / "dist/active").read_text(), original)
        self.assertTrue((self.root / "dist/development").is_file())


if __name__ == "__main__":
    unittest.main()
