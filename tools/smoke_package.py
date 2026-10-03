"""Exercise a native Linux package without Go, Git, or an explicit problem checkout."""

import argparse
import os
import re
from pathlib import Path
import shutil
import signal
import subprocess
import tarfile
import tempfile

active_child = None
interrupted = 0
cleaning = False


def handle_signal(signum, _frame):
    global interrupted
    interrupted = signum
    if active_child is not None and active_child.poll() is None and not cleaning:
        active_child.send_signal(signum)


def execute(binary, environment, args, *, expected=0, cleanup=False):
    global active_child, cleaning
    if interrupted and not cleanup:
        raise InterruptedError("package smoke check interrupted")
    print("+ " + binary.name + " " + " ".join(args), flush=True)
    cleaning = cleanup
    try:
        active_child = subprocess.Popen(
            [str(binary), *args], env=environment,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
        )
        if interrupted and not cleanup:
            active_child.send_signal(interrupted)
        output, diagnostic = active_child.communicate()
        print(output, end="", flush=True)
        print(diagnostic, end="", flush=True)
        if active_child.returncode != expected:
            raise subprocess.CalledProcessError(active_child.returncode, args)
        if interrupted and not cleanup:
            raise InterruptedError("package smoke check interrupted")
        return output
    finally:
        active_child = None
        cleaning = False


def extract_package(package, directory):
    with tarfile.open(package, "r:gz") as archive:
        for member in archive:
            parts = Path(member.name).parts
            if (not member.isfile() or len(parts) != 2 or parts[0] in (".", "..")
                    or parts[1] not in ("pwnden", "catalog.tar.gz", "distribution.json")):
                raise ValueError(f"unexpected package entry: {member.name}")
            target = directory / member.name
            target.parent.mkdir(parents=True, exist_ok=True)
            with archive.extractfile(member) as source, target.open("xb") as output:
                shutil.copyfileobj(source, output)
            target.chmod(member.mode & 0o777)
    binaries = list(directory.glob("*/pwnden"))
    if len(binaries) != 1:
        raise ValueError("package must contain one executable")
    return binaries[0]


def verified_answer(output, slug):
    """Trusted verification supplies test data; player exec cannot read solutions."""
    match = re.fullmatch(r'verified ' + re.escape(slug) + r': (pwnden\{[^\s{}]+\})'
                         r'(?:\npatched attack failed; functional check passed)?', output.strip())
    if match is None:
        raise ValueError('verification did not return one answer for ' + slug)
    return match.group(1)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--package", type=Path)
    source.add_argument("--checkout", type=Path)
    args = parser.parse_args()
    checkout = args.checkout.resolve() if args.checkout else None
    package = args.package.resolve() if args.package else None
    docker = shutil.which("docker")
    if docker is None:
        parser.error("Docker is required")
    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)
    parent = checkout / "dist" if checkout else package.parent
    parent.mkdir(exist_ok=True)
    work = Path(tempfile.mkdtemp(prefix=".smoke-", dir=parent))
    started = False
    preserve = False
    try:
        binary = checkout / "pwnden" if checkout else extract_package(package, work)
        tool_path = work / "path"
        tool_path.mkdir()
        (tool_path / "docker").symlink_to(Path(docker).resolve())
        if checkout:
            for name in ("sh", "dirname", "uname", "mkdir", "mktemp", "rm", "mv"):
                executable = shutil.which(name)
                if executable is None:
                    parser.error(f"checkout setup requires {name}")
                (tool_path / name).symlink_to(executable)
        environment = dict(os.environ, PATH=str(tool_path),
                           XDG_CONFIG_HOME=str(work / "config"),
                           XDG_CACHE_HOME=str(work / "cache"))
        assert shutil.which("go", path=environment["PATH"]) is None
        assert shutil.which("git", path=environment["PATH"]) is None
        for name in ("node", "pnpm", "python3"):
            assert shutil.which(name, path=environment["PATH"]) is None
        if checkout:
            execute(binary, environment, ["setup"])
            execute(binary, environment, ["setup"])
        else:
            execute(binary, environment, ["setup"])
            execute(binary, environment, ["setup"])
        listing = execute(binary, environment, ["list"])
        assert "note-vault" in listing and "rotor-lock" in listing
        execute(binary, environment, ["exec", "rotor-lock", "--", "python3", "files/checker.py", "wrong"], expected=1)
        file_flag = verified_answer(execute(binary, environment, ["verify", "rotor-lock"]), "rotor-lock")
        execute(binary, environment, ["submit", "rotor-lock", "wrong"], expected=1)
        execute(binary, environment, ["submit", "rotor-lock", file_flag])
        # The fresh data root makes this check the owner of any attempted start.
        started = True
        try:
            execute(binary, environment, ["run", "note-vault"])
            service_flag = verified_answer(execute(binary, environment, ["verify", "note-vault"]), "note-vault")
            execute(binary, environment, ["submit", "note-vault", service_flag])
        finally:
            try:
                execute(binary, environment, ["stop", "note-vault"], cleanup=True)
                started = False
            except BaseException:
                preserve = True
                print(f"Cleanup failed; keep package and run state at {work}", flush=True)
                raise
        print("Package smoke check passed without Go or Git.", flush=True)
    finally:
        if not preserve and not started:
            shutil.rmtree(work)


if __name__ == "__main__":
    main()
