# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Split Markdown pages into H2 sections and render them for review (standard library only).

A section is the text under one `## ` heading (the text before the first H2 is the page's intro section). Code
blocks keep their first CODE_LINES lines, tables their first TABLE_ROWS rows, so a reviewer (human or model) still sees
that they are there. Frontmatter is dropped; the page title is prepended as context.
"""
import hashlib
import re

CODE_LINES = 4
TABLE_ROWS = 6
MAX_CHARS = 3500
MIN_PROSE_WORDS = 40


def _title(md):
    m = re.search(r"^title:\s*\"?(.+?)\"?\s*$", md, re.M)
    if m:
        return m.group(1)
    m = re.search(r"^# (.+)$", md, re.M)
    return m.group(1).strip() if m else ""


def _strip_frontmatter(md):
    if md.startswith("---\n"):
        end = md.find("\n---", 4)
        if end != -1:
            return md[end + 4:].lstrip("\n")
    return md


def shorten(body):
    """Shorten code blocks and tables; return (text, prose_words)."""
    out, prose, lines, i = [], [], body.split("\n"), 0
    while i < len(lines):
        ln = lines[i]
        if ln.lstrip().startswith("```"):
            j = i + 1
            while j < len(lines) and not lines[j].lstrip().startswith("```"):
                j += 1
            code = lines[i + 1:j]
            out.append(ln)
            out.extend(code[:CODE_LINES])
            if len(code) > CODE_LINES:
                out.append(f"... ({len(code) - CODE_LINES} more lines)")
            out.append("```")
            i = j + 1
            continue
        if ln.lstrip().startswith("|"):
            j = i
            while j < len(lines) and lines[j].lstrip().startswith("|"):
                j += 1
            rows = lines[i:j]
            out.extend(rows[:TABLE_ROWS])
            if len(rows) > TABLE_ROWS:
                out.append(f"| ... ({len(rows) - TABLE_ROWS} more rows) |")
            i = j
            continue
        out.append(ln)
        prose.append(ln)
        i += 1
    words = len(re.findall(r"[A-Za-z]{2,}", " ".join(l for l in prose if not l.startswith("#"))))
    return "\n".join(out).strip(), words


def truncate(text, limit=MAX_CHARS):
    if len(text) <= limit:
        return text, False
    cut = text.rfind("\n\n", 0, limit)
    cut = cut if cut > limit // 2 else limit
    return text[:cut].rstrip() + "\n\n[... section truncated]", True


def split_page(path, md):
    """Yield dicts: id, page, title, heading, index, text, prose_words, truncated."""
    title = _title(md)
    body = _strip_frontmatter(md)
    parts = re.split(r"^(?=## )", body, flags=re.M)
    in_code = False
    # re.split ignores fences; merge parts whose split point fell inside a code block
    merged = []
    for p in parts:
        if merged and in_code:
            merged[-1] += p
        else:
            merged.append(p)
        in_code = (merged[-1].count("```") % 2) == 1
    # long H2 sections are split again at H3 headings, so fewer sections are truncated
    pieces = []
    for part in merged:
        if len(part) <= MAX_CHARS or "\n### " not in part:
            pieces.append(part)
            continue
        sub, code = [], False
        for chunk in re.split(r"^(?=### )", part, flags=re.M):
            if sub and not code:
                sub.append(chunk)
            elif sub:
                sub[-1] += chunk
            else:
                sub.append(chunk)
            code = (sub[-1].count("```") % 2) == 1
        h2 = part.split("\n", 1)[0][3:].strip() if part.startswith("## ") else "(intro)"
        pieces.append(sub[0])
        pieces += [f"## {h2} / {c.split(chr(10), 1)[0][4:].strip()}\n" + (c.split("\n", 1)[1] if "\n" in c else "")
                   for c in sub[1:]]
    for idx, part in enumerate(pieces):
        heading = part.split("\n", 1)[0][3:].strip() if part.startswith("## ") else "(intro)"
        text, words = shorten(part)
        text, cut = truncate(f"Page: {title}\n\n{text}")
        sid = hashlib.sha1(f"{path}#{idx}#{heading}".encode()).hexdigest()[:10]
        yield {"id": sid, "page": path, "title": title, "heading": heading, "index": idx, "text": text,
               "prose_words": words, "truncated": cut}


def section_key(s):
    """Stable key for matching a section across versions of a page (git pairs)."""
    return f"{s['page']}#{re.sub(r'[^a-z0-9]+', ' ', s['heading'].lower()).strip()}"
