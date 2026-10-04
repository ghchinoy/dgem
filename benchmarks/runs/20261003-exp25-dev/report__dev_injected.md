# EXP-25 report: `dev_injected`

Rubric hash `cea4c82b5ad6`. Configurations: `dgem/isolated/original/r1`, `dgem/joint/original/r1`, `dgem/joint/original/r2`, `dgem/joint/shuffled/r1`, `docstats/r1`, `gemini:gemini-3.8-flash/r1`, `dgem/joint/original/r1+thr`.

## Accuracy per question

| Question | `dgem/isolated/original/r1` | `dgem/joint/original/r1` | `dgem/joint/original/r2` | `dgem/joint/shuffled/r1` | `docstats/r1` | `gemini:gemini-3.8-flash/r1` | `dgem/joint/original/r1+thr` |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `openers` | 0.731 F1 0.515 AUROC 0.947 (n=175) | 0.714 F1 0.5 AUROC 0.962 (n=175) | 0.726 F1 0.51 AUROC 0.959 (n=175) | 0.743 F1 0.526 AUROC 0.992 (n=175) | 1.0 F1 1.0 (n=175) | 0.897 F1 0.735 (n=175) | 0.926 F1 0.745 AUROC 0.962 (n=175) |
| `framing` | 0.789 F1 0.493 AUROC 0.77 (n=175) | 0.88 F1 0.553 AUROC 0.781 (n=175) | 0.88 F1 0.604 AUROC 0.744 (n=175) | 0.846 F1 0.509 AUROC 0.778 (n=175) | 0.937 F1 0.718 (n=175) | 0.977 F1 0.926 (n=175) | 0.863 F1 0.586 AUROC 0.781 (n=175) |
| `actors` | 0.709 F1 0.32 AUROC 0.635 (n=175) | 0.874 F1 0.267 AUROC 0.716 (n=175) | 0.846 F1 0.229 AUROC 0.709 (n=175) | 0.863 F1 0.143 AUROC 0.72 (n=175) | 0.783 F1 0.537 (n=175) | 0.931 F1 0.769 (n=175) | 0.783 F1 0.424 AUROC 0.716 (n=175) |
| `sentences` | 0.886 F1 0.688 AUROC 0.961 (n=175) | 0.743 F1 0.526 AUROC 0.994 (n=175) | 0.76 F1 0.543 AUROC 0.995 (n=175) | 0.966 F1 0.885 AUROC 0.989 (n=175) | 0.949 F1 0.78 (n=175) | 0.994 F1 0.98 (n=175) | 0.977 F1 0.923 AUROC 0.994 (n=175) |
| `reader` | 0.76 F1 0.543 AUROC 0.999 (n=175) | 0.737 F1 0.521 AUROC 0.994 (n=175) | 0.743 F1 0.526 AUROC 0.987 (n=175) | 0.909 F1 0.758 AUROC 0.994 (n=175) | — | 0.954 F1 0.862 (n=175) | 0.971 F1 0.909 AUROC 0.994 (n=175) |
| `tone` | 0.903 F1 0.746 AUROC 0.998 (n=175) | 0.931 F1 0.806 AUROC 1.0 (n=175) | 0.937 F1 0.82 AUROC 1.0 (n=175) | 0.977 F1 0.926 AUROC 1.0 (n=175) | — | 0.994 F1 0.98 (n=175) | 1.0 F1 1.0 AUROC 1.0 (n=175) |

| Configuration | Mean accuracy (answers) | Hesitation AUROC for errors | p50 per request / per section |
| :--- | ---: | ---: | ---: |
| `dgem/isolated/original/r1` | 0.796 (1050) | 0.642 (214 errors) | 311.4 / 3194.6 ms |
| `dgem/joint/original/r1` | 0.813 (1050) | 0.75 (196 errors) | 359.8 / 1092.9 ms |
| `dgem/joint/original/r2` | 0.815 (1050) | 0.739 (194 errors) | 383.6 / 1198.9 ms |
| `dgem/joint/shuffled/r1` | 0.884 (1050) | 0.76 (122 errors) | 397.5 / 1198.4 ms |
| `docstats/r1` | 0.917 (700) | — (0 errors) | None / 0.0 ms |
| `gemini:gemini-3.8-flash/r1` | 0.958 (1050) | — (0 errors) | 6437.1 / 16575.5 ms |
| `dgem/joint/original/r1+thr` | 0.92 (1050) | 0.63 (84 errors) | 359.8 / 1092.9 ms |

## Hesitation-gated cascade (offline; pre-registered threshold 0.35; kept answers from `dgem/joint/original/r1+thr`)

| Hesitation ≥ | Accuracy | 95% CI | Handed to Gemini |
| ---: | ---: | :--- | ---: |
| 0.16 | 0.95 | [0.936, 0.962] | 0.303 |
| 0.25 | 0.949 | [0.935, 0.962] | 0.224 |
| 0.35 | 0.945 | [0.931, 0.958] | 0.171 |
| 0.5 | 0.938 | [0.924, 0.952] | 0.12 |

## Answer changes vs `dgem/joint/original/r1`

| Comparison | Flip rate | Answers |
| :--- | ---: | ---: |
| option order shuffled | 0.131 | 1750 |
| identical repeat | 0.078 | 1750 |
| one question per request | 0.157 | 1750 |
