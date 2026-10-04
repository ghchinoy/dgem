# EXP-25 labeling instructions

You are labelling 165 sections of this repository's documentation: 150 sections plus 15 repeats, shuffled in with
different ids. The repeats measure how consistent one labeller is, which sets the ceiling for any model. Your labels
are the **frozen test set**. The models are scored against them once, after the method has been fixed on development
data. Plan on about 90 minutes, in more than one sitting if you like. Progress is saved in your browser.

**Please don't look at model outputs or `benchmarks/runs/*exp25*` until you have finished**, and don't edit the
sections. Label what you see.

## Setup

1. Open `scripts/docs_eval/labeler.html` in a browser by double-clicking it. It needs no server and no network.
2. Load `benchmarks/docs_eval/label_set.jsonl` with the first file picker.
3. Label with the keyboard. `→` or `Enter` goes to the next section, `←` to the previous one. **First unlabelled**
   jumps to where you stopped.
4. When you finish, or before you close the browser, click **Export labels.jsonl** and save it as
   `benchmarks/docs_eval/labels.jsonl`. To continue on another machine, load the exported file with the second
   picker.

Each section starts with `Page: <title>`. Code blocks show only their first 4 lines and tables only their first
6 rows; the rest is marked `...`. Judge the section as shown.

## The five labels

Four labels are required (red until set); the problems list may stay empty.

### 1. Type: what the section does for its reader (`1`–`6`)

Pick the **dominant** purpose. Ignore where the page sits in the sidebar and what its title claims.

| Key | Label | Choose it when the reader… | Typical signs |
| :---: | :--- | :--- | :--- |
| `1` | `tutorial` | is learning, and the text leads them through an exercise to a known result | "In this tutorial", numbered steps that build on each other, "you should now see" |
| `2` | `how_to` | already knows the basics and wants to get a task done | imperative steps, commands, "to do X, run Y", prerequisites, troubleshooting |
| `3` | `reference` | is looking up a fact while working | tables of flags, fields, defaults, endpoints or limits; terse, complete, neutral |
| `4` | `explanation` | wants to understand why or how something works | concepts, design reasons, trade-offs, comparisons, analogies |
| `5` | `results_report` | is weighing evidence | an experiment's setup, measured numbers, findings, caveats, receipts |
| `6` | `navigation` | is being sent somewhere else | mostly a list of links or pages, little content of its own |

Ways to break a tie:
- **Tutorial or how-to?** A tutorial chooses the task for the reader so they can learn. A how-to serves a goal the
  reader already has. Most "Step N" deploy pages are how-tos.
- **Reference or results report?** A table of defaults or limits is reference. A table of measured accuracy or latency
  from a test run is a results report, unless it sits inside a guide as supporting evidence; then label the guide.
- **Explanation or results report?** If the numbers are the point, choose results report. If the numbers support an
  argument about how something works, choose explanation.

### 2. Purity (`q` `w` `e`)

- `q` **single_purpose**: everything serves the type you chose.
- `w` **mostly_one_type**: one type dominates, with a short aside of another type (a paragraph of background in a
  how-to, one command in an explanation).
- `e` **clearly_mixed**: two or more types compete. Examples: a how-to that turns into an essay, reference tables in
  the middle of a tutorial, a reference page that argues for the product.

### 3. Style problems present (toggle `a` `s` `d` `f` `g` `h`)

Tick a problem if it appears **at least once in the prose**. Ignore code, tables, headings and quoted examples of bad
style. These rules come from the `technical-post-editorial` house style.

| Key | Problem | Tick when you see | Don't tick for |
| :---: | :--- | :--- | :--- |
| `a` | **openers** | announcing instead of saying: "It's worth noting that", "Here's the thing:", "This matters because", "That distinction matters.", "The implications are significant" | a heading or a lead-in that already states the fact ("The gateway retries twice:") |
| `s` | **framing** | drama through contrast: "This isn't X, it's Y", "X isn't the problem, Y is", stacked "not just… not just…" | a plain factual contrast that informs ("Cloud Run scales to zero; Vertex does not") |
| `d` | **actors** | the doer is hidden where naming it would help: "the config was changed", "errors are handled", "the data tells us" | passive voice where the actor really doesn't matter or is obvious ("the image is published on each release" in a release table) |
| `f` | **sentences** | fragments or very short punchy sentences used for effect: "That's it.", "Fast. Simple. Cheap." | list items, table cells, labels |
| `g` | **reader** | hand-holding or meta-commentary: "Don't worry", "As you can see", "Simply", "Let's dive in", "In this section we will" | a necessary pointer ("See the runbook for rollback.") |
| `h` | **tone** | hype: "blazing fast", "seamless", "revolutionary", "paradigm shift", superlatives, or claims of superiority with no measurement | a specific measured claim ("p50 143 ms, n=500") |

Don't count em dashes, adverbs or bold text. Those are checked by deterministic rules, not by this set.

### 4. Style verdict (`z` `x` `c`)

- `z` **ship**: you would publish it as is.
- `x` **light_edit**: a few sentences need fixing.
- `c` **rewrite**: most of the section needs rewriting for style.

The verdict covers **style only**, not accuracy, structure or Diátaxis purity. A section with one problem ticked is
usually `light_edit`; a section with none is usually `ship`.

### 5. Measured results (`v` `b` `n` `m`)

A **measured result** is a number obtained by testing: accuracy, latency, cost, error rate, throughput, or a count of
correct items. Configuration values, defaults, versions, ports, sizes, dates and limits are **not** measured results.

- `v` **no_measured_results**: none in the section.
- `b` **all_sourced**: every measured result gives its sample size (n=…, "on 231 items", "100 per suite") or links the
  experiment, run or receipt (EXP-xx, `benchmarks/runs/…`).
- `n` **some_unsourced**: at least one measured result has neither a sample size nor a source. Choose this even if
  other results in the section are sourced.
- `m` **illustrative_labelled**: the only numbers are explicitly marked illustrative, simulated or example values.

A table counts as sourced if its caption, header or the sentence that introduces it names the run or the sample size.

### Flag and note

- `u` **unsure**: set it when you had to guess on any label. Unsure items are reported separately, not dropped.
- **note**: optional free text, for example "two types, chose how_to because of the steps".

## Worked examples (not from the set)

> Page: Deploy on Cloud Run. **Step 3.** Run `./scripts/deploy_cloudrun_vllm.sh`. When it finishes, check `/health`
> returns 200. If the revision fails to start, read the logs with `gcloud run services logs read dgemma`.

`2` how_to · `q` single_purpose · no problems · `z` ship · `v` no_measured_results.

> Page: Overview. It's worth noting that dgem isn't just a classifier, it's a whole new way to make decisions. It's
> blazing fast. Seriously fast. Simply point it at your data and let the magic happen.

`4` explanation · `q` single_purpose · `a` `s` `f` `g` `h` · `c` rewrite · `v` no_measured_results.

> Page: Latency. On one G4 replica, the median latency for a 3-question decision is 143 ms (500 requests,
> `benchmarks/runs/20260925-serving-speed`), and throughput reaches 54 decisions/s at 16 concurrent clients.

`5` results_report · `q` single_purpose · no problems · `z` ship · `n` some_unsourced (the throughput figure has
neither a sample size nor a source of its own; the 143 ms figure is sourced).

## When you're done

Save the export as `benchmarks/docs_eval/labels.jsonl` and tell me. I'll check it is complete (165 rows), compute
your agreement with yourself on the 15 repeats, and run the single pre-registered scoring pass. If you want to change a
label after the scoring pass, the change and its reason go in the write-up; the original file stays as scored.
