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

"""The three EXP-25 policies (templates/docs/*.json.tmpl) as one rubric.

The templates are the source of truth and also run through `dgem decide -t templates/docs/<policy>.json.tmpl -v text=...`.
This module renders them (they use only `.text` and `.samples`), converts the questions to the /v1/systemone
format (criteria), and builds the equivalent Gemini prompt and response schema, so every engine sees the same
instructions and option descriptions.
"""
import hashlib
import json
import os

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
POLICIES = ("docs_diataxis", "docs_style_audit", "docs_claim_evidence")
# Style questions that a human labels as a ticked "problem"; the second option of each is the problem.
PROBLEM_QUESTIONS = ("openers", "framing", "actors", "sentences", "reader", "tone")


def template_path(policy):
    return os.path.join(REPO, "templates", "docs", f"{policy}.json.tmpl")


def render(policy, text="", samples=1):
    raw = open(template_path(policy)).read()
    raw = raw.replace("{{ default 1 .samples | toJson }}", json.dumps(samples))
    raw = raw.replace('{{ default "" .text | toJson }}', json.dumps(text, ensure_ascii=False))
    if "{{" in raw:
        raise ValueError(f"{policy}: unsupported template construct")
    return json.loads(raw)


def questions(policy):
    """[(qid, type, instructions, labels, descriptions or None)] in template order."""
    out = []
    for q in render(policy)["schema"]["questions"]:
        if q["type"] == "choice":
            out.append((q["id"], "choice", q["instructions"], [o["name"] for o in q["options"]],
                        [o["description"] for o in q["options"]]))
        elif q["type"] == "score":
            out.append((q["id"], "score", q["instructions"], list(q["levels"]), None))
        else:
            raise ValueError(f"{policy}/{q['id']}: unsupported type {q['type']}")
    return out


def all_questions():
    return [(p, *q) for p in POLICIES for q in questions(p)]


def rubric_hash():
    h = hashlib.sha256()
    for p in POLICIES:
        h.update(open(template_path(p), "rb").read())
    return h.hexdigest()[:12]


def systemone_body(policy, text, qids=None, order=None, samples=1):
    """order: {qid: [labels in display order]} for the shuffled-order runs."""
    schema = render(policy, text, samples)
    qs = {}
    for qid, typ, instr, labels, descs in questions(policy):
        if qids and qid not in qids:
            continue
        if typ == "choice":
            pairs = list(zip(labels, descs))
            if order and qid in order:
                d = dict(pairs)
                pairs = [(l, d[l]) for l in order[qid]]
            qs[qid] = {"type": "choice", "instructions": instr, "criteria": dict(pairs)}
        else:
            qs[qid] = {"type": "score", "instructions": instr, "criteria": labels}
    instr = schema["schema"]["instructions"]
    return {"state": {"instructions": instr, **schema["state"]}, "questions": qs, "samples": samples}


def gemini_request(policy, text):
    """Prompt and responseSchema asking the same questions with the same option descriptions."""
    schema = render(policy, text)
    lines = [schema["schema"]["instructions"], "", "Answer every question with exactly one of its option names.", ""]
    props = {}
    for qid, typ, instr, labels, descs in questions(policy):
        lines.append(f"Question `{qid}`: {instr}")
        if descs:
            lines += [f"  - {l}: {d}" for l, d in zip(labels, descs)]
        else:
            lines.append("  Options: " + ", ".join(labels))
        props[qid] = {"type": "STRING", "enum": labels}
    lines += ["", "Section:", "<<<", text, ">>>"]
    return "\n".join(lines), {"type": "OBJECT", "properties": props, "required": list(props)}
