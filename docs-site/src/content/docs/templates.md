---
title: Template Catalog
description: Reference guide to the built-in Go template definition files for triage, code review, and security response.
---

`dgem` utilizes Go's `text/template` engine to define question schemas and dynamically inject user data into the canvas.

---

## 1. Customer Support Triage (`templates/support_triage.json.tmpl`)

Categorizes incoming support requests into department, urgency, and distress level.

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
    "ticket": "Customer ticket text injected here"
  }
}
```

### Usage
```bash
./bin/dgem decide -t templates/support_triage.json.tmpl \
  -v 'ticket=I was charged twice for renewal!' \
  --stats
```

---

## 2. Pull Request Code Review (`templates/code_review.json.tmpl`)

Performs immediate risk assessment and categorization of code changes.

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
    "diff": "Diff or PR summary injected here"
  }
}
```

---

## 3. Security Alert Containment (`templates/security_incident.json.tmpl`)

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
    "alert": "Alert payload injected here"
  }
}
```
