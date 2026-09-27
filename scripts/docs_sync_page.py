#!/usr/bin/env python3
"""Copy a docs/ page into docs-site/, keeping the site's front matter.

Transformations: the leading H1 is dropped (Starlight renders the front-matter
title) and relative links like (foo.md#anchor) become (/dgem/foo/#anchor).

Usage: scripts/docs_sync_page.py setup.md [user-guide.md ...]
"""

import os
import re
import sys

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
DOCS = os.path.join(REPO, "docs")
SITE = os.path.join(REPO, "docs-site", "src", "content", "docs")

LINK = re.compile(r"\]\((?!https?://|/|#|mailto:)([^)\s]+?)\.md(#[^)]*)?\)")


def front_matter(text):
    if text.startswith("---\n"):
        end = text.find("\n---", 4)
        if end != -1:
            return text[: end + 4].rstrip() + "\n", text[end + 4 :]
    return "", text


def site_link(m):
    target, anchor = m.group(1), m.group(2) or ""
    target = re.sub(r"^\./", "", target)
    if target.startswith("../"):
        return m.group(0)  # points outside docs/; leave as-is
    if target.endswith("README"):
        target = target[: -len("README")].rstrip("/")
    return f"](/dgem/{target}/{anchor})".replace("//", "/").replace("](/", "](/", 1)


def sync(rel):
    src = open(os.path.join(DOCS, rel), encoding="utf-8").read()
    dst_path = os.path.join(SITE, rel)
    fm, _ = front_matter(open(dst_path, encoding="utf-8").read())
    _, body = front_matter(src)
    lines = body.lstrip("\n").splitlines()
    if lines and lines[0].startswith("# "):
        lines = lines[1:]
    body = "\n".join(lines).lstrip("\n")
    body = LINK.sub(site_link, body)
    with open(dst_path, "w", encoding="utf-8") as f:
        f.write(fm + "\n" + body.rstrip() + "\n")
    print(f"synced docs/{rel} -> docs-site/.../{rel}")


SITE_LINK = re.compile(r"\]\(/dgem/([^)#]*?)/?(#[^)]*)?\)")


def sync_reverse(rel, site_rel=None):
    """docs-site -> docs: keep docs/ H1 title, convert /dgem/x/ links to x.md (relative)."""
    site = open(os.path.join(SITE, site_rel or rel), encoding="utf-8").read()
    _, body = front_matter(site)
    dst_path = os.path.join(DOCS, rel)
    old = open(dst_path, encoding="utf-8").read()
    fm, old_body = front_matter(old)
    title = next((l for l in old_body.splitlines() if l.startswith("# ")), "")
    depth = rel.count("/")
    prefix = "../" * depth

    def repl(m):
        target, anchor = m.group(1), m.group(2) or ""
        if target.endswith(".html") or target.startswith("assets"):
            return f"](https://ghchinoy.github.io/dgem/{target}{anchor})"
        if target in ("", "experiments"):
            target = (target + "/README").lstrip("/") if target else "index"
        return f"]({prefix}{target}.md{anchor})"

    body = SITE_LINK.sub(repl, body.lstrip("\n"))
    out = (fm + "\n" if fm else "") + (title + "\n\n" if title else "") + body.rstrip() + "\n"
    with open(dst_path, "w", encoding="utf-8") as f:
        f.write(out)
    print(f"synced docs-site/.../{site_rel or rel} -> docs/{rel}")


if __name__ == "__main__":
    args = sys.argv[1:]
    if args and args[0] == "--reverse":
        for rel in args[1:]:
            src, _, site_rel = rel.partition("=")
            sync_reverse(src, site_rel or None)
    else:
        for rel in args:
            sync(rel)
