#!/usr/bin/env python3
"""Render a GitHub Actions job summary from a Go test JUnit report and coverage percentage.

Usage: print_test_summary.py <junit.xml> <coverage_pct> <min_coverage_pct> <test_exit_code>
"""
import sys
import xml.etree.ElementTree as ET


def escape(text: str) -> str:
    return text.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def parse_failures(junit_path):
    failures = []
    total_tests = 0
    try:
        root = ET.parse(junit_path).getroot()
    except (FileNotFoundError, ET.ParseError):
        return failures, total_tests

    suites = root.findall("testsuite") if root.tag == "testsuites" else [root]
    for suite in suites:
        package = suite.get("name", "unknown")
        for case in suite.findall("testcase"):
            total_tests += 1
            failure = case.find("failure")
            if failure is not None:
                reason = (failure.text or failure.get("message") or "").strip()
                failures.append({
                    "package": package,
                    "test": case.get("name", "unknown"),
                    "reason": reason,
                })
    return failures, total_tests


def to_float(value, default=0.0):
    try:
        return float(value)
    except (TypeError, ValueError):
        return default


def main() -> int:
    if len(sys.argv) != 5:
        print("usage: print_test_summary.py <junit.xml> <coverage_pct> <min_coverage_pct> <test_exit_code>",
              file=sys.stderr)
        return 2

    sys.stdout.reconfigure(encoding="utf-8")

    junit_path, coverage_arg, min_coverage_arg, exit_code_arg = sys.argv[1:5]
    coverage = to_float(coverage_arg)
    min_coverage = to_float(min_coverage_arg)
    exit_code = to_float(exit_code_arg, default=1.0)

    failures, total_tests = parse_failures(junit_path)
    coverage_ok = coverage >= min_coverage
    tests_ok = exit_code == 0
    overall_ok = coverage_ok and tests_ok

    lines = [
        "## Unit Test Results",
        "",
        f"{'✅' if overall_ok else '❌'} **{len(failures)} failed / {total_tests} total** "
        f"— coverage {'✅' if coverage_ok else '❌'} **{coverage:.1f}%** (min {min_coverage:.1f}%)",
        "",
    ]

    if failures:
        lines.append(f"### ❌ Failed Tests ({len(failures)})")
        lines.append("")
        for f in failures:
            lines.append(f"<details><summary>❌ <code>{escape(f['test'])}</code> — {escape(f['package'])}</summary>")
            lines.append("")
            lines.append("```text")
            lines.append(f["reason"] or "(no failure message captured)")
            lines.append("```")
            lines.append("</details>")
            lines.append("")
    elif not tests_ok:
        lines.append(f"### ⚠️ Test run exited with code {int(exit_code)}")
        lines.append("")
        lines.append("No individual failing tests were captured in the JUnit report — this usually means a "
                      "build/compile error occurred before tests ran. Check the raw job logs for details.")
        lines.append("")
    else:
        lines.append("All tests passed. 🎉")
        lines.append("")

    print("\n".join(lines))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
