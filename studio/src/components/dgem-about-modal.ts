import { LitElement, html, css } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import type { GPUHealthStatus } from '../types.js';

@customElement('dgem-about-modal')
export class DgemAboutModal extends LitElement {
  @property({ type: Boolean, reflect: true }) open = false;
  @property({ type: String, reflect: true }) resolvedTheme: 'light' | 'dark' = 'light';
  @property({ type: Object }) gpuStatus: GPUHealthStatus | null = null;
  @property({ type: Number }) templateCount = 26;

  static styles = css`
    :host {
      display: none;
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;

      --modal-bg: #ffffff;
      --modal-surface: #f8fafc;
      --modal-border: #e2e8f0;
      --modal-heading: #0f172a;
      --modal-body: #334155;
      --modal-muted: #64748b;
      --modal-brand: #1447e6;
      --modal-brand-soft: #eff6ff;
      --modal-brand-border: #bfdbfe;
    }

    :host([open]) {
      display: block;
    }

    :host([resolvedTheme='dark']) {
      --modal-bg: #0f172a;
      --modal-surface: #1e293b;
      --modal-border: #334155;
      --modal-heading: #f8fafc;
      --modal-body: #cbd5e1;
      --modal-muted: #94a3b8;
      --modal-brand: #3b82f6;
      --modal-brand-soft: rgba(59, 130, 246, 0.15);
      --modal-brand-border: rgba(59, 130, 246, 0.35);
    }

    .material-symbols-outlined {
      font-family: 'Material Symbols Outlined';
      font-weight: normal;
      font-style: normal;
      font-size: 19px;
      line-height: 1;
      display: inline-flex;
      align-items: center;
      justify-content: center;
    }

    .backdrop {
      position: fixed;
      inset: 0;
      z-index: 100;
      background: rgba(15, 23, 42, 0.55);
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 1.25rem;
    }

    .dialog {
      width: 100%;
      max-width: 620px;
      background: var(--modal-bg);
      color: var(--modal-body);
      border: 1px solid var(--modal-border);
      border-radius: 12px;
      box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.25);
      overflow: hidden;
    }

    .dialog-header {
      padding: 1rem 1.25rem;
      border-bottom: 1px solid var(--modal-border);
      display: flex;
      align-items: center;
      justify-content: space-between;
      background: var(--modal-surface);
    }

    .dialog-title {
      margin: 0;
      font-family: 'Google Sans', sans-serif;
      font-size: 1rem;
      font-weight: 700;
      color: var(--modal-heading);
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .close-btn {
      border: 1px solid var(--modal-border);
      background: var(--modal-bg);
      color: var(--modal-muted);
      width: 30px;
      height: 30px;
      border-radius: 6px;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      justify-content: center;
    }

    .close-btn:hover {
      color: var(--modal-heading);
    }

    .dialog-body {
      padding: 1.25rem;
      font-size: 0.83rem;
      line-height: 1.55;
    }

    .spec-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 0.65rem;
      margin: 1rem 0;
    }

    .spec-item {
      padding: 0.7rem 0.85rem;
      border-radius: 8px;
      border: 1px solid var(--modal-border);
      background: var(--modal-surface);
    }

    .spec-label {
      font-size: 0.68rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      color: var(--modal-muted);
      margin-bottom: 0.2rem;
    }

    .spec-val {
      font-family: 'JetBrains Mono', monospace;
      font-variant-numeric: tabular-nums;
      font-size: 0.78rem;
      font-weight: 600;
      color: var(--modal-heading);
      word-break: break-all;
    }

    .pillar-list {
      margin: 0.75rem 0 0;
      padding-left: 1.15rem;
      color: var(--modal-body);
    }

    .pillar-list li {
      margin-bottom: 0.35rem;
    }
  `;

  private closeModal() {
    this.dispatchEvent(
      new CustomEvent('close-about', {
        bubbles: true,
        composed: true,
      })
    );
  }

  render() {
    if (!this.open) return null;

    return html`
      <div class="backdrop" @click=${this.closeModal}>
        <div
          class="dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="about-title"
          @click=${(e: Event) => e.stopPropagation()}
        >
          <div class="dialog-header">
            <h2 class="dialog-title" id="about-title">
              <span class="material-symbols-outlined" style="color:var(--modal-brand)">info</span>
              About DiffusionGemma Decision Studio (dgem)
            </h2>
            <button class="close-btn" @click=${this.closeModal} title="Close">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>

          <div class="dialog-body">
            <p style="margin-top:0">
              <strong>DiffusionGemma (<code>dgemma</code>)</strong> is a Zero-Shot Decision Model that
              evaluates structured multi-slot policies (<code>.json.tmpl</code>) jointly in
              <strong>O(1) forward passes</strong> on a bidirectional discrete-diffusion canvas,
              returning calibrated slot probabilities and epistemic
              <strong>Shannon entropy (H)</strong> without conversational token overhead.
            </p>

            <div class="spec-grid">
              <div class="spec-item">
                <div class="spec-label">Inference Engine &amp; Vision</div>
                <div class="spec-val">vLLM (TRITON_ATTN) + Gemma 4 SigLIP</div>
              </div>
              <div class="spec-item">
                <div class="spec-label">Cloud Run GPU Hardware</div>
                <div class="spec-val">
                  ${this.gpuStatus?.gpu_hardware || '1× NVIDIA RTX Pro 6000 · 48GB VRAM'}
                </div>
              </div>
              <div class="spec-item">
                <div class="spec-label">Model Checkpoint</div>
                <div class="spec-val">nvidia/diffusiongemma-26B-A4B-it-NVFP4</div>
              </div>
              <div class="spec-item">
                <div class="spec-label">Embedded Policy Templates</div>
                <div class="spec-val">${this.templateCount} (.json.tmpl policies)</div>
              </div>
              <div class="spec-item">
                <div class="spec-label">Upstream GPU Endpoint</div>
                <div class="spec-val">${this.gpuStatus?.upstream_url || '/v1'}</div>
              </div>
              <div class="spec-item">
                <div class="spec-label">Observability &amp; Tracing</div>
                <div class="spec-val">OpenTelemetry + Google Cloud Trace</div>
              </div>
            </div>

            <div style="font-weight:600;color:var(--modal-heading);font-size:0.8rem">
              The 4-Pillar Package:
            </div>
            <ul class="pillar-list">
              <li>
                <strong>Lit WebComponents Decision Studio</strong>: Interactive single-pass policy
                evaluator, multimodal <code>EXP-09</code> Softmax Expectation bounding-box canvas, and
                <code>EXP-05</code> Entropy Cascade simulator.
              </li>
              <li>
                <strong>HTTP REST &amp; OpenAI Gateway</strong>: Idempotent single-flight cold-start
                wakeup coordinator (<code>/api/warmup</code>, <code>/api/status</code>,
                <code>/api/decide/{template}</code>).
              </li>
              <li>
                <strong>Model Context Protocol (MCP) Server</strong>: 6 MCP tools over Streamable HTTP
                (<code>POST /mcp</code>) and stdio (<code>dgem mcp</code>).
              </li>
              <li>
                <strong><code>dgem</code> CLI &amp; Benchmark Suites</strong>: Direct
                <code>--gcp-auth</code> CLI execution and reproducible evaluation suites.
              </li>
            </ul>
          </div>
        </div>
      </div>
    `;
  }
}
