"""Build and verify the native package and HTTP/PTY delivery used by CI."""

import argparse
from pathlib import Path
import signal
import sys

import verify


def main():
    platform = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--challenges', type=Path, default=platform.parent / 'challenges')
    parser.add_argument('--out', type=Path, default=platform / 'dist')
    args = parser.parse_args()
    output = args.out.resolve()
    output.mkdir(parents=True, exist_ok=True)
    signal.signal(signal.SIGINT, verify.handle_signal)
    signal.signal(signal.SIGTERM, verify.handle_signal)
    driver = output / 'smoke-terminal'
    package = output / 'pwnden-linux-amd64.tar.gz'
    verify.execute(['go', 'run', './tools/package', '--challenges', str(args.challenges.resolve()),
                    '--out', str(output), '--target', 'linux/amd64'], cwd=platform)
    verify.execute(['go', 'build', '-o', str(driver), './tools/smoke_terminal'], cwd=platform)
    verify.execute([sys.executable, '-B', 'tools/smoke_package.py', '--package', str(package)], cwd=platform)
    verify.execute([sys.executable, '-B', 'tools/smoke_http.py', '--package', str(package),
                    '--terminal-driver', str(driver)], cwd=platform)
    print('Native package, HTTP and PTY delivery checks passed.', flush=True)


if __name__ == '__main__':
    main()
