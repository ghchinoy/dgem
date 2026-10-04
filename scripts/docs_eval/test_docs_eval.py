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

"""Offline tests for EXP-25 (no network): python3 -m unittest scripts.docs_eval.test_docs_eval"""
import unittest

from scripts.docs_eval import report, rubric, sections


class Rubric(unittest.TestCase):
    def test_policies_render_and_convert(self):
        for p in rubric.POLICIES:
            body = rubric.systemone_body(p, 'a "quoted" section\nwith lines')
            self.assertEqual(body["state"]["section"], 'a "quoted" section\nwith lines')
            for qid, q in body["questions"].items():
                self.assertIn(q["type"], ("choice", "score"))
                self.assertLessEqual(len(q["criteria"]), 26)

    def test_style_questions_have_clean_then_problem(self):
        qs = {q[0]: q for q in rubric.questions("docs_style_audit")}
        for qid in rubric.PROBLEM_QUESTIONS:
            self.assertEqual(len(qs[qid][3]), 2)

    def test_shuffled_order_keeps_descriptions(self):
        body = rubric.systemone_body("docs_diataxis", "x", order={"doc_type": ["navigation", "tutorial", "how_to",
                                                                             "reference", "explanation",
                                                                             "results_report"]})
        crit = body["questions"]["doc_type"]["criteria"]
        self.assertEqual(list(crit)[0], "navigation")
        self.assertIn("links", crit["navigation"])

    def test_gemini_schema_enums(self):
        _prompt, schema = rubric.gemini_request("docs_claim_evidence", "x")
        self.assertEqual(len(schema["properties"]["evidence"]["enum"]), 4)


class Sections(unittest.TestCase):
    MD = "---\ntitle: T\n---\n\nIntro " + "word " * 50 + "\n\n## A\n\n```bash\n## not a heading\n" + "x\n" * 10 + \
         "```\n\n| a | b |\n" + "| 1 | 2 |\n" * 10 + "\n## B\n\ntext\n"

    def test_split_respects_code_fences_and_shortens(self):
        secs = list(sections.split_page("docs/t.md", self.MD))
        self.assertEqual([s["heading"] for s in secs], ["(intro)", "A", "B"])
        self.assertIn("more lines", secs[1]["text"])
        self.assertIn("more rows", secs[1]["text"])
        self.assertTrue(secs[0]["text"].startswith("Page: T"))


class Metrics(unittest.TestCase):
    def test_auroc_and_f1(self):
        self.assertEqual(report.auroc([0.9, 0.8, 0.1], [True, True, False]), 1.0)
        self.assertEqual(report.f1(["p", "p", "c"], ["p", "c", "c"], "p"), 2 / 3)

    def test_thresholds_apply_to_style_only(self):
        preds = {"s": {"answers": {"openers": {"answer": "announces_points",
                                               "probs": {"states_points": 0.4, "announces_points": 0.6}},
                                   "doc_type": {"answer": "how_to", "probs": None}}}}
        out = report.apply_thresholds(preds, {"openers": 0.9})
        self.assertEqual(out["s"]["answers"]["openers"]["answer"], "states_points")
        self.assertEqual(out["s"]["answers"]["doc_type"]["answer"], "how_to")


if __name__ == "__main__":
    unittest.main()
