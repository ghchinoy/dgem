#!/usr/bin/env python3
"""
Generates the three figures used by `docs/confidence-beyond-shannon.md`
(Invariant Decision Calibration, IDC).

Figures 1 and 2 are *conceptual illustrations* built from synthetic Gaussians and
logistic curves. They show the shape of the problem (position bias + over-sharp
logits) and what each IDC step is meant to do. They contain no measured data and
every figure title says so.

Figure 3 is computed *only* from committed benchmark receipts:
  * benchmarks/results_permutation_cloudrun.json   (EXP-13, 16 synthetic items)
  * benchmarks/results_calibration_cloudrun.json   (EXP-04/11 baseline, 50 items)
  * benchmarks/results_calibration_null_prior.json (EXP-13B, 50 items)
  * benchmarks/results_calibration_dual_mirror.json(EXP-13C, 50 items)

Outputs are written as .png (gitignored) + .webp (committed) into both
docs/assets/idc/ and docs-site/public/assets/idc/.

Usage:
    python3 scripts/generate_idc_diagrams.py
"""

import json
import os

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt  # noqa: E402
import numpy as np  # noqa: E402
from matplotlib.gridspec import GridSpec  # noqa: E402
from PIL import Image  # noqa: E402

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
OUT_ROOTS = [
    os.path.join(REPO_ROOT, "docs", "assets", "idc"),
    os.path.join(REPO_ROOT, "docs-site", "public", "assets", "idc"),
]

plt.rcParams.update({
    "font.family": "sans-serif",
    "font.sans-serif": ["DejaVu Sans", "Arial", "Helvetica"],
    "axes.edgecolor": "#334155",
    "axes.linewidth": 1.0,
    "grid.color": "#CBD5E1",
    "grid.linestyle": "--",
    "grid.alpha": 0.5,
    "figure.facecolor": "#FFFFFF",
    "axes.facecolor": "#F8FAFC",
})

ILLUSTRATIVE_TAG = "ILLUSTRATION: synthetic curves, not measured data"


def load_json(rel_path):
    with open(os.path.join(REPO_ROOT, rel_path)) as f:
        return json.load(f)


def save_fig_formats(fig, name):
    """Saves .png (gitignored) and .webp (committed) into every OUT_ROOT."""
    for root in OUT_ROOTS:
        os.makedirs(root, exist_ok=True)
        png_path = os.path.join(root, name + ".png")
        webp_path = os.path.join(root, name + ".webp")
        fig.savefig(png_path, bbox_inches="tight", dpi=200)
        with Image.open(png_path) as im:
            im.save(webp_path, "WEBP", quality=90, method=6)


def sigmoid(z):
    return 1.0 / (1.0 + np.exp(-np.clip(z, -30, 30)))


# ---------------------------------------------------------------------------
# Figure 1 (illustrative): class overlap and how bias/sharpness distort confidence
# ---------------------------------------------------------------------------
def generate_fig1_class_intersections():
    fig = plt.figure(figsize=(13.8, 8.8))
    gs = GridSpec(2, 1, height_ratios=[1.0, 1.35], hspace=0.36)

    x = np.linspace(-3.6, 3.6, 800)
    mu, sigma_cls = 1.2, 1.0
    pdf_b = np.exp(-0.5 * ((x + mu) / sigma_cls) ** 2) / (np.sqrt(2 * np.pi) * sigma_cls)
    pdf_a = np.exp(-0.5 * ((x - mu) / sigma_cls) ** 2) / (np.sqrt(2 * np.pi) * sigma_cls)
    overlap = np.minimum(pdf_a, pdf_b)

    true_logit = 2.4 * x
    p_true = sigmoid(true_logit)

    # Toy model: position bias expressed as a logit offset for "Box A",
    # plus over-sharp logits (multiplier > 1). Values chosen for visual clarity.
    b_pos = 2.02  # ~= ln(0.883/0.117), the measured K=2 null prior
    sharp = 2.35
    p_raw_fwd = sigmoid(sharp * (true_logit + 0.85 * b_pos))  # option 1 printed in Box A
    p_raw_rev = sigmoid(sharp * (true_logit - 0.85 * b_pos))  # option 1 printed in Box B
    p_tare = sigmoid(1.35 * true_logit)                        # bias removed, still too sharp
    p_tare_temp = sigmoid((1.35 / 1.30) * true_logit)          # + temperature T=1.30

    ax1 = fig.add_subplot(gs[0])
    ax1.plot(x, pdf_b, color="#2563EB", lw=2.6, label="Evidence when the true answer is Option 2")
    ax1.fill_between(x, 0, pdf_b, color="#3B82F6", alpha=0.14)
    ax1.plot(x, pdf_a, color="#059669", lw=2.6, label="Evidence when the true answer is Option 1")
    ax1.fill_between(x, 0, pdf_a, color="#10B981", alpha=0.14)
    ax1.fill_between(x, 0, overlap, color="#F59E0B", alpha=0.42, hatch="//", edgecolor="#D97706",
                     label="Overlap: genuinely ambiguous inputs (or a key fact is missing)")
    ax1.axvline(0.0, color="#0F172A", lw=1.8, ls="--", label="Where an honest model should say 50/50")
    ax1.annotate("Genuine toss-ups live here.\nA trustworthy score must say ~50%.",
                 xy=(0.0, 0.15), xytext=(-0.6, 0.30), fontsize=9.2, fontweight="bold", color="#92400E",
                 ha="center", bbox=dict(boxstyle="round,pad=0.35", facecolor="#FEF3C7", edgecolor="#F59E0B"),
                 arrowprops=dict(arrowstyle="->", color="#D97706", lw=1.6))
    ax1.set_title("A. Two classes overlap: some inputs are honestly ambiguous", fontsize=12.5,
                  fontweight="bold", color="#0F172A", pad=10)
    ax1.set_ylabel("How often this evidence appears", fontsize=10.5, fontweight="bold")
    ax1.set_xlim(-3.5, 3.5)
    ax1.set_ylim(0, 0.56)
    ax1.grid(True)
    ax1.legend(loc="upper center", fontsize=8.6, framealpha=0.96, ncol=2)

    ax2 = fig.add_subplot(gs[1], sharex=ax1)
    trap = (x >= -0.72) & (x <= 0.15)
    ax2.fill_between(x[trap], p_true[trap], p_raw_fwd[trap], color="#EF4444", alpha=0.22,
                     label="False-certainty zone: raw score says >90% on a real toss-up")
    ax2.plot(x, p_raw_fwd, color="#DC2626", lw=2.4, label="Raw score, Option 1 printed in Box A (biased + too sharp)")
    ax2.plot(x, p_raw_rev, color="#F97316", lw=2.0, ls=":", label="Raw score, same input with the list reversed")
    ax2.plot(x, p_tare, color="#0284C7", lw=2.2, ls="-.", label="After removing the Box-A bias (null-prior)")
    ax2.plot(x, p_tare_temp, color="#059669", lw=3.0, label="After bias removal + temperature scaling")
    ax2.plot(x, p_true, color="#0F172A", lw=2.2, ls="--", label="Ideal: the true probability")
    ax2.annotate("Forward and reversed readings\ndisagree most exactly where\nthe input is ambiguous",
                 xy=(0.0, 0.5), xytext=(-3.3, 0.72), fontsize=9.0, fontweight="bold", color="#7C2D12",
                 bbox=dict(boxstyle="round,pad=0.35", facecolor="#FFEDD5", edgecolor="#F97316"),
                 arrowprops=dict(arrowstyle="->", color="#F97316", lw=1.5))
    ax2.axhline(0.5, color="#64748B", lw=1.0, ls=":")
    ax2.axvline(0.0, color="#0F172A", lw=1.5, ls="--")
    ax2.set_title("B. Confidence reported for Option 1 vs. the ideal", fontsize=12.5,
                  fontweight="bold", color="#0F172A", pad=10)
    ax2.set_xlabel("Strength of evidence (negative = favors Option 2, 0 = balanced, positive = favors Option 1)",
                   fontsize=10.5, fontweight="bold")
    ax2.set_ylabel("Reported P(Option 1)", fontsize=10.5, fontweight="bold")
    ax2.set_ylim(-0.03, 1.04)
    ax2.grid(True)
    ax2.legend(loc="lower right", fontsize=8.4, framealpha=0.96)

    fig.suptitle("Why raw confidence can mislead, and what IDC corrects\n(" + ILLUSTRATIVE_TAG + ")",
                 fontsize=14, fontweight="bold", color="#0F172A", y=0.995)
    save_fig_formats(fig, "idc_1d_class_intersections")
    plt.close(fig)


# ---------------------------------------------------------------------------
# Figure 2 (illustrative): 2D decision boundary under each IDC step
# ---------------------------------------------------------------------------
def generate_fig2_5stage_boundaries_2d():
    fig, axes = plt.subplots(1, 5, figsize=(21.0, 5.2), sharey=True)
    fig.subplots_adjust(wspace=0.13, top=0.70, bottom=0.16)

    s = np.linspace(-3.0, 3.0, 220)
    s1, s2 = np.meshgrid(s, s)
    delta = s1 - s2
    b_pos = 1.65

    p_fwd = sigmoid(2.3 * (delta + b_pos))
    p_rev = sigmoid(2.3 * (delta - b_pos))
    p_tare = sigmoid(2.0 * delta)
    p_temp = sigmoid((2.0 / 1.30) * delta)

    stages = [
        ("1. Raw single reading", "Boundary pushed off the diagonal\nby the Box-A preference", "raw"),
        ("2. Print the ballot twice", "Forward + reversed on one canvas;\nthe wedge between them = disagreement", "mirror"),
        ("3. Remove the Box-A preference", "Null-prior correction moves the\nboundary back to the diagonal", "tare"),
        ("4. Temperature scaling", "Softens over-sharp scores\n(needs labeled data to fit T*)", "temp"),
        ("5. Gate", "Uncertain / disagreeing band goes\nto a stronger model or a human", "gate"),
    ]

    pts_1 = [(2.1, -1.5), (1.7, -1.8)]
    pts_2 = [(-1.6, 2.0), (-2.0, 1.4)]
    pts_amb = [(-0.35, 0.35), (0.10, -0.10)]

    for idx, (ax, (title, subtitle, mode)) in enumerate(zip(axes, stages)):
        if mode == "raw":
            ax.contourf(s1, s2, p_fwd, levels=np.linspace(0, 1, 21), cmap="RdYlGn", alpha=0.85)
            ax.plot(s, s, color="#0F172A", lw=1.6, ls="--")
            ax.plot(s, s + b_pos, color="#DC2626", lw=2.5)
            ax.text(-0.9, 2.1, "Ambiguous inputs\ncalled for Option 1", fontsize=8.2, fontweight="bold",
                    color="#991B1B", bbox=dict(boxstyle="round,pad=0.25", facecolor="#FEE2E2", edgecolor="#DC2626"))
        elif mode == "mirror":
            ax.contourf(s1, s2, 0.5 * (p_fwd + p_rev), levels=np.linspace(0, 1, 21), cmap="RdYlGn", alpha=0.75)
            ax.plot(s, s + b_pos, color="#DC2626", lw=2.0)
            ax.plot(s, s - b_pos, color="#EA580C", lw=2.0, ls="-.")
            ax.plot(s, s, color="#0F172A", lw=1.8, ls="--")
            ax.fill_between(s, s - b_pos, s + b_pos, color="#F59E0B", alpha=0.35, hatch="//", edgecolor="#D97706")
            ax.text(1.0, 1.9, "Forward and reversed\nreadings disagree", fontsize=8.2, fontweight="bold",
                    ha="center", va="center", color="#78350F",
                    bbox=dict(boxstyle="round,pad=0.28", facecolor="#FEF3C7", edgecolor="#D97706"))
        elif mode == "tare":
            ax.contourf(s1, s2, p_tare, levels=np.linspace(0, 1, 21), cmap="RdYlGn", alpha=0.85)
            ax.plot(s, s, color="#0284C7", lw=2.8)
            ax.plot(s, s + b_pos, color="#94A3B8", lw=1.3, ls=":")
            ax.annotate("", xy=(0.4, 0.4), xytext=(-0.6, 1.05),
                        arrowprops=dict(arrowstyle="->", color="#0284C7", lw=2.2))
        elif mode == "temp":
            ax.contourf(s1, s2, p_temp, levels=np.linspace(0, 1, 21), cmap="RdYlGn", alpha=0.85)
            ax.plot(s, s, color="#059669", lw=2.6)
            ax.text(0.0, -2.4, "Wider yellow band =\nless false certainty", fontsize=8.0, fontweight="bold",
                    ha="center", color="#065F46",
                    bbox=dict(boxstyle="round,pad=0.25", facecolor="#D1FAE5", edgecolor="#059669"))
        elif mode == "gate":
            ax.contourf(s1, s2, p_temp, levels=np.linspace(0, 1, 21), cmap="RdYlGn", alpha=0.88)
            ax.fill_between(s, s - 0.85, s + 0.85, color="#E0E7FF", alpha=0.75, edgecolor="#4F46E5", lw=1.6)
            ax.text(-1.3, -1.1, "Escalate", fontsize=9.0, fontweight="bold", ha="center", va="center",
                    color="#312E81", bbox=dict(boxstyle="round,pad=0.25", facecolor="#EEF2FF", edgecolor="#4F46E5"))
            ax.text(1.6, -2.5, "Answer\ndirectly", fontsize=8.2, fontweight="bold", ha="center", color="#065F46",
                    bbox=dict(boxstyle="round,pad=0.22", facecolor="#D1FAE5", edgecolor="#059669"))

        for (px, py) in pts_1:
            ax.scatter(px, py, color="#059669", s=55, edgecolor="white", lw=1.2, zorder=6)
        for (px, py) in pts_2:
            ax.scatter(px, py, color="#2563EB", s=55, edgecolor="white", lw=1.2, zorder=6)
        for (px, py) in pts_amb:
            ax.scatter(px, py, color="#DC2626" if mode == "raw" else "#D97706", marker="D", s=65,
                       edgecolor="white", lw=1.3, zorder=7)

        ax.set_xlim(-3.0, 3.0)
        ax.set_ylim(-3.0, 3.0)
        ax.set_title(title, fontsize=11.0, fontweight="bold", color="#0F172A", pad=34)
        ax.text(0.5, 1.03, subtitle, transform=ax.transAxes, fontsize=8.1, ha="center", va="bottom",
                color="#475569", fontweight="bold")
        ax.set_xlabel("Evidence for Option 1", fontsize=9.2, fontweight="bold")
        if idx == 0:
            ax.set_ylabel("Evidence for Option 2", fontsize=9.5, fontweight="bold")

    fig.suptitle("The IDC steps as decision-boundary geometry\n(" + ILLUSTRATIVE_TAG
                 + "; diamonds = ambiguous inputs)", fontsize=14, fontweight="bold", color="#0F172A", y=0.99)
    save_fig_formats(fig, "idc_5stage_decision_boundaries_2d")
    plt.close(fig)


# ---------------------------------------------------------------------------
# Figure 3 (measured): real EXP-13 trajectories + real 10-bin reliability
# ---------------------------------------------------------------------------
def bary(p_a, p_b, p_c):
    """Vertices: a = top, b = bottom-left, c = bottom-right."""
    return 0.5 * (2.0 * p_c + p_a), (np.sqrt(3.0) / 2.0) * p_a


def draw_simplex(ax, labels):
    h = np.sqrt(3.0) / 2.0
    ax.plot([0, 1, 0.5, 0], [0, 0, h, 0], color="#1E293B", lw=2.0)
    ax.text(0.5, h + 0.04, labels[0], ha="center", va="bottom", fontsize=9.0, fontweight="bold")
    ax.text(-0.02, -0.04, labels[1], ha="left", va="top", fontsize=9.0, fontweight="bold")
    ax.text(1.02, -0.04, labels[2], ha="right", va="top", fontsize=9.0, fontweight="bold")
    cx, cy = bary(1 / 3, 1 / 3, 1 / 3)
    ax.scatter([cx], [cy], color="#94A3B8", s=25, marker="+")
    ax.set_xlim(-0.12, 1.12)
    ax.set_ylim(-0.20, h + 0.16)
    ax.set_aspect("equal")
    ax.axis("off")


def plot_pt(ax, probs, keys, **kw):
    x, y = bary(*(probs[k] for k in keys))
    ax.scatter([x], [y], zorder=7, edgecolor="white", lw=1.2, **kw)
    return x, y


def reliability(receipt, temperature):
    """Recompute per-bin (conf, acc, n) from stored top_probabilities at a given temperature."""
    applied = receipt["jev_parity"]["temperature_applied"]
    bins = [[] for _ in range(10)]
    for c in receipt["cases"]:
        raw = {k: v ** applied for k, v in c["top_probabilities"].items()}  # undo stored scaling
        s = sum(raw.values())
        scaled = {k: (v / s) ** (1.0 / temperature) for k, v in raw.items()}
        s2 = sum(scaled.values())
        conf = max(scaled.values()) / s2
        bins[min(int(conf * 10), 9)].append((conf, 1.0 if c["accurate"] else 0.0))
    n = len(receipt["cases"])
    ece = sum(len(b) / n * abs(np.mean([q[0] for q in b]) - np.mean([q[1] for q in b])) for b in bins if b)
    pts = [(np.mean([q[0] for q in b]), np.mean([q[1] for q in b]), len(b)) for b in bins if b]
    return pts, ece


def generate_fig3_simplex_and_reliability():
    perm = load_json("benchmarks/results_permutation_cloudrun.json")
    cases = {c["id"][:7]: c for c in perm["cases"]}
    c08, c06 = cases["perm_08"], cases["perm_06"]

    fig, (ax_a, ax_b, ax_c) = plt.subplots(1, 3, figsize=(19.5, 6.9),
                                           gridspec_kw={"width_ratios": [1, 1, 1.15]})
    fig.subplots_adjust(wspace=0.22, top=0.80, bottom=0.25)

    # Panel A: perm_08 - single reading looks certain, mirror reveals order sensitivity
    k08 = ["incidental_fair_use", "requires_sync_license", "public_domain_waiver"]
    draw_simplex(ax_a, ["incidental_fair_use\n(expected)", "requires_sync_license", "public_domain_waiver"])
    canon = c08["cyclic_passes"][0]["raw_probs"]
    dm = c08["dual_mirror"]
    plot_pt(ax_a, canon, k08, color="#DC2626", s=110, marker="o",
            label="1 slot, original order: %.1f%%, H~=%.3f -> would auto-exit"
                  % (100 * canon["incidental_fair_use"], c08["canonical_normalized_h"]))
    plot_pt(ax_a, dm["fwd_probs"], k08, color="#0284C7", s=95, marker="s",
            label="Mirror canvas, forward slot: %.1f%%" % (100 * dm["fwd_probs"]["incidental_fair_use"]))
    plot_pt(ax_a, dm["rev_probs"], k08, color="#F97316", s=95, marker="^",
            label="Mirror canvas, reversed slot: %.1f%%" % (100 * dm["rev_probs"]["incidental_fair_use"]))
    plot_pt(ax_a, dm["ensemble_probs"], k08, color="#059669", s=120, marker="D",
            label="Average of the two: %.1f%%  (Mirror TVD = %.3f -> escalate)"
                  % (100 * dm["ensemble_probs"]["incidental_fair_use"], dm["tvd"]))
    ax_a.set_title("A. perm_08 (measured): the mirror catches\nfalse certainty", fontsize=11.2,
                   fontweight="bold", pad=10)
    ax_a.legend(loc="upper center", bbox_to_anchor=(0.5, -0.02), fontsize=8.0, framealpha=0.96)

    # Panel B: perm_06 - cyclic reorder flips, but reversal alone does not detect it
    k06 = ["neutral", "entailment", "contradiction"]
    draw_simplex(ax_b, ["neutral (expected)", "entailment", "contradiction"])
    colors = ["#DC2626", "#7C3AED", "#EA580C"]
    for i, p in enumerate(c06["cyclic_passes"]):
        order = "/".join(o[:4] for o in p["option_order"])
        plot_pt(ax_b, p["raw_probs"], k06, color=colors[i], s=100, marker="o",
                label="Separate pass, order %s -> %s" % (order, p["raw_winner"]))
    dm6 = c06["dual_mirror"]
    plot_pt(ax_b, dm6["ensemble_probs"], k06, color="#059669", s=120, marker="D",
            label="Mirror canvas: fwd and rev agree (TVD = %.4f) -> NOT flagged" % dm6["tvd"])
    ax_b.set_title("B. perm_06 (measured): a different ordering flips\nthe answer, but the mirror misses it",
                   fontsize=11.2, fontweight="bold", pad=10)
    ax_b.legend(loc="upper center", bbox_to_anchor=(0.5, -0.02), fontsize=8.0, framealpha=0.96)

    # Panel C: reliability recomputed from receipts
    base = load_json("benchmarks/results_calibration_cloudrun.json")
    nullp = load_json("benchmarks/results_calibration_null_prior.json")
    mirror = load_json("benchmarks/results_calibration_dual_mirror.json")
    series = [
        (base, 1.0, "#F59E0B", "^", "Baseline (T=1)"),
        (base, 1.35, "#A16207", "v", "Baseline + T*=1.35 (fit on these same 50 items)"),
        (nullp, 1.0, "#059669", "o", "Null-prior de-biasing (T=1)"),
        (mirror, 1.0, "#0284C7", "s", "Dual-mirror (T=1)"),
    ]
    ax_c.plot([0, 1], [0, 1], color="#0F172A", lw=1.6, ls="--", label="Perfect calibration")
    for rec, t, col, mk, lab in series:
        pts, ece = reliability(rec, t)
        xs, ys, ns = zip(*pts)
        ax_c.plot(xs, ys, color=col, lw=1.8, alpha=0.9)
        ax_c.scatter(xs, ys, s=[25 + 6 * n for n in ns], color=col, marker=mk, edgecolor="white", lw=1.0,
                     zorder=6, label="%s  ECE=%.3f" % (lab, ece))
    ax_c.set_xlim(0.35, 1.02)
    ax_c.set_ylim(-0.02, 1.05)
    ax_c.set_xlabel("Predicted confidence (bin mean)", fontsize=10.2, fontweight="bold")
    ax_c.set_ylabel("Observed accuracy in bin", fontsize=10.2, fontweight="bold")
    ax_c.set_title("C. 10-bin reliability, 50-item calibration suite (measured)\nmarker size = items in bin; "
                   "most items sit in the top bin", fontsize=11.2, fontweight="bold", pad=10)
    ax_c.grid(True)
    ax_c.legend(loc="upper center", bbox_to_anchor=(0.5, -0.13), fontsize=8.0, framealpha=0.96)

    fig.suptitle("Measured IDC behaviour from committed receipts (EXP-13 permutation suite, n=16; "
                 "calibration suite, n=50)\nSmall samples: read as directional evidence, not proof",
                 fontsize=13.2, fontweight="bold", color="#0F172A", y=0.985)
    save_fig_formats(fig, "idc_3way_simplex_and_reliability")
    plt.close(fig)


if __name__ == "__main__":
    generate_fig1_class_intersections()
    generate_fig2_5stage_boundaries_2d()
    generate_fig3_simplex_and_reliability()
    print("Regenerated IDC figures into:", ", ".join(OUT_ROOTS))
