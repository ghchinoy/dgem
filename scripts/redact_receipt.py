#!/usr/bin/env python3
"""Redact or detect internal identifiers in files before publication.

Patterns are NOT stored in this repository. They are loaded from a gitignored
file (default: scratch/leak-patterns.txt, override with DGEM_LEAK_PATTERNS_FILE).
Format, one per line:  <python-regex><TAB><replacement>   ('#' comments allowed;
replacement optional, in which case the pattern is detect-only).

Usage:
  scripts/redact_receipt.py <file.json> [--in-place | --output <path>]
  scripts/redact_receipt.py benchmarks/runs/*/*.json --in-place
  scripts/redact_receipt.py --check            # scan tracked files (used by `make check-public`)
"""

import argparse
import os
import re
import subprocess
import sys

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
DEFAULT_PATTERNS = os.path.join(REPO, "scratch", "leak-patterns.txt")
# Paths excluded from --check: raw JSON receipts, internal scratch, and this tool.
# Markdown under benchmarks/ (run READMEs, deployment logs) is public prose and is scanned.
CHECK_EXCLUDES = [":!benchmarks/*.json", ":!benchmarks/*.jsonl", ":!scratch"]


def load_patterns(path):
    patterns = []
    with open(path, encoding="utf-8") as f:
        for raw in f:
            line = raw.rstrip("\n")
            if not line.strip() or line.lstrip().startswith("#"):
                continue
            regex, _, repl = line.partition("\t")
            patterns.append((re.compile(regex), repl or None))
    return patterns


def redact_text(text, patterns):
    for pattern, repl in patterns:
        if repl is not None:
            text = pattern.sub(repl, text)
    return text


def check_repo(patterns):
    files = subprocess.check_output(
        ["git", "ls-files", "--", "."] + CHECK_EXCLUDES, cwd=REPO, text=True
    ).splitlines()
    hits = 0
    for rel in files:
        path = os.path.join(REPO, rel)
        try:
            with open(path, encoding="utf-8") as f:
                lines = f.readlines()
        except (UnicodeDecodeError, FileNotFoundError, IsADirectoryError):
            continue
        for n, line in enumerate(lines, 1):
            for pattern, _ in patterns:
                if pattern.search(line):
                    print(f"{rel}:{n}: matches internal pattern #{patterns.index((pattern, _)) + 1}")
                    hits += 1
                    break
    return hits


def main():
    p = argparse.ArgumentParser(description="Redact or detect internal identifiers")
    p.add_argument("files", nargs="*", help="Files to redact")
    p.add_argument("--in-place", "-i", action="store_true", help="Overwrite files in place")
    p.add_argument("--output", "-o", help="Output path (single input only)")
    p.add_argument("--check", action="store_true", help="Scan tracked files; exit 1 on any match")
    p.add_argument("--patterns", default=os.environ.get("DGEM_LEAK_PATTERNS_FILE", DEFAULT_PATTERNS))
    args = p.parse_args()

    if not os.path.isfile(args.patterns):
        print(f"WARNING: pattern file not found ({args.patterns}); skipping. "
              "Internal maintainers: see scratch/internal-deployment.md.", file=sys.stderr)
        return 0
    patterns = load_patterns(args.patterns)

    if args.check:
        hits = check_repo(patterns)
        if hits:
            print(f"ERROR: {hits} line(s) contain internal identifiers.", file=sys.stderr)
            return 1
        print(f"✅ PASSED: no internal identifiers found ({len(patterns)} patterns checked).")
        return 0

    if not args.files:
        p.error("no input files (or use --check)")
    if args.output and len(args.files) > 1:
        p.error("--output can only be used with a single input file")
    for path in args.files:
        if not os.path.isfile(path):
            print(f"Warning: {path} is not a file, skipping", file=sys.stderr)
            continue
        with open(path, encoding="utf-8") as f:
            redacted = redact_text(f.read(), patterns)
        if args.in_place:
            with open(path, "w", encoding="utf-8") as f:
                f.write(redacted)
            print(f"Redacted in-place: {path}")
        elif args.output:
            with open(args.output, "w", encoding="utf-8") as f:
                f.write(redacted)
            print(f"Wrote redacted output to: {args.output}")
        else:
            sys.stdout.write(redacted)
    return 0


if __name__ == "__main__":
    sys.exit(main())
