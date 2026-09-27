#!/usr/bin/env python3
"""Redact internal Google Cloud project IDs, numbers, endpoint IDs, and internal domains
from benchmark receipts before publication.

Usage:
  scripts/redact_receipt.py <receipt.json> [--in-place] [--output <redacted.json>]
  scripts/redact_receipt.py benchmarks/runs/*/*.json --in-place
"""

import argparse
import os
import re
import sys

REDACTIONS = [
    # Dedicated endpoint DNS and IDs
    (re.compile(r"4423577720856772608\.us-central1-882920967572\.prediction\.vertexai\.goog"), "<endpoint-id>.<region>-<project-number>.prediction.vertexai.goog"),
    (re.compile(r"4217256562927861760\.us-central1-882920967572\.prediction\.vertexai\.goog"), "<legacy-endpoint-id>.<region>-<project-number>.prediction.vertexai.goog"),
    (re.compile(r"4423577720856772608"), "<endpoint-id>"),
    (re.compile(r"4217256562927861760"), "<legacy-endpoint-id>"),
    (re.compile(r"5387194109486170112"), "<model-id>"),
    (re.compile(r"2976360933959401472"), "<legacy-model-id>"),
    # Projects and numbers
    (re.compile(r"genai-blackbelt-fishfooding"), "<project-id>"),
    (re.compile(r"882920967572"), "<project-number>"),
    # Cloud Run internal hostnames
    (re.compile(r"dgemma-[a-z0-9]+-uc\.a\.run\.app"), "dgemma-<hash>-uc.a.run.app"),
    (re.compile(r"dgemma-[0-9]+\.us-central1\.run\.app"), "dgemma-<project-number>.us-central1.run.app"),
    # Internal domains and emails
    (re.compile(r"dgemma\.aaie\.cloud"), "<your-dgem-gateway>"),
    (re.compile(r"aaie-decision-model@google\.com"), "<authorized-group>@example.com"),
]

def redact_text(text: str) -> str:
    for pattern, replacement in REDACTIONS:
        text = pattern.sub(replacement, text)
    return text

def redact_file(path: str, in_place: bool = False, output: str = None):
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()
    redacted = redact_text(content)
    if in_place:
        with open(path, "w", encoding="utf-8") as f:
            f.write(redacted)
        print(f"Redacted in-place: {path}")
    elif output:
        with open(output, "w", encoding="utf-8") as f:
            f.write(redacted)
        print(f"Wrote redacted output to: {output}")
    else:
        sys.stdout.write(redacted)

def main():
    parser = argparse.ArgumentParser(description="Redact internal GCP IDs from benchmark receipts")
    parser.add_argument("files", nargs="+", help="Receipt JSON file(s) to redact")
    parser.add_argument("--in-place", "-i", action="store_true", help="Overwrite file in-place")
    parser.add_argument("--output", "-o", help="Output file path (valid only when single file specified)")
    args = parser.parse_args()

    if args.output and len(args.files) > 1:
        parser.error("--output can only be used with a single input file")

    for path in args.files:
        if not os.path.isfile(path):
            print(f"Warning: {path} is not a file, skipping", file=sys.stderr)
            continue
        redact_file(path, in_place=args.in_place, output=args.output)

if __name__ == "__main__":
    main()
