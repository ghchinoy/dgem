#!/usr/bin/env python3
"""Check that relative links in docs/ and README.md point at files that exist (anchors not checked)."""
import glob, os, re, sys

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
LINK = re.compile(r"\]\((?!https?://|/|#|mailto:)([^)\s#]+)(?:#[^)\s]*)?\)")
bad = 0
files = glob.glob(os.path.join(REPO, "docs", "**", "*.md"), recursive=True) + [os.path.join(REPO, "README.md")]
for f in sorted(files):
    for i, line in enumerate(open(f, encoding="utf-8"), 1):
        for tgt in LINK.findall(line):
            if not os.path.exists(os.path.normpath(os.path.join(os.path.dirname(f), tgt))):
                print(f"{os.path.relpath(f, REPO)}:{i}: broken link -> {tgt}")
                bad += 1
print(f"{'✅ PASSED' if not bad else '❌ FAILED'}: {bad} broken relative links in {len(files)} files")
sys.exit(1 if bad else 0)
