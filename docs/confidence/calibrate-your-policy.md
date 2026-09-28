---
title: "Calibrate Your Policy"
description: "Check that a dgem policy's confidence matches its accuracy on your own labelled data, choose a hesitation threshold for escalation, and decide when to add samples, a cascade or temperature scaling."
---

# Calibrate Your Policy

A confidence of 95% is only useful if answers given at 95% are right about 95% of the time *on your data*. This
page shows how to check that for your own policy and pick an escalation threshold. You need a policy
([Your first decision policy](../policies/first-policy.md)) and some labelled examples.

## 1. Label a sample

Collect real inputs with the answer you consider correct, one JSON object per line. Columns named
`expected_<question id>` are labels; the rest are template variables:

```json
{"id": "r01", "request": "Ordered a blue kettle, the box had a red toaster in it.", "expected_refund_eligible": "yes", "expected_reason": "wrong_item"}
{"id": "r04", "request": "Package never showed up, tracking stuck for 10 days.", "expected_refund_eligible": "no", "expected_reason": "late"}
```

Aim for **100+ labelled answers per question**, including the hard cases you actually see; a sample of easy cases
will look perfectly calibrated and tell you nothing.

## 2. Run the policy on it

Use the dataset runner from [Run a dataset](../policies/datasets.md#step-4-run-large-datasets-10010000-rows-from-python)
(set `TEMPLATE_PATH`, `DATASET_PATH` and your gateway URL). It writes `results_receipt.jsonl` with each
question's answer, confidence and hesitation. Keep `samples` at what you will use in production.

## 3. Read the report

```bash
python3 scripts/policy_calibration.py results_receipt.jsonl
```

Example from a 12-row refund dataset (illustrative size; recorded on a Vertex G4 endpoint):

```text
== refund_eligible: n=12  accuracy=91.7%  Brier(top)=0.0769  ECE(10 bins)=0.0937
  confidence bin   n   mean conf  accuracy
       0.7-0.8    1      0.793     1.000
       0.9-1.0   11      0.993     0.909
  keep if hesitation <   kept   accuracy of kept   escalated
                    5%   75.0%             100.0%           3
                   16%   83.3%             100.0%           2
                   35%   91.7%              90.9%           1
                  any  100.0%              91.7%           0
```

- **Accuracy**: share of answers that match your labels.
- **Brier (top answer)** and **ECE**: how far confidence is from accuracy (0 is perfect). The reliability table
  shows it bin by bin: answers given at ~99% were right 91% of the time here.
- **Hesitation thresholds**: what happens if every answer above the cut-off goes to a person or a larger model.
  At 16% the policy handles 83% of answers on its own, all correct in this sample, and escalates 2 of 12.

The one error is instructive: "package never showed up" was answered *eligible* at 94% confidence, yet its
hesitation was 34%. On a yes/no question 94% vs 6% is still a noticeable split, which is why hesitation (entropy
scaled to the number of options) is a better gate than raw confidence. The label itself is also debatable: the
policy text only mentions damaged or wrong items. Errors on a small sample are often policy wording, not the model.

## 4. Choose what to do with hesitant answers

| Situation | Action |
| :--- | :--- |
| Few hesitant answers, errors are costly | Escalate above ~16% hesitation to a person |
| Many hesitant answers, a larger model is acceptable | Stage 2 cascade: `"cascade_mode": "entropy"` sends only hesitant items to Gemini ([authoring guide](../policies/authoring.md)) |
| Hesitation clusters on one question | Rewrite that question or its option descriptions; add missing options ([taxonomy discovery](../policies/taxonomy-discovery.md)) |
| You need an error bar per decision | `samples: 4` adds standard error and agreement (~+35–40 ms) |
| Confidence is systematically too high or too low | Fit a temperature on held-out data (below) |

What we measured on public benchmarks: an offline cascade escalating answers above 16% hesitation to Gemini
reached 221/231 on JevBench with 39% escalated (Gemini on everything: 225) and 48/50 on the calibration suite
with 34% escalated ([EXP-14](../experiments/exp-14-idc-rerun.md)). Your escalation rate will depend on your data.

## 5. Temperature scaling (optional)

A temperature $T$ rescales probabilities after the fact ($T > 1$ softens overconfident answers). Fit it on data you
did not use to choose it, otherwise the improvement is illusory. For the built-in suites:

```bash
./bin/dgem bench-calibration --from-receipt <receipt.json> --auto-temperature
./bin/dgem bench-jev --from-receipt <receipt.json> --auto-temperature
```

On JevBench a held-out fit ($T^* \approx 1.5$) cut ECE by 24–33%; on the 50-item suite it gave no reliable gain
([EXP-14](../experiments/exp-14-idc-rerun.md)). Serve a fitted temperature through `dgem serve
--systemone-temperature` for `/v1/systemone`, or apply it in your client.

## 6. Re-check when things change

Re-run the report after changing the policy wording, the options, `samples`, or the serving image. A new vLLM
version can change a few percent of individual answers even when overall accuracy is unchanged
([image parity run](../../benchmarks/runs/20260927-image-parity/README.md)).
