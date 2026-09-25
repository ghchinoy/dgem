---
title: "EXP-13: Permutation Sensitivity & O(1) Dual-Mirror Canvas"
description: "Measuring content-free positional 'A'-bias, Cyclic Jensen-Shannon Divergence (I(Y; Π | X)), and O(1) single-pass Dual-Mirror Canvas calibration on Cloud Run GPU."
---

# EXP-13: Permutation Sensitivity, Content-Free Null-Prior De-Biasing & O(1) Dual-Mirror Canvas Calibration

## 1. Motivation & Problem Statement

In constrained decision models (`DiffusionGemma` / `dgem`), single-pass epistemic confidence is typically derived from the restricted softmax probability $p(y = \ell_k \mid X, \pi)$ and Shannon entropy $H(Y \mid X, \pi) = -\sum_{k=1}^K p_k \ln p_k$ over option letters $\ell_k \in \{\text{A}, \text{B}, \dots, \text{Z}\}$ under a fixed presentation order $\pi$.

However, **Shannon entropy under a single option ordering $\pi_0$ measures token-slot certainty, not semantic invariance**:
$$z(\ell_k \mid X, \pi) = s(o_{\pi(k)} \mid X) + b_{\text{pos}}(k) + \epsilon_{\text{fmt}}(X, \pi, k)$$
where:
- $s(o_{\pi(k)} \mid X)$ is the true semantic preference for option $o_{\pi(k)}$,
- $b_{\text{pos}}(k)$ is the structural positional/token bias for slot index $k$ (e.g., primacy bias toward `'A'`), and
- $\epsilon_{\text{fmt}}(X, \pi, k)$ is formatting-order interaction noise.

**The Epistemic Blindspot**: When a task is genuinely ambiguous ($s(o_1 \mid X) \approx s(o_2 \mid X)$), a strong positional prior $b_{\text{pos}}(0) \gg b_{\text{pos}}(1)$ artificially inflates the probability of whichever option happens to sit in Slot `'A'`, collapsing single-pass Shannon entropy $H(Y \mid X, \pi_0) \to 0$ and causing a **False Early Exit** in entropy-gated cascades (`EXP-05`).

---

## 2. Falsifiable Hypotheses (`H1`–`H4`)

1. **`H1` (Content-Free Positional `'A'`-Bias)**: When `dgemma` evaluates a content-free state (`[REDACTED / CONTENT-FREE CALIBRATION PROBE]`) with neutral option strings, $p_0(k) \neq 1/K$ exhibits a strong structural primacy bias toward Slot `'A'`. Subtracting $\log p_0(k)$ in logit space (`Null-Prior De-Biasing`) improves calibration without extra inference passes.
2. **`H2` (Single-Pass Entropy Blindspot vs. Permutation Mutual Information)**: On human-disagreement tasks (`ChaosNLI`), single-pass normalized Shannon entropy $\tilde{H}(Y \mid X, \pi_0)$ can fall below early-exit thresholds (`< 0.16`) when a plausible option aligns with Slot `'A'`, whereas **Permutation Jensen-Shannon Divergence (`Cyclic JSD`)** $I(Y; \Pi \mid X)$ spikes by $>100\times$ and reveals that the `argmax` flips under option reordering.
3. **`H3` ($O(1)$ Single-Pass Dual-Mirror Canvas)**: Because `DiffusionGemma` denoises all canvas slots simultaneously in **one bidirectional forward pass (`reads=1`)**, emitting two mirrored slots (`decision_fwd` $[o_1 \dots o_K]$ and `decision_rev` $[o_K \dots o_1]$) on the **same diffusion canvas** runs with **zero latency overhead** (`0 ms` penalty vs. 1-slot), cancels first/last positional bias ($b_{\text{pos}}(k) + b_{\text{pos}}(K-1-k)$), guarantees `0.0%` reversal flip rate, and provides a live **Single-Pass Mirror JSD / TVD** disagreement signal.
4. **`H4` (Joint Entropy + Mirror JSD Cascade Rule)**: Augmenting single-pass entropy gating with Dual-Mirror disagreement (`Escalate if` $\tilde{H}_{\text{mirror}} \ge \tau_H \lor \text{TVD}_{\text{mirror}} \ge \tau_{\text{TVD}}$) catches every permutation-unstable ambiguity that fools single-pass Shannon entropy.

---

## 3. Empirical Results on Live Cloud Run GPU (`dgemma`)

Executed via `./bin/dgem bench-permutation -u "https://dgemma-tkb3aiuiea-uc.a.run.app/v1" --gcp-auth -w 2` (`benchmarks/results_permutation_cloudrun.json`).

### 3.1 `EXP-13B`: Measured Content-Free Positional Null Priors ($p_0$)

When prompted with a content-free context (`[REDACTED / CONTENT-FREE CALIBRATION PROBE]`) and neutral option labels (`Option 1` $\dots$ `Option K`), `dgemma` exhibits a severe structural bias toward **Slot 0 (`'A'`)** (and secondary recency bias on `'D'` for $K=4$, starving middle slot `'B'` to `4.8%`):

| Cardinality ($K$) | Slot `'A'` ($p_0(0)$) | Slot `'B'` ($p_0(1)$) | Slot `'C'` ($p_0(2)$) | Slot `'D'` ($p_0(3)$) | Uniform Baseline ($1/K$) | Slot `'A'` Bias Ratio | Null Entropy ($H_0$ vs $\ln K$) |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **$K=2$ (`Binary`)** | **`88.3%`** | `11.7%` | — | — | `50.0%` | **`1.77x`** | `0.361` / `0.693 nats` |
| **$K=3$ (`3-Way NLI`)** | **`78.3%`** | `8.6%` | `13.1%` | — | `33.3%` | **`2.35x`** | `0.669` / `1.099 nats` |
| **$K=4$ (`4-Way MCQ`)** | **`49.3%`** | **`4.8%`** | `16.3%` | `29.5%` | `25.0%` | **`1.97x`** | `1.164` / `1.386 nats` |

Applying **Logit-Space Null-Prior De-Biasing** ($\tilde{p}_k \propto p_k / p_0(k)^\alpha$ with $\alpha=1.0$) reduced the **Multi-Class Brier Score by `-90.2%`** (`0.0173` $\rightarrow$ **`0.0017`**).

---

### 3.2 `EXP-13A` & `EXP-13C`: Method Comparison (`1-Slot` vs. `Null-Prior` vs. `Cyclic-K` vs. `O(1) Dual-Mirror Canvas`)

| Calibration / Readout Method | Forward Passes (`reads`) | Mean Latency (`ms`) | Overall Accuracy | Multi-Class Brier (`↓`) | Soft-Label TVD (`↓`) | Order Flip Rate (`↓`) | Permutation Signal Available? |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :--- |
| **`1. Baseline 1-Slot (Canonical)`** | `1` | `128.8 ms` | `100.0%` | `0.0173` | `0.1536` | `12.5%` cyclic (`2/16`; `2/4` on ambiguous) | ❌ No |
| **`2. 1-Slot + Null-Prior De-Biased (13B, α=0.75)`** | `1` | `128.8 ms` | `100.0%` | **`0.0017` (`-90.2%`)** | **`0.1355` (`-11.8%`)** | `25.0%` cyclic (**worse**) | ❌ Static prior correction only |
| **`3. K-Pass Cyclic Ensemble (13A Oracle)`** | `K` (`2–4`) | `369.5 ms` (`2.86x`) | `93.75%` | `0.0369` | `0.1570` | n/a (ensemble = one answer) | ✅ Full `Cyclic JSD` ($I(Y;\Pi\mid X)$) |
| **`4. O(1) Dual-Mirror Canvas (13C)`** | **`1` (`2 slots`)** | **`124.7 ms` (`0.97x`)** | `100.0%` | `0.0410` (**worse**) | `0.1370` (`-10.8%`) | n/a (merged = one answer) | ✅ Live `Mirror JSD` & `Mirror TVD` in 1 pass (reversal only) |

> **Key architectural result**: Because `DiffusionGemma` fills both `decision_fwd` ($[o_1 \dots o_K]$) and `decision_rev` ($[o_K \dots o_1]$) in the **same bidirectional canvas**, the Dual-Mirror readout costs no extra forward pass (`124.7 ms` vs. `128.8 ms`, n=16) and exposes a per-request `Mirror TVD` / `JSD`. Caveats: a merged output cannot "flip" by construction, so reversal flip rate is not a meaningful metric for it; its Brier score is worse than baseline on this suite; and it only tests one alternative order (see `perm_06` below). The baseline made **zero errors** on these 16 synthetic items, so this suite measures confidence honesty, not error catching.

---

### 3.3 Regime Breakdown: Where Single-Pass Entropy Is Blind and What Catches It (`H2`)

| Regime (`4 synthetic cases each`) | Mean 1-Pass Norm Entropy ($\tilde{H}$) | Mean `Cyclic JSD` ($I(Y;\Pi\mid X)$) | `Cyclic JSD` vs Consensus | Cyclic Permutation Flip Rate | Mean `Mirror TVD` (`13C`) |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **`consensus`** (Unambiguous facts) | `0.0065` | `0.0017 nats` | `1.0x` (baseline) | `0.0%` (`0/4`) | `0.0044` |
| **`adversarial_trap`** (Distractor in `'A'`) | `0.0176` | `0.0036 nats` | `2.1x` | `0.0%` (`0/4`) | `0.0673` |
| **`binary_ablation`** ($K=2$ policies) | `0.0330` | `0.0016 nats` | `0.9x` | `0.0%` (`0/4`) | `0.0431` |
| **`ambiguous_chaosnli`** (ChaosNLI-style split) | `0.2077` | **`0.2683 nats`** | **`~156x`** | **`50.0%`** (`2/4` flip `argmax`) | **`0.2626`** (driven by `perm_07` `0.785` and `perm_08` `0.258`; `perm_05`/`perm_06` ≈ `0.001–0.007`) |

*(All values from `benchmarks/results_permutation_cloudrun.json`, 2026-09-23. "ChaosNLI-style" items are authored for this suite with a stated human split; they are not drawn from the ChaosNLI release.)*

#### Concrete Cases (`perm_08`, `perm_06`, `perm_07`)
1. **`perm_08_ambiguous_fair_use_parody`** (*14-second chorus captured incidentally in a documentary; expected `incidental_fair_use`*): **Dual-Mirror catches false certainty.**
   - **Canonical 1-Slot** (`A=requires_sync_license, B=incidental_fair_use, C=public_domain_waiver`): `incidental_fair_use` at **`99.9%`**, $\tilde{H} = 0.007$, so it would early-exit under the `0.16` entropy gate.
   - **Dual-Mirror canvas (1 pass)**: forward slot `71.3%`, reversed slot `96.5%`, giving **`Mirror TVD = 0.258`** and merged $\tilde{H} = 0.465$, so it is **escalated**. (The forward reading itself dropped from 99.9% to 71.3% once the reversed slot shared the canvas: the two slots are not independent.)
2. **`perm_06_ambiguous_chaosnli_hospital`** (*surgeon sighs and summons the family; stated split 48% entailment / 46% neutral; expected `neutral`*): **Dual-Mirror misses a real order sensitivity.**
   - **Canonical 1-Slot** (`A=entailment, B=neutral, C=contradiction`): `neutral` at `99.5%` ($\tilde{H} = 0.031$). Correct, and *not* the Slot-`'A'` option.
   - **Cyclic shift** (`A=contradiction, B=entailment, C=neutral`): flips to `entailment` at `95.6%` (`Cyclic JSD = 0.573 nats`).
   - **Dual-Mirror canvas**: forward and reversed readings are both ≈ `99.9%` `neutral` (`Mirror TVD = 0.0006`), so it is **not flagged**. Reversal is only one of the orderings that matter.
3. **`perm_07_ambiguous_dual_intent_vip`** (*ticket that is equally an SRE outage and a legal termination*): caught by **both** gates. Canonical $\tilde{H} = 0.71$; the forward and reversed slots pick different answers (`Mirror TVD = 0.785`).

### 3.4 Verdict on Hypotheses

| Hypothesis | Verdict | Evidence |
| :--- | :--- | :--- |
| **`H1`** Content-free Slot-`'A'` bias exists; subtracting it improves calibration | ✅ **Supported** (bias); ⚠️ **Mixed** (effect) | Bias is large (`88% / 78% / 49%`). Brier improves `-90.2%` on already-correct answers, but cyclic flip rate rises `12.5% → 25%`. On the 50-item calibration suite it is the best IDC component (see [IDC §6](../confidence-beyond-shannon.md#6-the-evidence-so-far-with-sample-sizes)). |
| **`H2`** Single-pass entropy can miss order-sensitive items; cyclic JSD reveals them | ✅ **Supported** (n=4) | `perm_06` and `perm_08` have $\tilde{H} < 0.04$ with large cyclic JSD. |
| **`H3`** Dual-Mirror is `O(1)`, cancels position bias, gives a live disagreement signal | ⚠️ **Partly** | `O(1)` latency ✅; live signal ✅ (`perm_08`); Brier **worse** (`0.0410`); "0% reversal flip" is by construction; misses `perm_06`. |
| **`H4`** Entropy + Mirror gate catches every order-unstable ambiguity | ❌ **Not supported** | Catches `perm_07` and `perm_08`; misses `perm_06`. Also escalates 3 correct non-ambiguous items (`perm_10`, `perm_12`, `perm_16`). |

---

## 4. Reproducibility & CLI Usage

```bash
# Run from saved Cloud Run GPU receipt (instant table + JSON validation):
./bin/dgem bench-permutation --from-receipt benchmarks/results_permutation_cloudrun.json

# Run live against Cloud Run GPU (executes 13B Null-Prior probes + 13A K-Cyclic shifts + 13C Dual-Mirror Canvas):
./bin/dgem bench-permutation -u "https://dgemma-tkb3aiuiea-uc.a.run.app/v1" --gcp-auth -w 2
```
