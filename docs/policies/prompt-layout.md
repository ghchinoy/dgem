---
title: Prompt Layout (document first or questions first)
description: How dgem lays out the prompt for a decision, what changed in v0.2.0, the measured effect of each layout, when to choose the other one, and how to compare them on your own data.
---

Every decision sends the model two things: your **input** (the ticket, document, log or diff) and your **questions**
(each question with its allowed answers). The prompt layout decides the order and where each part goes.

| Layout | What the model sees | Default |
| :--- | :--- | :--- |
| `document_first` | One user message: your input inside `<user_text>`, then the questions inside `<instructions>` | **Yes, from serving v0.2.0** |
| `schema_first` | The questions as the system message, your input as the user message | Before v0.2.0 |

The questions, options, answer format and readout are identical in both layouts. Only the order and the message roles
change, so the model reads your input before or after it knows what it will be asked.

## What each layout looks like

For a policy with one question, `intent`, and the input `My card was charged twice for the same coffee.`:

**`document_first`** (one user message):

```text
<user_text>
My card was charged twice for the same coffee.
</user_text>

<instructions>
Answer a fixed set of questions about the state the user provides. Each question lists its allowed answers;
reply with exactly one label per question.

Question intent: What does the customer want?
  A: refund (money back for a charge)
  B: new_card (a replacement card)
  C: balance (account balance)

Reply with one line per question, in this order, formatted as "id: label".
</instructions>
```

**`schema_first`** (system message, then user message):

```text
[system]  Answer a fixed set of questions ... Question intent: What does the customer want? ... "id: label".
[user]    My card was charged twice for the same coffee.
```

## Why document first is the default

We measured both layouts on development data before changing the default, and again at the release gate. Accuracy is
the share of answers matching the gold label; differences are paired, item by item
([EXP-19](../experiments/README.md), release-gate run in `benchmarks/runs/20261002-v020-release-t2/`).

| Data | Items | `schema_first` | `document_first` |
| :--- | ---: | ---: | ---: |
| typed-decisions test, 5 questions per decision (frozen set, release gate) | 2,000 | 0.674 | **0.728** |
| typed-decisions train, joint read of 5 questions (dev) | 200 × 5 | 0.534 | **0.604** |
| XNLI test, 15 languages (frozen set, release gate) | 4,500 | 0.662 | **0.699** |
| CLINC150 validation, 151 options with an "out of scope" option (development split, release gate) | 100 × 3 | 0.740 | **0.863** |
| RAGTruth train, "does the response contain unsupported content?" (development split, release gate) | 198 × 3 | 0.712 | **0.771** |
| MASSIVE test, 20 intents, 51 languages (frozen set) | 5,100 | 0.820 | 0.824 |
| JevBench, single questions | 231 × 3 | 0.840 | 0.848 (within noise) |

The gain is largest when a decision has **several questions** or **many options**. With the questions first, the
model tends to settle on an answer shape before it has read the input. Single, simple questions barely change.

On the [Decision Index](../experiments/exp-12-decision-index.md) (0.2.1, scored with the index's own kit and shown on
our internal board; dgem is not on a public leaderboard), the release with this default, together with yes/no
descriptions now reaching the model, moved dgem from 40.77 to 45.34. In the EXP-19 development session the same layout
change moved CLINC150 from 0.790 to 0.870 and RAGTruth from 0.705 to 0.770 (`benchmarks/runs/20261002-w2-canary-t1/`).

## Trade-offs to know about

- **"None of these" options are chosen less often.** On CLINC150, in-scope requests answered "out of scope" fell from
  21% to 2.5%, but genuinely out-of-scope requests were recognised 82.5% of the time instead of 100% (40 items). If
  catching out-of-scope inputs matters more than classifying in-scope ones, compare both layouts on your data.
- **A few more prompt tokens.** The tags add a handful of tokens. A prompt right at the server's context limit can tip
  over and be refused: one of 231 JevBench items did at 4,096 tokens. The deploy scripts now default to 8,192 tokens on
  RTX PRO 6000 (EXP-20), and the regression matrix reports refusals by reason.
- **Two Decision Index drops are not explained by the layout.** ForecastBench (probability forecasts) and
  PhishNChips (phishing decisions) went down in the same release. On development stand-ins (EXP-21: 320 single
  dataset-source and 182 market forecasting questions from non-index dates, 200 phishing emails) neither the layout
  nor the yes/no descriptions changed the result beyond noise. ForecastBench's combination questions are still
  untested.
- **Calibration** improved on most sets (typed-decisions ECE 0.226 → 0.176, XNLI 0.263 → 0.229) and got slightly worse
  on MASSIVE (0.082 → 0.104). Fitting a temperature per policy on your own data matters more than the layout here; see
  [calibrate your policy](../confidence/calibrate-your-policy.md).

## When to choose `schema_first`

Keep the default unless one of these applies, and then measure before switching:

1. **You validated a policy before v0.2.0** and need byte-identical prompts (for example, an audited workflow). Pin
   `schema_first` in the template.
2. **Out-of-scope recall is the priority** for a policy with a catch-all option (see the trade-off above).
3. **Your prompts sit at the context limit** and a few extra tokens cause refusals (or ask your operator to raise
   the context length).

## How to set it

The narrowest setting wins: request, then template, then deployment.

| Where | How |
| :--- | :--- |
| A template (policy) | Add `"layout": "schema_first"` next to `"questions"` in the schema |
| One request to the server | `"layout": "schema_first"` in the schema (chat) or the `/v1/systemone` body |
| `dgem decide` | `--layout schema_first` |
| Gateway `POST /api/decide` | JSON `"layout": "schema_first"`, header `X-DGem-Layout`, or `?layout=` |
| MCP `decide_policy`, `decide_custom_questions` | `"layout": "schema_first"` |
| Web Studio | **Prompt layout** menu in the evaluation panel (also used by the batch runner) |
| Decision Index adapter | `dgem systemone serve --prompt-layout schema_first` |
| A whole deployment | Container env `DEFAULT_LAYOUT=schema_first` |

Leaving the field empty uses the template's own `layout`, or else the server's default. Responses report the layout the
server used in `diagnostics.layout`; the Studio shows it next to the menu. Servers older than v0.2.0 ignore the field
and always use `schema_first`.

## Yes/no questions get their own read

From serving v0.3.0, in a request with 2 to 8 questions, each yes/no question is read on its own, in parallel with the
rest of the request and with a prompt that lists only that question. The other questions still share one read.

**Why:** read together, a yes/no question that follows another one tended to copy its answer. On a development suite
of 120 rule-labelled cases ([PROP-19](../experiments/proposed.md#prop-19-cross-slot-coupling-on-mixed-polarity-yesno-questions)),
an action question ("should the assistant ask before booking?") paired with a fact question scored 0.61 in one joint
read and 0.89 with its own read; in agent and tool-call settings, 0.34 against 0.98. On multi-question decisions
(typed-decisions train, 200 × 5) accuracy did not drop (+1.5 points), latency rose by about 60 ms, and throughput at 16
concurrent requests did not fall.

**Controls:** `"isolate": "none"` on a request restores one joint read (`"all"` isolates every question);
`"alone": false` on a question keeps it in the joint read; `DEFAULT_ISOLATE` sets the deployment default. Responses list
the isolated questions in `diagnostics.isolated`.

## How to compare both on your own data

Run the same labelled rows twice and compare accuracy, agreement and hesitation:

- **Studio:** load the dataset in the batch runner, run once per **Prompt layout** setting, and compare the results.
- **Python or CI:** add `"layout": "document_first"` or `"schema_first"` to the `POST /api/decide` body in the
  [dataset runner](datasets.md) and run it twice.
- **Operators:** the [regression matrix](../operate/regression-matrix.md) compares layouts on one deployment with
  `--target base=<URL> --target old=<URL>#layout=schema_first --baseline base`.

Use at least a few hundred labelled items, compare item by item, and decide before you look at the results what
difference would make you switch.
