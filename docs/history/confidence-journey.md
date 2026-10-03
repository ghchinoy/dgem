# How dgem's confidence story evolved: from entropy to hesitation-gating

> [!NOTE]
> **Historical narrative; not a guide.** This page explains how the current approach was reached, in order. For what
> `dgem` does today and the evidence behind it, read
> [Confidence beyond Shannon: hesitation-gated decisions](../confidence/overview.md). Every number below is quoted
> from the linked experiment, with its sample size.

## 1. Entropy as a signal (EXP-04, EXP-05)

The first question was whether one forward pass could say anything useful about its own reliability. On a 50-item
suite drawn from 11 public datasets, `dgem` answered 44/50 correctly, and per-question Shannon entropy rose on the
ChaosNLI items where human annotators disagreed (only 3 items per tier, so this is a direction, not a constant)
([EXP-04](../experiments/README.md)).

Scaling entropy by the number of options gave a single threshold that works for 2-option and 26-option questions alike.
Escalating items above a normalized threshold of 0.16 to `gemini-3.8-flash` reached 49/50 with 34% escalated
([EXP-05](../experiments/README.md)). The threshold was picked on the same 50 items, so this was promising but not yet
proof.

## 2. The first-option habit (EXP-13)

Entropy has a blind spot. On blank, content-free questions the model put 88.3% / 78.3% / 49.3% of its probability on
the first option for 2 / 3 / 4 options ([EXP-13](../experiments/exp-13-permutation-invariance.md)). On a borderline
input, that habit can make a toss-up look certain, and an entropy gate lets it through.

That led to a set of per-request corrections, named **Invariant Decision Calibration (IDC)**:

- **Null-prior de-biasing:** divide out the habit measured on a blank input.
- **Dual-mirror canvas:** add a reversed copy of each question to the same pass and compare the two readings.
- **Temperature scaling:** soften over-sharp probabilities, fitted on labelled data.

The write-up from that time is kept in [`idc-confidence-beyond-shannon.md`](idc-confidence-beyond-shannon.md).

## 3. Testing the corrections properly (EXP-14 to EXP-17)

The early IDC numbers mixed runs from different days and serving revisions. Re-running everything in one session
changed the picture:

- **EXP-14 (same session, 50 + 231 items):** identical runs moved by 1–2 items, which set the noise floor.
  Null-prior helped the 50-item suite (Brier 0.147 against 0.175–0.193 for three baselines) but not JevBench (186 vs
  187 correct, worse calibration). The same-pass mirror lowered forward accuracy on JevBench
  ([EXP-14](../experiments/exp-14-idc-rerun.md)).
- **EXP-15:** the mirror's damage came from **letter collision**. Both questions used A, B, C for different options,
  and the model copied letters across. A lettered reversed slot scored 154 against a baseline band of 182–189; a
  digit-labelled one stayed within the band ([EXP-15](../experiments/exp-15-letter-collision.md)).
- **EXP-16:** question ids are part of the prompt. A second slot named `…__mirror_rev` cost 19 items against `__rev`
  with identical options ([EXP-16](../experiments/exp-16-slot-names.md)).
- **EXP-17:** a reversed read in its own pass added a statistically real but small signal: at most +0.016 AUROC over
  hesitation alone (about 0.85), for twice the cost ([EXP-17](../experiments/exp-17-separate-pass-mirror.md)).

The order-bias problem was real and reproducible, but none of the corrections improved decisions reliably. Two
lessons outlived the corrections: **the answer template is part of the prompt**, and **compare only against noise
measured in the same session**.

## 4. Hesitation-gating, tested live (EXP-18)

Gating on hesitation alone was then tested with the threshold fixed in advance. Across 13 human-labelled judge suites
(n=100 each), a 35% gate to `gemini-3.8-flash` reached 81/100 on safety and 86/100 on faithfulness while escalating
8–32% of items, at a median 105 ms against 3,125 ms
([EXP-18](../experiments/exp-18-mizan-judge-capability.md)). The live result matched the offline simulation.

## 5. Making the probabilities more trustworthy (EXP-19 to EXP-21, v0.2.0)

With the gate settled, the work moved to what feeds it:

- **Prompt layout (EXP-19):** putting the input before the questions (`document_first`) raised accuracy on
  multi-question and wide-option sets with no loss elsewhere, and became the default in serving v0.2.0. At the release
  gate it moved frozen held-out sets such as XNLI from 0.662 to 0.699 (n=4,500), at the cost of lower out-of-scope
  recall ([prompt layout](../policies/prompt-layout.md)).
- **Joint reads (EXP-19):** asking each question in its own pass cost 5× the latency and gained only +1.6 points (not
  significant) under the new layout, so joint reads stayed.
- **Context length (EXP-20)** and **two Decision Index drops (EXP-21)** were checked so that changes to serving were
  not mistaken for changes in the model ([ledger](../experiments/README.md)).
- **Regression matrix:** every serving change is judged against run-to-run agreement measured in the same session,
  with option order shuffled to report extra flips and a temperature fitted per domain
  ([regression matrix](../operate/regression-matrix.md)).

## 6. Where that leaves the name

"IDC" described corrections that did not survive testing. What `dgem` does today is simpler: a probability for every
option, a hesitation score, a hand-off when hesitation is high, templates and layout that do not distort the
probabilities, order bias measured on every release, and calibration fitted per domain. That is **hesitation-gating**,
described in [Confidence beyond Shannon](../confidence/overview.md).
