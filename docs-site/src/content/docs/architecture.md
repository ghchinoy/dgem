---
title: Discrete Diffusion vs. Autoregression
description: Theoretical and mechanical breakdown of DiffusionGemma's discrete block diffusion canvas compared to sequential autoregression.
---

## 1. The Bottleneck of Autoregressive LLMs

Standard Large Language Models (LLMs) operate under a sequential autoregressive factorization:

$$P(x_1, x_2, \dots, x_T) = \prod_{t=1}^T P(x_t \mid x_{<t})$$

Each forward pass generates exactly **one token**. Even if the model only needs to output a single boolean decision or three fields of a JSON object:
1. It must sequentially predict structural characters (`{`, `\n`, `"`, `k`, `e`, `y`, `"`, `:`, ` `).
2. It suffers from high memory bandwidth pressure: loading billions of parameters from memory to compute a single token's logits.
3. It takes dozens or hundreds of forward passes (10–30 seconds on consumer hardware).

---

## 2. Discrete Block Diffusion & Multi-Canvas Sampling

**DiffusionGemma** breaks this sequential bottleneck by using **discrete diffusion**:

* **Block-Autoregressive Canvas**: The decoder works on a **256-token canvas** with bidirectional attention.
* **Iterative Denoising**: The entire block of tokens begins as noise and is iteratively denoised in parallel across a small number of denoising steps (typically 1 to 8 steps).
* **Throughput**: By predicting multiple tokens per step, DiffusionGemma achieves **15–20 tokens per forward pass**, decoupling throughput from pure sequential memory bandwidth.

---

## 3. How Jev-Style Structured Reading Works

In a structured decision query, no text is generated at all:

1. **Canvas Seeding**: The known question template (e.g. `urgent: @\nteam: @\nsentiment: @`) is pre-seeded into the canvas, where `@` represents masked noise tokens at the candidate slots.
2. **Single-Pass Denoise**: A single forward pass (or minimal diffusion step) is executed on the Metal GPU (~850–900 ms).
3. **Logit Readout**: Rather than decoding text, the engine reads the logits directly at each candidate slot index.
4. **Normalized Softmax**: The logits corresponding to the allowed single-token labels (`yes`/`no` for boolean, `A`/`B`/`C` for choices, `1`/`2`/`3` for scores) are extracted and normalized via softmax at temperature 1.

```
Seeded Canvas:
[<|channel>thought\n<channel|>urgent: @ \nteam: @ \nsentiment: @ ]
                                      ▲          ▲           ▲
                                   Slot 1     Slot 2      Slot 3
                              [p(yes), p(no)] [p(A), p(B)] [p(1), p(2), p(3)]
```

---

## 4. Empirical Uncertainty & The `auto` Sampling Policy

Unlike traditional LLMs that exhibit overconfident hallucinations, single-pass diffusion permits true empirical error estimation:

* **Shannon Entropy**: Calculated over the top logprobs for each slot:
  $$H = -\sum_{i} p_i \ln p_i$$
* **Adaptive Multi-Sampling (`samples: "auto"`)**:
  - If entropy across all slots is low ($H < 0.10$ nats), the answer is decisive. The engine concludes after **1 sample** (~880 ms).
  - If any slot exhibits entropy above the threshold ($H \ge 0.10$ nats), the question is ambiguous. The engine draws 3 additional independent noise vectors, computes the mean probability across reads, and reports the standard error:
    $$\text{stderr} = \frac{\sigma}{\sqrt{N}}$$
* **Agreement**: The percentage of independent noise draws that converged on the winning label ($0.0 \dots 1.0$).
