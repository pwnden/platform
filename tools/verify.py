"""Run the same challenge lifecycle locally and on disposable CI runners."""

import argparse
import os
from pathlib import Path
import signal
import subprocess
import tempfile

interrupted = 0
child = None
cleaning = False


def handle_signal(signum, _frame):
    global interrupted
    interrupted = signum
    if child is not None and child.poll() is None and not cleaning:
        if os.name == "nt":
            child.send_signal(signal.CTRL_BREAK_EVENT)
        else:
            child.send_signal(signum)


def execute(args, *, cleanup=False, **kwargs):
    global child, cleaning
    if interrupted and not cleanup:
        raise InterruptedError("verification interrupted")
    print("+ " + " ".join(str(arg) for arg in args), flush=True)
    cleaning = cleanup
    try:
        if os.name == "nt":
            kwargs["creationflags"] = subprocess.CREATE_NEW_PROCESS_GROUP
        child = subprocess.Popen(args, **kwargs)
        if interrupted and not cleanup:
            handle_signal(interrupted, None)
        # Wait for the CLI's own cleanup before starting another command.
        code = child.wait()
        if code:
            raise subprocess.CalledProcessError(code, args)
    finally:
        child = None
        cleaning = False


def main():
    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)
    platform = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--challenges", type=Path, default=platform.parent / "challenges")
    args = parser.parse_args()
    repo = args.challenges.resolve()
    manifests = sorted((repo / "challenges").glob("*/challenge.toml"))
    if not manifests:
        parser.error("no challenge.toml files found")
    with tempfile.TemporaryDirectory(prefix="pwnden-ci-") as directory:
        binary = Path(directory) / ("pwnden.exe" if os.name == "nt" else "pwnden")
        execute(["go", "build", "-trimpath", "-o", str(binary), "./cmd/pwnden"], cwd=platform)
        for manifest in manifests:
            slug = manifest.parent.name
            command = [str(binary), "--repo", str(repo)]
            execute(command + ["validate", slug])
            execute(command + ["run", slug])
            try:
                execute(command + ["verify", slug])
            finally:
                execute(command + ["stop", slug], cleanup=True)
    if interrupted:
        raise InterruptedError("verification interrupted")
    print(f"Verified {len(manifests)} challenges.", flush=True)


if __name__ == "__main__":
    try:
        main()
    except (InterruptedError, subprocess.CalledProcessError):
        if interrupted:
            raise SystemExit(128 + interrupted)
        raise
