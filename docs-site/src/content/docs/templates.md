---
title: Template Catalog
description: Complete reference guide to dgem decision schemas and templates across operational triage, autonomous agent routing, semiotic text normalization, and multimodal vision inspection.
---

import { Card, CardGrid, Tabs, TabItem, Badge } from '@astrojs/starlight/components';

`dgem` leverages Go's `text/template` engine to define declarative decision schemas. Each template pairs a **Decision Schema** (specifying the discrete questions, authorized labels, and sampling policy) with an **Input State** (injecting variables, text payloads, and optional multimodal references).

---

## 1. Operational Decisions & Risk Triage

<Tabs>
  <TabItem label="Customer Support Triage">
    **File**: `templates/support_triage.json.tmpl`

    Triages customer operations tickets across department, same-day urgency, and emotional distress in a single forward pass (~880 ms).

    ```json
    {
      "schema": {
        "instructions": "You are triaging incoming customer operations tickets.",
        "questions": [
          {
            "id": "urgent",
            "type": "boolean",
            "instructions": "Does this issue require immediate same-day escalation?"
          },
          {
            "id": "team",
            "type": "choice",
            "instructions": "Which department owns resolution of this ticket?",
            "options": [
              {"name": "billing", "description": "Charges, invoices, payment methods, renewals"},
              {"name": "support", "description": "General questions, password resets, account setup"},
              {"name": "engineering", "description": "System outages, 500 errors, bugs, API failures"}
            ]
          },
          {
            "id": "sentiment",
            "type": "score",
            "instructions": "Customer anger or distress level",
            "levels": ["calm", "frustrated", "furious"]
          }
        ],
        "samples": "auto"
      },
      "state": {
        "ticket": {{ default "No ticket text provided" .ticket | toJson }}
      }
    }
    ```

    ```bash
    ./bin/dgem decide -t templates/support_triage.json.tmpl \
      -v 'ticket=Emergency: database returned 500 across all cluster nodes!' \
      --stats
    ```
  </TabItem>

  <TabItem label="PR Code Review & Risk">
    **File**: `templates/code_review.json.tmpl`

    Performs immediate pre-merge risk assessment and security classification on code diffs or commit summaries.

    ```json
    {
      "schema": {
        "instructions": "You are a senior software engineer reviewing a pull request diff.",
        "questions": [
          {
            "id": "approved",
            "type": "boolean",
            "instructions": "Is this code safe and sound to merge without blocking issues?"
          },
          {
            "id": "category",
            "type": "choice",
            "instructions": "Primary classification of this change",
            "options": [
              {"name": "bugfix", "description": "Fixes broken functionality or defect"},
              {"name": "feature", "description": "Adds new capability or endpoint"},
              {"name": "refactor", "description": "Internal code reorganization without behavior change"},
              {"name": "security", "description": "Patches vulnerability, auth, or secret exposure"}
            ]
          },
          {
            "id": "risk_level",
            "type": "score",
            "instructions": "Deployment risk assessment",
            "levels": ["low", "medium", "high", "critical"]
          }
        ],
        "samples": "auto"
      },
      "state": {
        "diff": {{ default "No diff provided" .diff | toJson }}
      }
    }
    ```

    ```bash
    ./bin/dgem decide -t templates/code_review.json.tmpl \
      -v 'diff=Replace raw string concatenation in SQL query with parameterized statements to eliminate SQL injection vulnerability.' \
      --stats
    ```
  </TabItem>

  <TabItem label="SecOps Incident Containment">
    **File**: `templates/security_incident.json.tmpl`

    Evaluates SIEM alerts to recommend automated containment actions and triage severity.

    ```json
    {
      "schema": {
        "instructions": "You are an automated SecOps incident responder analyzing an alert.",
        "questions": [
          {
            "id": "data_compromise",
            "type": "boolean",
            "instructions": "Is there plausible evidence of unauthorized data exfiltration or access?"
          },
          {
            "id": "action",
            "type": "choice",
            "instructions": "Primary immediate containment action required",
            "options": [
              {"name": "monitor", "description": "False positive or low risk log anomaly"},
              {"name": "revoke_key", "description": "Rotate leaked API key or credentials"},
              {"name": "isolate_host", "description": "Quarantine compromised server instance"},
              {"name": "page_oncall", "description": "Immediate high-priority security team page"}
            ]
          },
          {
            "id": "severity",
            "type": "score",
            "instructions": "Incident severity tier",
            "levels": ["p3_low", "p2_medium", "p1_high", "p0_critical"]
          }
        ],
        "samples": "auto"
      },
      "state": {
        "alert": {{ default "No alert provided" .alert | toJson }}
      }
    }
    ```

    ```bash
    ./bin/dgem decide -t templates/security_incident.json.tmpl \
      -v 'alert=Critical: Production AWS root credentials detected committed in public GitHub repository.' \
      --stats
    ```
  </TabItem>
</Tabs>

---

## 2. Autonomous Agent Dispatch & High-Cardinality NLU

<Tabs>
  <TabItem label="Agent Tool Router">
    **File**: `templates/agent_router.json.tmpl`

    Determines an autonomous agent's next action in a single forward pass without generating conversational chain-of-thought tokens.

    ```json
    {
      "schema": {
        "instructions": "Select the single best tool for the agent to execute next based on user intent.",
        "questions": [
          {
            "id": "tool",
            "type": "choice",
            "instructions": "Next immediate agent action",
            "options": [
              {"name": "search", "description": "Search web or documentation for external facts"},
              {"name": "read_code", "description": "Inspect local repository files or definitions"},
              {"name": "edit_code", "description": "Modify existing files or write new code"},
              {"name": "run_tests", "description": "Execute test suite or verify changes"},
              {"name": "ask_user", "description": "Clarify ambiguous instructions with the human"}
            ]
          }
        ],
        "samples": "auto"
      },
      "state": {
        "prompt": {{ default "No prompt provided" .prompt | toJson }},
        "last_action": {{ default "none" .last_action | toJson }}
      }
    }
    ```

    ```bash
    ./bin/dgem decide -t templates/agent_router.json.tmpl \
      -v 'prompt=Can you fix the unit test failure in pkg/client?' \
      -v 'last_action=run_tests' \
      --stats
    ```
  </TabItem>

  <TabItem label="Banking77 Intent Router">
    **File**: `templates/intent_banking77.json.tmpl`

    Routes customer utterances into 77 fine-grained financial categories with dynamic option projection. Used by `dgem bench-intents --dataset banking77`.

    ```json
    {
      "schema": {
        "instructions": "You are an enterprise retail banking intent router (PolyAI/Banking77). Classify the customer utterance into the exact matching banking intent from the allowed options. Output a JSON object with the key \"intent\" set to the exact option name.",
        "questions": [
          {
            "id": "intent",
            "type": "choice",
            "instructions": "Select the exact fine-grained Banking77 intent for this customer query.",
            "options": [
              {{ range $i, $opt := .options }}
              {{ if $i }},{{ end }}{"name": {{ $opt | toJson }}}
              {{ end }}
            ]
          }
        ],
        "samples": {{ default "auto" .samples | toJson }}
      },
      "state": {
        "customer_utterance": {{ .text | toJson }}
      }
    }
    ```
  </TabItem>

  <TabItem label="CLINC150 Multi-Domain + OOS">
    **File**: `templates/intent_clinc150.json.tmpl`

    Simultaneously evaluates domain, intent, and Out-of-Scope (`oos`) status across 150 intents in 10 domains. Used by `dgem bench-intents --dataset clinc150`.

    ```json
    {
      "schema": {
        "instructions": "You are a multi-domain virtual assistant router with Out-of-Scope (OOS) rejection (CLINC150). Determine both the high-level domain and the specific intent for the user utterance. If the request falls outside the supported domains/intents, select \"oos\" for both \"domain\" and \"intent\". Output a JSON object with keys \"domain\" and \"intent\".",
        "questions": [
          {
            "id": "domain",
            "type": "choice",
            "instructions": "High-level service domain, or \"oos\" if out-of-scope.",
            "options": [
              {{ range $i, $dom := .domains }}
              {{ if $i }},{{ end }}{"name": {{ $dom | toJson }}}
              {{ end }}
            ]
          },
          {
            "id": "intent",
            "type": "choice",
            "instructions": "Specific user intent within the domain, or \"oos\" if out-of-scope.",
            "options": [
              {{ range $i, $opt := .options }}
              {{ if $i }},{{ end }}{"name": {{ $opt | toJson }}}
              {{ end }}
            ]
          }
        ],
        "samples": {{ default "auto" .samples | toJson }}
      },
      "state": {
        "user_utterance": {{ .text | toJson }}
      }
    }
    ```
  </TabItem>

  <TabItem label="Entropy-Gated Tie-Breaker">
    **File**: `templates/intent_tiebreak.json.tmpl`

    Dynamically invoked when Shannon entropy $H \ge 0.08$ nats, focusing the canvas strictly on the top 2–5 competing finalist candidates.

    ```json
    {
      "schema": {
        "instructions": "High-entropy tie-breaker: Multiple candidate intents had competing probability mass on the first pass. Carefully compare the exact wording of the user utterance against ONLY these finalist intents... Output a JSON object with key \"intent\" set to the single most accurate finalist option.",
        "questions": [
          {
            "id": "intent",
            "type": "choice",
            "instructions": "Select the exact best-matching finalist intent.",
            "options": [
              {{ range $i, $opt := .options }}
              {{ if $i }},{{ end }}{"name": {{ $opt | toJson }}}
              {{ end }}
            ]
          }
        ],
        "samples": 1
      },
      "state": {
        "utterance": {{ .text | toJson }},
        "first_pass_guess": {{ .first_guess | toJson }},
        "candidate_finalists": {{ .options | toJson }}
      }
    }
    ```
  </TabItem>
</Tabs>

---

## 3. Semiotic Text Normalization & Speech Sidecars

<Tabs>
  <TabItem label="Semiotic Disambiguation">
    **File**: `templates/tn_disambiguation.json.tmpl`

    The template driving the Ecotone benchmark (`dgem bench-ecotone`). Resolves truth-conditional semiotic polysemy (`St.` as Saint vs. Street; `1984` as year vs. quantity; `lead` as metal vs. verb) conditioned on full sentence context.

    ```json
    {
      "schema": {
        "instructions": "Determine the correct forward text normalization (spoken expansion) for the ambiguous token in the provided context. Output a JSON object with the key \"expansion\" set to the exact selected option name.",
        "questions": [
          {
            "id": "expansion",
            "type": "choice",
            "instructions": {{ printf "Select the exact spoken verbalization for target token '%s'" .target_token | toJson }},
            "options": [
              {{ range $i, $opt := .options }}
              {{ if $i }},{{ end }}{"name": {{ $opt | toJson }}}
              {{ end }}
            ]
          }
        ],
        "samples": {{ default "auto" .samples | toJson }}
      },
      "state": {
        "sentence": {{ .input | toJson }},
        "target_token": {{ .target_token | toJson }}
      }
    }
    ```

    ```bash
    ./bin/dgem decide -t templates/tn_disambiguation.json.tmpl \
      -v 'input=Deliver the package to 123 St. Mark St., Apt. 4B.' \
      -v 'target_token=St.' \
      --stats
    ```
  </TabItem>
</Tabs>

---

## 4. Multimodal Vision & Audio Inspection (`templates/multimodal/`)

DiffusionGemma natively accepts image files (PNG, JPEG, WebP) directly alongside text state inputs.

<Tabs>
  <TabItem label="UI Design & Accessibility Review">
    **File**: `templates/multimodal/ui_design_review.json.tmpl`

    Audits rendered web pages and application mockups for WCAG 4.5:1 contrast compliance, visual hierarchy, and layout density.

    ```json
    {
      "schema": {
        "instructions": "Review the rendered UI screen screenshot against accessibility and layout design standards.",
        "questions": [
          {
            "id": "wcag_contrast_pass",
            "type": "boolean",
            "instructions": "Does body copy and primary button text meet minimum WCAG 4.5:1 contrast against the background?"
          },
          {
            "id": "primary_call_to_action",
            "type": "choice",
            "instructions": "The most prominent visual focal point on the page",
            "options": [
              {"name": "checkout_button", "description": "High-contrast action button"},
              {"name": "promotional_banner", "description": "Marketing image or carousel banner"},
              {"name": "nav_menu", "description": "Navigation header"},
              {"name": "none_diffuse", "description": "No clear visual hierarchy"}
            ]
          },
          {
            "id": "visual_density",
            "type": "score",
            "instructions": "Information density and whitespace balance",
            "levels": ["sparse", "balanced_modern", "cramped_cluttered"]
          }
        ],
        "samples": "auto"
      },
      "state": {
        "viewport": {{ default "desktop-1440px" .viewport | toJson }},
        "component_id": {{ default "CheckoutForm" .component | toJson }}
      }
    }
    ```

    ```bash
    ./bin/dgem decide -t templates/multimodal/ui_design_review.json.tmpl \
      -i /path/to/screenshot.png \
      -v 'viewport=mobile-375px' \
      --stats
    ```
  </TabItem>

  <TabItem label="PCB Manufacturing Defect Triage">
    **File**: `templates/multimodal/pcb_defect_triage.json.tmpl`

    Automated Optical Inspection (AOI) for electronics manufacturing, identifying solder bridges, tombstoning, and component misalignment.

    ```json
    {
      "schema": {
        "instructions": "Inspect the attached surface-mount PCB optical inspection image for manufacturing defects.",
        "questions": [
          {
            "id": "defect_detected",
            "type": "boolean",
            "instructions": "Is there visible evidence of a manufacturing defect?"
          },
          {
            "id": "defect_type",
            "type": "choice",
            "instructions": "Primary defect classification",
            "options": [
              {"name": "none", "description": "Board passes visual inspection"},
              {"name": "solder_bridge", "description": "Unintended solder short between adjacent pins"},
              {"name": "missing_component", "description": "Empty pad where SMD component should be placed"},
              {"name": "tombstoning", "description": "Passive component lifted vertically on one terminal"},
              {"name": "misalignment", "description": "Component rotated or offset from pad footprint"}
            ]
          },
          {
            "id": "action_tier",
            "type": "score",
            "instructions": "Rework severity and line intervention required",
            "levels": ["pass", "minor_rework", "scrap_and_halt_line"]
          }
        ],
        "samples": "auto"
      },
      "state": {
        "inspection_point": {{ default "SMD-Line-B-Camera-04" .camera_id | toJson }},
        "notes": {{ default "routine automated optical inspection" .notes | toJson }}
      }
    }
    ```

    ```bash
    ./bin/dgem decide -t templates/multimodal/pcb_defect_triage.json.tmpl \
      -i /path/to/circuit_macro.jpg \
      -v 'camera_id=Line-3-Optical-01' \
      --stats
    ```
  </TabItem>

  <TabItem label="KYC Document Verification">
    **File**: `templates/multimodal/kyc_document_audit.json.tmpl`

    Assesses customer identity documents for legibility, document type, glare, and digital tampering or screen recapture indicators.

    ```json
    {
      "schema": {
        "instructions": "Assess the uploaded identity verification document image for compliance and fraud indicators.",
        "questions": [
          {
            "id": "document_legible",
            "type": "boolean",
            "instructions": "Is text and portrait clearly legible without blinding glare or blur?"
          },
          {
            "id": "document_type",
            "type": "choice",
            "instructions": "Recognized document format",
            "options": [
              {"name": "passport", "description": "Official national passport booklet"},
              {"name": "drivers_license", "description": "State or regional driver identification card"},
              {"name": "national_id", "description": "Government-issued citizen identity card"},
              {"name": "unsupported", "description": "Utility bill, credit card, or unrecognizable document"}
            ]
          },
          {
            "id": "tampering_suspected",
            "type": "boolean",
            "instructions": "Is there visible evidence of digital alteration, physical photo replacement, or screen-recapture?"
          },
          {
            "id": "image_quality",
            "type": "score",
            "instructions": "Overall capture quality rating",
            "levels": ["unusable", "poor", "acceptable", "pristine"]
          }
        ],
        "samples": "auto"
      },
      "state": {
        "country_code": {{ default "US" .country | toJson }},
        "submission_id": {{ default "sub-auto-001" .sub_id | toJson }}
      }
    }
    ```
  </TabItem>

  <TabItem label="Workplace Safety & Hazard Detection">
    **File**: `templates/multimodal/workplace_hazard_video.json.tmpl`

    Audits sequential video frames from warehouse and industrial facilities for PPE compliance, spills, and blocked fire exits.

    ```json
    {
      "schema": {
        "instructions": "Analyze the sequential video frames from the warehouse loading dock camera for safety violations.",
        "questions": [
          {
            "id": "ppe_compliant",
            "type": "boolean",
            "instructions": "Are all visible workers wearing required hardhats and high-visibility safety vests?"
          },
          {
            "id": "primary_hazard",
            "type": "choice",
            "instructions": "Most immediate safety hazard observed across the sequence",
            "options": [
              {"name": "none", "description": "Safe operating conditions"},
              {"name": "forklift_pedestrian", "description": "Pedestrian walking within 2 meters of active moving forklift"},
              {"name": "spill_slip", "description": "Liquid spill or debris obstructing walkway"},
              {"name": "blocked_egress", "description": "Pallets or cargo blocking marked fire exit"},
              {"name": "improper_lifting", "description": "Worker lifting heavy crate without mechanical assistance"}
            ]
          },
          {
            "id": "urgency",
            "type": "score",
            "instructions": "Immediate risk level to human life or limb",
            "levels": ["safe", "cautionary_advisory", "imminent_hazard_stop_work"]
          }
        ],
        "samples": "auto"
      },
      "state": {
        "zone_id": {{ default "Dock-Door-12" .zone | toJson }},
        "frame_rate_fps": {{ default 1 .fps | toJson }}
      }
    }
    ```
  </TabItem>

  <TabItem label="Contact Center Call Quality">
    **File**: `templates/multimodal/call_quality_audio.json.tmpl`

    Evaluates customer service interactions across disclosure compliance, first-contact resolution, agent empathy, and final customer sentiment.

    ```json
    {
      "schema": {
        "instructions": "Evaluate the customer service call interaction for compliance, resolution, and emotional trajectory.",
        "questions": [
          {
            "id": "mandatory_disclosure_read",
            "type": "boolean",
            "instructions": "Did the representative explicitly read the mandatory call recording disclosure?"
          },
          {
            "id": "resolution_outcome",
            "type": "choice",
            "instructions": "Final disposition of customer inquiry",
            "options": [
              {"name": "resolved_first_contact", "description": "Issue completely resolved on this call"},
              {"name": "escalated_tier2", "description": "Transferred to technical specialist"},
              {"name": "callback_scheduled", "description": "Follow-up promise made"},
              {"name": "unresolved_customer_hung_up", "description": "Call ended abruptly without resolution"}
            ]
          },
          {
            "id": "agent_empathy",
            "type": "score",
            "instructions": "Agent active listening and empathy rating",
            "levels": ["adversarial", "robotic_indifferent", "adequate", "exemplary_empathetic"]
          },
          {
            "id": "customer_final_sentiment",
            "type": "score",
            "instructions": "Customer emotional state at call termination",
            "levels": ["furious", "dissatisfied", "neutral", "satisfied", "delighted"]
          }
        ],
        "samples": "auto"
      },
      "state": {
        "call_duration_seconds": {{ default 180 .duration | toJson }},
        "transcript_with_acoustic_tags": {{ default "Representative: Thank you for calling... Customer: My issue is fixed now, thank you!" .transcript | toJson }}
      }
    }
    ```
  </TabItem>
</Tabs>

---

## 5. Template Engine Functions Reference

The Go template engine in `dgem` exposes the following helper functions:

| Function | Signature | Description | Example |
| :--- | :--- | :--- | :--- |
| `toJson` | `toJson(v interface{}) (string, error)` | Marshals any value or struct into valid escaped JSON. | `{{ .ticket \| toJson }}` |
| `toPrettyJson`| `toPrettyJson(v interface{}) (string, error)` | Marshals value with 2-space indentation. | `{{ .options \| toPrettyJson }}` |
| `default` | `default(def, val interface{}) interface{}` | Returns fallback default if `val` is empty or nil. | `{{ default "auto" .samples }}` |
| `upper` | `upper(s string) string` | Converts string to uppercase. | `{{ upper .mode }}` |
| `lower` | `lower(s string) string` | Converts string to lowercase. | `{{ lower .status }}` |
| `trim` | `trim(s string) string` | Trims leading and trailing whitespace. | `{{ trim .text }}` |
| `indent` | `indent(spaces int, s string) string` | Prepends $N$ spaces to each non-empty line. | `{{ indent 2 .payload }}` |
