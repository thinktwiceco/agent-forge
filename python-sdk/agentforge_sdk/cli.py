"""CLI commands for ThinkTwice SDK development tasks."""

import subprocess
import sys
from pathlib import Path


def _run_command(cmd: list[str], description: str) -> int:
    """Run a command and return exit code."""
    print(f"Running: {description}")
    print(f"Command: {' '.join(cmd)}\n")
    result = subprocess.run(cmd, cwd=Path(__file__).parent.parent)
    return result.returncode


def test() -> int:
    """Run tests."""
    return _run_command(["pytest", "tests/", "-v"], "Tests")


def test_cov() -> int:
    """Run tests with coverage."""
    return _run_command(
        [
            "pytest",
            "tests/",
            "--cov=thinktwice_sdk",
            "--cov-report=html",
            "--cov-report=term",
        ],
        "Tests with coverage",
    )


def test_fast() -> int:
    """Run tests with short traceback."""
    return _run_command(["pytest", "tests/", "-v", "--tb=short"], "Fast tests")


def lint() -> int:
    """Run linting."""
    return _run_command(["ruff", "check", "."], "Linting")


def lint_fix() -> int:
    """Run linting with auto-fix."""
    return _run_command(["ruff", "check", ".", "--fix"], "Linting (auto-fix)")


def format_code() -> int:
    """Format code."""
    return _run_command(["ruff", "format", "."], "Format code")


def format_check() -> int:
    """Check code formatting."""
    return _run_command(["ruff", "format", ".", "--check"], "Format check")


def check() -> int:
    """Run linting and tests."""
    lint_code = lint()
    if lint_code != 0:
        return lint_code
    return test()


def all_checks() -> int:
    """Run all checks (lint, format check, test)."""
    lint_code = lint()
    if lint_code != 0:
        return lint_code
    format_code_result = format_check()
    if format_code_result != 0:
        return format_code_result
    return test()


def main() -> None:
    """Main CLI entry point."""
    if len(sys.argv) < 2:
        print("Usage: thinktwice-sdk <command>")
        print("\nAvailable commands:")
        print("  test        Run tests")
        print("  test-cov    Run tests with coverage")
        print("  test-fast   Run tests with short traceback")
        print("  lint        Run linting")
        print("  lint-fix    Run linting with auto-fix")
        print("  format      Format code")
        print("  format-check Check code formatting")
        print("  check       Run linting and tests")
        print("  all         Run all checks")
        sys.exit(1)

    command = sys.argv[1]
    commands = {
        "test": test,
        "test-cov": test_cov,
        "test-fast": test_fast,
        "lint": lint,
        "lint-fix": lint_fix,
        "format": format_code,
        "format-check": format_check,
        "check": check,
        "all": all_checks,
    }

    if command not in commands:
        print(f"Unknown command: {command}")
        sys.exit(1)

    exit_code = commands[command]()
    sys.exit(exit_code)


if __name__ == "__main__":
    main()

