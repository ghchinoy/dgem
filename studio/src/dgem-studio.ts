import { LitElement, html, css, svg } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import type {
  TemplateCatalogEntry,
  GatewayDecideResponse,
  GatewayStatusResponse,
} from './types.ts';

@customElement('dgem-studio')
export class DgemStudio extends LitElement {
  static styles = css`
    :host {
      display: block;
      min-height: 100vh;
      --bg: #0b0f17;
      --panel: #131b2e;
      --panel-alt: #19233c;
      --border: #263457;
      --text: #e8eefb;
      --muted: #94a3b8;
      --accent: #38bdf8;
      --pass: #22c55e;
      --warn: #f59e0b;
      --fail: #ef4444;
      background: var(--bg);
      color: var(--text);
    }

    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 14px 24px;
      background: var(--panel);
      border-bottom: 1px solid var(--border);
    }

    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      flex-wrap: wrap;
    }

    .brand h1 {
      margin: 0;
      font-size: 18px;
      font-weight: 700;
      letter-spacing: -0.02em;
    }

    .badge {
      font-size: 11px;
      padding: 3px 9px;
      border-radius: 999px;
      font-weight: 600;
      background: rgba(56, 189, 248, 0.15);
      color: var(--accent);
      border: 1px solid rgba(56, 189, 248, 0.35);
    }

    .status-pill {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
      padding: 6px 12px;
      border-radius: 999px;
      background: var(--panel-alt);
      border: 1px solid var(--border);
    }

    .dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: var(--muted);
    }
    .dot.ok {
      background: var(--pass);
      box-shadow: 0 0 8px var(--pass);
    }
    .dot.warn {
      background: var(--warn);
      box-shadow: 0 0 8px var(--warn);
    }

    main {
      max-width: 1400px;
      margin: 0 auto;
      padding: 22px;
      display: grid;
      grid-template-columns: 1fr 1.15fr;
      gap: 22px;
    }

    @media (max-width: 980px) {
      main {
        grid-template-columns: 1fr;
      }
    }

    .card {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 18px;
    }

    .card h2 {
      margin: 0 0 12px 0;
      font-size: 14px;
      color: var(--accent);
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    label {
      display: block;
      font-size: 12px;
      font-weight: 600;
      color: var(--muted);
      margin: 12px 0 5px;
    }

    select,
    input[type='text'],
    textarea {
      width: 100%;
      box-sizing: border-box;
      background: var(--bg);
      color: var(--text);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 9px 11px;
      font-size: 13px;
      font-family: inherit;
    }

    textarea {
      font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
      resize: vertical;
    }

    button.primary {
      margin-top: 16px;
      width: 100%;
      padding: 11px 16px;
      border: none;
      border-radius: 8px;
      background: linear-gradient(135deg, #0284c7, #38bdf8);
      color: #041019;
      font-weight: 700;
      font-size: 14px;
      cursor: pointer;
    }

    button.primary:disabled {
      opacity: 0.6;
      cursor: wait;
    }

    .slot-table {
      width: 100%;
      border-collapse: collapse;
      margin-top: 10px;
      font-size: 13px;
    }

    .slot-table th,
    .slot-table td {
      text-align: left;
      padding: 9px 10px;
      border-bottom: 1px solid var(--border);
    }

    .slot-table th {
      color: var(--muted);
      font-size: 11px;
      text-transform: uppercase;
    }

    .entropy-pill {
      display: inline-block;
      padding: 2px 8px;
      border-radius: 6px;
      font-weight: 600;
      font-size: 12px;
    }
    .entropy-low {
      background: rgba(34, 197, 94, 0.16);
      color: #4ade80;
    }
    .entropy-mid {
      background: rgba(245, 158, 11, 0.16);
      color: #fbbf24;
    }
    .entropy-high {
      background: rgba(239, 68, 68, 0.16);
      color: #f87171;
    }

    pre {
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px;
      overflow-x: auto;
      font-size: 12px;
      color: #cbd5e1;
    }

    .canvas-wrap {
      position: relative;
      display: inline-block;
      max-width: 100%;
      margin-top: 12px;
    }

    .canvas-wrap img {
      max-width: 100%;
      border-radius: 8px;
      display: block;
    }

    .canvas-wrap svg {
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      pointer-events: none;
    }
  `;

  @state() private catalog: TemplateCatalogEntry[] = [];
  @state() private selectedIdx = 0;
  @state() private vars: Record<string, string> = {};
  @state() private rawTemplate = '';
  @state() private uploadedDataURI = '';
  @state() private status: GatewayStatusResponse | null = null;
  @state() private loading = false;
  @state() private elapsedSec = 0;
  @state() private result: GatewayDecideResponse | null = null;
  @state() private errorMsg = '';

  private timerId?: number;

  connectedCallback() {
    super.connectedCallback();
    this.fetchStatus();
    this.fetchCatalog();
  }

  private async fetchStatus() {
    try {
      const res = await fetch('/api/status', { credentials: 'same-origin' });
      if (res.ok) {
        this.status = await res.json();
      }
    } catch {
      // ignore transient network errors
    }
  }

  private async fetchCatalog() {
    try {
      const res = await fetch('/api/templates', { credentials: 'same-origin' });
      const data = await res.json();
      this.catalog = data.templates || [];
      const defaultIdx = this.catalog.findIndex((t) => t.id === 'support_triage');
      this.selectTemplate(defaultIdx >= 0 ? defaultIdx : 0);
    } catch (err) {
      this.errorMsg = `Failed to load policy catalog: ${(err as Error).message}`;
    }
  }

  private selectTemplate(idx: number) {
    this.selectedIdx = idx;
    const t = this.catalog[idx];
    if (!t) return;
    this.rawTemplate = t.raw_template;
    this.vars = { ...(t.sample_vars || {}) };
    this.result = null;
    this.errorMsg = '';
  }

  private onImageChange(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      this.uploadedDataURI = String(reader.result || '');
    };
    reader.readAsDataURL(file);
  }

  private async executeDecision() {
    const t = this.catalog[this.selectedIdx];
    if (!t) return;
    this.loading = true;
    this.errorMsg = '';
    this.elapsedSec = 0;
    const start = Date.now();
    this.timerId = window.setInterval(() => {
      this.elapsedSec = Math.floor((Date.now() - start) / 1000);
    }, 1000);

    try {
      const res = await fetch('/api/decide', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          template: t.id,
          custom_template: this.rawTemplate,
          variables: this.vars,
          image: this.uploadedDataURI || undefined,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        this.errorMsg = data.error || JSON.stringify(data, null, 2);
      } else {
        this.result = data;
      }
    } catch (err) {
      this.errorMsg = (err as Error).message;
    } finally {
      if (this.timerId) clearInterval(this.timerId);
      this.loading = false;
      this.fetchStatus();
    }
  }

  private renderBboxOverlay() {
    if (!this.result) return null;
    const answers = this.result.answers || {};
    const parseCoord = (obj?: { label?: string; value?: unknown; choice?: string }) => {
      if (!obj) return null;
      const raw = obj.label || obj.value || obj.choice;
      if (raw === undefined || raw === null) return null;
      const m = String(raw).match(/(\d+)/);
      return m ? parseInt(m[1], 10) : null;
    };
    const ymin = parseCoord(answers.ymin);
    const xmin = parseCoord(answers.xmin);
    const ymax = parseCoord(answers.ymax);
    const xmax = parseCoord(answers.xmax);
    if (ymin === null || xmin === null || ymax === null || xmax === null) return null;

    return svg`
      <rect
        x="${xmin}"
        y="${ymin}"
        width="${Math.max(1, xmax - xmin)}"
        height="${Math.max(1, ymax - ymin)}"
        fill="rgba(56, 189, 248, 0.18)"
        stroke="#38bdf8"
        stroke-width="8"
      />
    `;
  }

  render() {
    const current = this.catalog[this.selectedIdx];
    const origin = window.location.origin;
    const curlBody = JSON.stringify({ variables: this.vars }, null, 2);

    return html`
      <header>
        <div class="brand">
          <h1>DiffusionGemma Decision Studio</h1>
          <span class="badge">Lit WebComponents</span>
          <span class="badge">O(1) Discrete Diffusion Readout</span>
        </div>
        <div class="status-pill">
          <span class="dot ${this.status?.reachable ? 'ok' : 'warn'}"></span>
          <span>
            ${this.status?.reachable
              ? `GPU Container Reachable (${this.status.upstream_url})`
              : 'Idle / Scaled-to-Zero (Auto-wakes on first request)'}
          </span>
        </div>
      </header>

      <main>
        <div class="card">
          <h2>1. Select Policy Template (.json.tmpl)</h2>
          <label for="tmplSelect">Policy Catalog (${this.catalog.length} templates)</label>
          <select
            id="tmplSelect"
            @change=${(e: Event) =>
              this.selectTemplate(Number((e.target as HTMLSelectElement).value))}
          >
            ${this.catalog.map(
              (t, i) => html`
                <option value=${i} ?selected=${i === this.selectedIdx}>
                  [${t.category}] ${t.id}
                </option>
              `
            )}
          </select>

          ${(current?.variables || []).map(
            (v) => html`
              <label>Variable: {{ .${v} }}</label>
              <textarea
                rows="2"
                .value=${this.vars[v] || ''}
                @input=${(e: Event) => {
                  this.vars = {
                    ...this.vars,
                    [v]: (e.target as HTMLTextAreaElement).value,
                  };
                }}
              ></textarea>
            `
          )}

          ${current?.multimodal
            ? html`
                <label>Attach Image (PNG/JPEG for Multimodal SigLIP BBox Readout)</label>
                <input type="file" accept="image/*" @change=${this.onImageChange} />
              `
            : null}

          <details style="margin-top:14px;">
            <summary style="cursor:pointer; font-size:12px; color:var(--muted);">
              View / Edit Raw .json.tmpl Policy Source
            </summary>
            <textarea
              rows="10"
              style="margin-top:8px;"
              .value=${this.rawTemplate}
              @input=${(e: Event) => {
                this.rawTemplate = (e.target as HTMLTextAreaElement).value;
              }}
            ></textarea>
          </details>

          <button class="primary" ?disabled=${this.loading} @click=${this.executeDecision}>
            ${this.loading
              ? `⏳ Evaluating (${this.elapsedSec}s elapsed)...`
              : '⚡ Execute Joint Decision Readout'}
          </button>
        </div>

        <div class="card">
          <h2>2. Joint Slot Decisions & Epistemic Shannon Entropy (H)</h2>
          ${this.errorMsg
            ? html`<pre style="color:var(--fail);">${this.errorMsg}</pre>`
            : this.result
              ? this.renderDecisionResults(this.result)
              : html`
                  <p style="color:var(--muted); font-size:13px;">
                    Select a policy on the left and click
                    <b>Execute Joint Decision Readout</b>. If the Cloud Run RTX Pro 6000
                    GPU is scaled to zero, this gateway automatically wakes it and holds
                    the request until vLLM completes warmup.
                  </p>
                `}

          ${this.uploadedDataURI
            ? html`
                <div class="canvas-wrap">
                  <img src=${this.uploadedDataURI} alt="Uploaded preview" />
                  <svg viewBox="0 0 1000 1000" preserveAspectRatio="none">
                    ${this.renderBboxOverlay()}
                  </svg>
                </div>
              `
            : null}

          <h2 style="margin-top:22px;">3. Zero-CLI cURL & dgem Snippet</h2>
          <pre># Option A: Direct REST API call (No dgem CLI needed)
curl -s "${origin}/api/decide/${current?.id || 'support_triage'}" \\
  -H "Authorization: Bearer $(gcloud auth print-identity-token)" \\
  -H "Content-Type: application/json" \\
  -d '${curlBody}' | jq .

# Option B: Using dgem CLI against this gateway
./bin/dgem decide -u "${origin}/v1" --gcp-auth -t templates/${current?.id || 'support_triage'}.json.tmpl -s</pre>
        </div>
      </main>
    `;
  }

  private renderDecisionResults(res: GatewayDecideResponse) {
    const answers = res.answers || {};
    const qDiag = res.diagnostics?.questions || {};
    const keys = Object.keys(answers).sort();

    return html`
      <div
        style="display:flex; gap:14px; font-size:12px; color:var(--muted); margin-bottom:10px; flex-wrap:wrap;"
      >
        <span>⏱️ Latency: <b style="color:var(--text)">${res.wall_time_ms} ms</b></span>
        <span>
          📊 Max Entropy (H):
          <b style="color:var(--text)">${(res.max_entropy || 0).toFixed(4)} nats</b>
        </span>
        <span>
          🔄 Warmup Attempts: <b style="color:var(--text)">${res.warmup_attempts}</b>
        </span>
      </div>

      <table class="slot-table">
        <thead>
          <tr>
            <th>Slot</th>
            <th>Decision Value</th>
            <th>Confidence (P)</th>
            <th>Shannon Entropy (H)</th>
          </tr>
        </thead>
        <tbody>
          ${keys.map((k) => {
            const a = answers[k];
            const d = qDiag[k];
            const val =
              a.label !== undefined && a.label !== ''
                ? a.label
                : a.value !== undefined
                  ? a.value
                  : a.choice;
            const conf = `${((a.confidence || 0) * 100).toFixed(1)}%`;
            const ent =
              a.entropy !== undefined && a.entropy > 0
                ? a.entropy
                : d?.entropy !== undefined
                  ? d.entropy
                  : 0;
            const entClass =
              ent >= 0.35
                ? 'entropy-high'
                : ent >= 0.15
                  ? 'entropy-mid'
                  : 'entropy-low';

            return html`
              <tr>
                <td><b>${k}</b></td>
                <td style="color:var(--accent); font-weight:600;">
                  ${JSON.stringify(val)}
                </td>
                <td>${conf}</td>
                <td>
                  <span class="entropy-pill ${entClass}">${ent.toFixed(4)} nats</span>
                </td>
              </tr>
            `;
          })}
        </tbody>
      </table>
    `;
  }
}
