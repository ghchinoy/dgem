import { LitElement, html, css } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';

export interface BatchExpectedSlot {
  question: string;
  label: string;
  type: string;
  expected: string;
}

export interface BatchPresetItem {
  id: string;
  domain: string;
  tier: string;
  preview: string;
  template?: string;
  custom_template?: string;
  variables: Record<string, any>;
  expected_slots: BatchExpectedSlot[];
}

export interface BatchPresetSuite {
  id: string;
  title: string;
  category: string;
  badge: string;
  description: string;
  default_concurrency: number;
  total_items: number;
  total_questions: number;
  items: BatchPresetItem[];
}

interface BatchTableRow {
  rowKey: string;
  itemIndex: number; // 1-based item index
  slotIndex: number; // 1-based slot index within item
  totalSlotsForItem: number;
  item: BatchPresetItem;
  slot: BatchExpectedSlot;
  state: 'idle' | 'queued' | 'running' | 'pass' | 'miss' | 'error';
  predicted?: string;
  confidence?: number;
  entropy?: number;
  serverMs?: number;
  roundTripMs?: number;
  errorMsg?: string;
}

@customElement('dgem-batch-runner')
export class DgemBatchRunner extends LitElement {
  @property({ type: String, reflect: true }) resolvedTheme: 'light' | 'dark' = 'light';

  @state() private suites: BatchPresetSuite[] = [];
  @state() private selectedSuiteId = 'enterprise_multislot_25';
  @state() private concurrency = 4;
  @state() private rowFilter: 'all' | 'miss' | 'high_entropy' = 'all';
  @state() private rows: BatchTableRow[] = [];
  @state() private running = false;
  @state() private completedItems = 0;
  @state() private totalItems = 0;
  @state() private wallElapsedSec = 0;
  @state() private loadingPresets = true;

  private abortRun = false;
  private wallTimer?: number;
  private wallStartMs = 0;

  static styles = css`
    :host {
      display: block;
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
      color: var(--text-primary, #0f172a);

      --bg-surface: #ffffff;
      --bg-subtle: #f8fafc;
      --bg-muted: #f1f5f9;
      --border: #e2e8f0;
      --border-strong: #cbd5e1;
      --text-primary: #0f172a;
      --text-secondary: #475569;
      --text-muted: #64748b;
      --brand: #1447e6;
      --brand-soft: #eff6ff;
      --brand-border: #bfdbfe;
      --pass-fg: #15803d;
      --pass-bg: #dcfce7;
      --pass-border: #86efac;
      --miss-fg: #b91c1c;
      --miss-bg: #fee2e2;
      --miss-border: #fca5a5;
      --warn-fg: #b45309;
      --warn-bg: #fef3c7;
    }

    :host([resolvedTheme='dark']) {
      --bg-surface: #0f172a;
      --bg-subtle: #1e293b;
      --bg-muted: #0b1120;
      --border: #1e293b;
      --border-strong: #334155;
      --text-primary: #f8fafc;
      --text-secondary: #cbd5e1;
      --text-muted: #94a3b8;
      --brand: #3b82f6;
      --brand-soft: rgba(59, 130, 246, 0.15);
      --brand-border: rgba(59, 130, 246, 0.4);
      --pass-fg: #4ade80;
      --pass-bg: rgba(22, 163, 74, 0.2);
      --pass-border: rgba(74, 222, 128, 0.35);
      --miss-fg: #f87171;
      --miss-bg: rgba(220, 38, 38, 0.2);
      --miss-border: rgba(248, 113, 113, 0.35);
      --warn-fg: #fbbf24;
      --warn-bg: rgba(245, 158, 11, 0.2);
    }

    .batch-shell {
      display: flex;
      flex-direction: column;
      gap: 1.15rem;
    }

    .card {
      background: var(--bg-surface);
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 1.25rem 1.4rem;
      box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
    }

    .header-row {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 1rem;
      flex-wrap: wrap;
      margin-bottom: 1rem;
    }

    .title-group h2 {
      margin: 0;
      font-size: 1.15rem;
      font-weight: 700;
      font-family: 'Google Sans', 'Inter', sans-serif;
      color: var(--text-primary);
    }

    .title-group p {
      margin: 0.3rem 0 0;
      font-size: 0.84rem;
      color: var(--text-secondary);
      line-height: 1.45;
    }

    .suite-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(270px, 1fr));
      gap: 0.75rem;
      margin-bottom: 1.1rem;
    }

    .suite-card {
      text-align: left;
      background: var(--bg-subtle);
      border: 1px solid var(--border);
      border-radius: 11px;
      padding: 0.85rem 1rem;
      cursor: pointer;
      transition: all 0.14s ease;
      font-family: inherit;
      color: inherit;
    }

    .suite-card:hover {
      border-color: var(--brand);
    }

    .suite-card.active {
      background: var(--brand-soft);
      border-color: var(--brand);
      box-shadow: 0 0 0 1px var(--brand);
    }

    .suite-card-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 0.5rem;
      margin-bottom: 0.35rem;
    }

    .suite-cat {
      font-size: 0.68rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--brand);
    }

    .suite-badge {
      font-size: 0.68rem;
      font-weight: 600;
      font-family: 'JetBrains Mono', monospace;
      padding: 0.14rem 0.45rem;
      border-radius: 999px;
      background: var(--bg-surface);
      border: 1px solid var(--border);
      color: var(--text-secondary);
    }

    .suite-title {
      font-size: 0.88rem;
      font-weight: 700;
      color: var(--text-primary);
      margin-bottom: 0.25rem;
    }

    .suite-desc {
      font-size: 0.76rem;
      color: var(--text-muted);
      line-height: 1.38;
    }

    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 0.85rem;
      padding-top: 0.85rem;
      border-top: 1px solid var(--border);
    }

    .control-group {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      flex-wrap: wrap;
    }

    .control-label {
      font-size: 0.75rem;
      font-weight: 600;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.04em;
    }

    .seg-group {
      display: inline-flex;
      background: var(--bg-subtle);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 2px;
    }

    .seg-btn {
      border: none;
      background: transparent;
      color: var(--text-secondary);
      font-size: 0.76rem;
      font-weight: 600;
      padding: 0.34rem 0.65rem;
      border-radius: 6px;
      cursor: pointer;
      font-family: inherit;
    }

    .seg-btn.active {
      background: var(--bg-surface);
      color: var(--brand);
      box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
    }

    .run-btn {
      display: inline-flex;
      align-items: center;
      gap: 0.45rem;
      padding: 0.6rem 1.2rem;
      border-radius: 9px;
      border: none;
      background: var(--brand);
      color: #ffffff;
      font-size: 0.86rem;
      font-weight: 700;
      cursor: pointer;
      font-family: 'Google Sans', 'Inter', sans-serif;
      box-shadow: 0 1px 3px rgba(20, 71, 230, 0.3);
      transition: all 0.14s ease;
    }

    .run-btn:hover {
      filter: brightness(1.07);
    }

    .run-btn.stop {
      background: #dc2626;
    }

    /* Live Scoreboard Strip */
    .scoreboard {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
      gap: 0.75rem;
    }

    .metric-tile {
      background: var(--bg-surface);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 0.9rem 1.05rem;
      display: flex;
      flex-direction: column;
      gap: 0.25rem;
    }

    .metric-label {
      font-size: 0.71rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--text-muted);
    }

    .metric-val {
      font-size: 1.35rem;
      font-weight: 700;
      font-family: 'JetBrains Mono', monospace;
      color: var(--text-primary);
      display: flex;
      align-items: baseline;
      gap: 0.4rem;
    }

    .metric-sub {
      font-size: 0.74rem;
      font-weight: 500;
      color: var(--text-muted);
      font-family: 'Inter', sans-serif;
    }

    .progress-track {
      width: 100%;
      height: 5px;
      background: var(--bg-subtle);
      border-radius: 999px;
      overflow: hidden;
      margin-top: 0.25rem;
    }

    .progress-fill {
      height: 100%;
      background: var(--brand);
      transition: width 0.2s ease;
    }

    /* Results Table */
    .table-wrap {
      overflow-x: auto;
      border: 1px solid var(--border);
      border-radius: 12px;
      background: var(--bg-surface);
    }

    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.81rem;
    }

    thead th {
      text-align: left;
      padding: 0.7rem 0.85rem;
      font-size: 0.7rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--text-muted);
      background: var(--bg-subtle);
      border-bottom: 1px solid var(--border);
      white-space: nowrap;
    }

    tbody tr {
      border-bottom: 1px solid var(--border);
      transition: background 0.12s ease;
    }

    tbody tr:last-child {
      border-bottom: none;
    }

    tbody tr:hover {
      background: var(--bg-subtle);
    }

    tbody td {
      padding: 0.65rem 0.85rem;
      vertical-align: middle;
    }

    .col-idx {
      font-family: 'JetBrains Mono', monospace;
      font-size: 0.75rem;
      font-weight: 600;
      color: var(--text-muted);
      white-space: nowrap;
    }

    .state-pill {
      display: inline-flex;
      align-items: center;
      gap: 0.28rem;
      padding: 0.18rem 0.52rem;
      border-radius: 999px;
      font-size: 0.7rem;
      font-weight: 700;
      font-family: 'JetBrains Mono', monospace;
      white-space: nowrap;
    }

    .state-pill.idle,
    .state-pill.queued {
      background: var(--bg-subtle);
      color: var(--text-muted);
      border: 1px solid var(--border);
    }

    .state-pill.running {
      background: var(--brand-soft);
      color: var(--brand);
      border: 1px solid var(--brand-border);
      animation: pulse 1.2s infinite ease-in-out;
    }

    .state-pill.pass {
      background: var(--pass-bg);
      color: var(--pass-fg);
      border: 1px solid var(--pass-border);
    }

    .state-pill.miss,
    .state-pill.error {
      background: var(--miss-bg);
      color: var(--miss-fg);
      border: 1px solid var(--miss-border);
    }

    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.55; }
    }

    .q-cell {
      display: flex;
      flex-direction: column;
      gap: 0.18rem;
      max-width: 460px;
    }

    .q-head {
      display: flex;
      align-items: center;
      gap: 0.45rem;
      flex-wrap: wrap;
    }

    .q-slot {
      font-family: 'JetBrains Mono', monospace;
      font-weight: 700;
      font-size: 0.79rem;
      color: var(--text-primary);
    }

    .q-domain {
      font-size: 0.66rem;
      font-weight: 600;
      padding: 0.08rem 0.38rem;
      border-radius: 4px;
      background: var(--bg-subtle);
      border: 1px solid var(--border);
      color: var(--text-muted);
    }

    .q-preview {
      font-size: 0.75rem;
      color: var(--text-secondary);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .type-pill {
      font-family: 'JetBrains Mono', monospace;
      font-size: 0.7rem;
      font-weight: 600;
      padding: 0.14rem 0.45rem;
      border-radius: 5px;
      background: var(--bg-subtle);
      border: 1px solid var(--border);
      color: var(--text-secondary);
    }

    .val-pill {
      font-family: 'JetBrains Mono', monospace;
      font-size: 0.75rem;
      font-weight: 700;
      padding: 0.18rem 0.5rem;
      border-radius: 6px;
      display: inline-block;
    }

    .val-pill.match {
      background: var(--pass-bg);
      color: var(--pass-fg);
    }

    .val-pill.mismatch {
      background: var(--miss-bg);
      color: var(--miss-fg);
    }

    .val-pill.expected {
      background: var(--bg-subtle);
      color: var(--text-primary);
      border: 1px solid var(--border);
    }

    .conf-cell {
      display: flex;
      flex-direction: column;
      gap: 0.22rem;
      min-width: 115px;
    }

    .conf-top {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      font-family: 'JetBrains Mono', monospace;
      font-size: 0.75rem;
    }

    .conf-pct {
      font-weight: 700;
      color: var(--text-primary);
    }

    .conf-ent {
      font-size: 0.67rem;
      color: var(--text-muted);
    }

    .conf-bar {
      width: 100%;
      height: 4px;
      background: var(--bg-subtle);
      border-radius: 999px;
      overflow: hidden;
    }

    .conf-fill {
      height: 100%;
      border-radius: 999px;
    }

    .ms-cell {
      font-family: 'JetBrains Mono', monospace;
      white-space: nowrap;
    }

    .ms-primary {
      font-weight: 700;
      font-size: 0.78rem;
      color: var(--text-primary);
    }

    .ms-sub {
      font-size: 0.67rem;
      color: var(--text-muted);
    }

    .inspect-btn {
      border: 1px solid var(--border);
      background: var(--bg-subtle);
      color: var(--text-secondary);
      border-radius: 6px;
      padding: 0.22rem 0.5rem;
      font-size: 0.7rem;
      font-weight: 600;
      cursor: pointer;
      font-family: inherit;
      white-space: nowrap;
    }

    .inspect-btn:hover {
      border-color: var(--brand);
      color: var(--brand);
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this.loadBatchPresets();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    if (this.wallTimer) {
      window.clearInterval(this.wallTimer);
    }
  }

  private async loadBatchPresets() {
    this.loadingPresets = true;
    try {
      const res = await fetch('/api/batch/presets');
      if (res.ok) {
        const data = await res.json();
        this.suites = data.suites || [];
        if (this.suites.length > 0) {
          this.selectSuite(this.suites[0].id);
        }
      }
    } catch (e) {
      console.error('Failed to load batch presets:', e);
    } finally {
      this.loadingPresets = false;
    }
  }

  private selectSuite(suiteId: string) {
    if (this.running) return;
    this.selectedSuiteId = suiteId;
    const suite = this.suites.find((s) => s.id === suiteId);
    if (!suite) return;
    this.concurrency = suite.default_concurrency || 4;
    this.totalItems = suite.items.length;
    this.completedItems = 0;
    this.wallElapsedSec = 0;

    const newRows: BatchTableRow[] = [];
    suite.items.forEach((item, idx) => {
      const slots = item.expected_slots || [];
      slots.forEach((slot, sIdx) => {
        newRows.push({
          rowKey: `${item.id}:${slot.question}`,
          itemIndex: idx + 1,
          slotIndex: sIdx + 1,
          totalSlotsForItem: slots.length,
          item,
          slot,
          state: 'idle',
        });
      });
    });
    this.rows = newRows;
  }

  private normalizeAnswer(val: any): string {
    if (val === undefined || val === null) return '';
    const s = String(val).trim().toLowerCase();
    if (s === 'true') return 'yes';
    if (s === 'false') return 'no';
    return String(val).trim();
  }

  private isMatch(predicted: string | undefined, expected: string): boolean {
    if (!predicted) return false;
    const p = this.normalizeAnswer(predicted).toLowerCase();
    const e = this.normalizeAnswer(expected).toLowerCase();
    return p === e;
  }

  private median(values: number[]): number {
    if (values.length === 0) return 0;
    const sorted = [...values].sort((a, b) => a - b);
    const mid = Math.floor(sorted.length / 2);
    return sorted.length % 2 !== 0
      ? sorted[mid]
      : Math.round((sorted[mid - 1] + sorted[mid]) / 2);
  }

  private async runBatch() {
    const suite = this.suites.find((s) => s.id === this.selectedSuiteId);
    if (!suite || this.running) return;

    this.abortRun = false;
    this.running = true;
    this.completedItems = 0;
    this.totalItems = suite.items.length;
    this.wallStartMs = performance.now();
    this.wallElapsedSec = 0;

    // Reset all rows to 'queued'
    this.rows = this.rows.map((r) => ({
      ...r,
      state: 'queued',
      predicted: undefined,
      confidence: undefined,
      entropy: undefined,
      serverMs: undefined,
      roundTripMs: undefined,
      errorMsg: undefined,
    }));

    this.dispatchEvent(new CustomEvent('batch-started', { bubbles: true, composed: true }));

    if (this.wallTimer) window.clearInterval(this.wallTimer);
    this.wallTimer = window.setInterval(() => {
      this.wallElapsedSec = Number(((performance.now() - this.wallStartMs) / 1000).toFixed(1));
    }, 100);

    const queue = suite.items.map((item, idx) => ({ item, itemIndex: idx + 1 }));
    let cursor = 0;

    const worker = async () => {
      while (!this.abortRun && cursor < queue.length) {
        const currentIdx = cursor++;
        const { item } = queue[currentIdx];

        // Mark this item's rows as 'running'
        this.rows = this.rows.map((r) =>
          r.item.id === item.id ? { ...r, state: 'running' } : r
        );

        const t0 = performance.now();
        try {
          const body: Record<string, any> = {
            variables: item.variables || {},
          };
          if (item.custom_template) {
            body.custom_template = item.custom_template;
          } else {
            body.template = item.template || 'support_triage';
          }

          const res = await fetch('/api/decide', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-DGem-Surface': 'web_studio_batch',
            },
            body: JSON.stringify(body),
          });
          const rttMs = Math.round(performance.now() - t0);

          if (!res.ok) {
            const errText = await res.text();
            this.rows = this.rows.map((r) =>
              r.item.id === item.id
                ? { ...r, state: 'error', roundTripMs: rttMs, errorMsg: `HTTP ${res.status}: ${errText}` }
                : r
            );
          } else {
            const payload = await res.json();
            const answers = payload.answers || {};
            const timing = payload.diagnostics?.timing || {};
            const serverMs =
              timing.total_ms ||
              timing.denoise_ms ||
              payload.gpu_forward_ms ||
              payload.diagnostics?.server_denoise_ms ||
              rttMs;
            const qDiags = payload.diagnostics?.questions || {};

            this.rows = this.rows.map((r) => {
              if (r.item.id !== item.id) return r;
              const slotAns = answers[r.slot.question];
              const rawVal =
                slotAns?.value !== undefined
                  ? slotAns.value
                  : slotAns?.choice !== undefined && slotAns?.choice !== ''
                    ? slotAns.choice
                    : slotAns?.level !== undefined && slotAns?.level !== ''
                      ? slotAns.level
                      : slotAns?.label;
              const predStr = this.normalizeAnswer(rawVal);
              const matched = this.isMatch(predStr, r.slot.expected);
              const conf = typeof slotAns?.confidence === 'number' ? slotAns.confidence : 0;
              const qDiag = qDiags[r.slot.question];
              let ent =
                typeof slotAns?.entropy === 'number' && slotAns.entropy > 0
                  ? slotAns.entropy
                  : typeof qDiag?.entropy === 'number' && qDiag.entropy > 0
                    ? qDiag.entropy
                    : 0;
              if (ent === 0 && slotAns?.probabilities && typeof slotAns.probabilities === 'object') {
                for (const p of Object.values(slotAns.probabilities)) {
                  const prob = Number(p);
                  if (prob > 1e-12) {
                    ent -= prob * Math.log(prob);
                  }
                }
              }

              return {
                ...r,
                state: matched ? 'pass' : 'miss',
                predicted: predStr || '—',
                confidence: conf,
                entropy: ent,
                serverMs: Math.round(serverMs),
                roundTripMs: rttMs,
              };
            });
          }
        } catch (err: any) {
          const rttMs = Math.round(performance.now() - t0);
          this.rows = this.rows.map((r) =>
            r.item.id === item.id
              ? { ...r, state: 'error', roundTripMs: rttMs, errorMsg: err?.message || 'Network error' }
              : r
          );
        } finally {
          this.completedItems += 1;
        }
      }
    };

    const workers = Array.from({ length: Math.min(this.concurrency, queue.length) }, () => worker());
    await Promise.all(workers);

    if (this.wallTimer) {
      window.clearInterval(this.wallTimer);
      this.wallTimer = undefined;
    }
    this.wallElapsedSec = Number(((performance.now() - this.wallStartMs) / 1000).toFixed(1));
    this.running = false;
    this.dispatchEvent(new CustomEvent('batch-completed', { bubbles: true, composed: true }));
  }

  private stopBatch() {
    this.abortRun = true;
    this.running = false;
    if (this.wallTimer) {
      window.clearInterval(this.wallTimer);
      this.wallTimer = undefined;
    }
  }

  private openRowInStudio(row: BatchTableRow) {
    this.dispatchEvent(
      new CustomEvent('inspect-batch-item', {
        detail: {
          template: row.item.template,
          customTemplate: row.item.custom_template,
          variables: row.item.variables,
          preview: row.item.preview,
        },
        bubbles: true,
        composed: true,
      })
    );
  }

  render() {
    const completedRows = this.rows.filter((r) => r.state === 'pass' || r.state === 'miss');
    const passRows = this.rows.filter((r) => r.state === 'pass');
    const accPct =
      completedRows.length > 0 ? ((passRows.length / completedRows.length) * 100).toFixed(1) : '—';

    // Collect per-item latencies (1 per itemIndex)
    const seenItems = new Set<number>();
    const serverLatencies: number[] = [];
    const rttLatencies: number[] = [];
    for (const r of completedRows) {
      if (!seenItems.has(r.itemIndex)) {
        seenItems.add(r.itemIndex);
        if (r.serverMs !== undefined) serverLatencies.push(r.serverMs);
        if (r.roundTripMs !== undefined) rttLatencies.push(r.roundTripMs);
      }
    }
    const serverP50 = this.median(serverLatencies);
    const rttP50 = this.median(rttLatencies);
    const pctDone = this.totalItems > 0 ? Math.round((this.completedItems / this.totalItems) * 100) : 0;

    const visibleRows = this.rows.filter((r) => {
      if (this.rowFilter === 'miss') return r.state === 'miss' || r.state === 'error';
      if (this.rowFilter === 'high_entropy') return (r.entropy || 0) >= 0.25;
      return true;
    });

    return html`
      <div class="batch-shell">
        <!-- Top Suite Selector & Execution Controls Card -->
        <div class="card">
          <div class="header-row">
            <div class="title-group">
              <h2>Batch Decision Evaluation & Live Streaming Telemetry</h2>
              <p>
                Select a ground-truth challenge suite below and run concurrent single-pass DiffusionGemma evaluations with live accuracy, confidence, and latency metrics.
              </p>
            </div>
            <div>
              ${this.running
                ? html`
                    <button class="run-btn stop" @click=${this.stopBatch}>
                      <span>■ Stop Batch (${this.completedItems}/${this.totalItems})</span>
                    </button>
                  `
                : html`
                    <button class="run-btn" @click=${this.runBatch}>
                      <span>▶ Run Batch (${this.totalItems} items)</span>
                    </button>
                  `}
            </div>
          </div>

          <div class="suite-grid">
            ${this.suites.map(
              (s) => html`
                <button
                  class="suite-card ${this.selectedSuiteId === s.id ? 'active' : ''}"
                  @click=${() => this.selectSuite(s.id)}
                >
                  <div class="suite-card-top">
                    <span class="suite-cat">${s.category}</span>
                    <span class="suite-badge">${s.badge}</span>
                  </div>
                  <div class="suite-title">${s.title}</div>
                  <div class="suite-desc">${s.description}</div>
                </button>
              `
            )}
          </div>

          <div class="toolbar">
            <div class="control-group">
              <span class="control-label">Concurrency</span>
              <div class="seg-group">
                ${[1, 4, 8].map(
                  (c) => html`
                    <button
                      class="seg-btn ${this.concurrency === c ? 'active' : ''}"
                      @click=${() => {
                        if (!this.running) this.concurrency = c;
                      }}
                    >
                      ${c}x Workers
                    </button>
                  `
                )}
              </div>
            </div>

            <div class="control-group">
              <span class="control-label">Filter Rows</span>
              <div class="seg-group">
                <button
                  class="seg-btn ${this.rowFilter === 'all' ? 'active' : ''}"
                  @click=${() => (this.rowFilter = 'all')}
                >
                  All (${this.rows.length})
                </button>
                <button
                  class="seg-btn ${this.rowFilter === 'miss' ? 'active' : ''}"
                  @click=${() => (this.rowFilter = 'miss')}
                >
                  ✗ Misses (${this.rows.filter((r) => r.state === 'miss' || r.state === 'error').length})
                </button>
                <button
                  class="seg-btn ${this.rowFilter === 'high_entropy' ? 'active' : ''}"
                  @click=${() => (this.rowFilter = 'high_entropy')}
                >
                  High H ≥ 0.25 (${this.rows.filter((r) => (r.entropy || 0) >= 0.25).length})
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Live Scoreboard Strip -->
        <div class="scoreboard">
          <div class="metric-tile">
            <span class="metric-label">Items Evaluated</span>
            <div class="metric-val">
              <span>${this.completedItems}/${this.totalItems}</span>
              <span class="metric-sub">items</span>
            </div>
            <div class="progress-track">
              <div class="progress-fill" style="width: ${pctDone}%"></div>
            </div>
          </div>

          <div class="metric-tile">
            <span class="metric-label">Accuracy</span>
            <div class="metric-val">
              <span>${accPct}${accPct !== '—' ? '%' : ''}</span>
              <span class="metric-sub">(${passRows.length}/${completedRows.length || this.rows.length} questions)</span>
            </div>
          </div>

          <div class="metric-tile">
            <span class="metric-label">Server p50</span>
            <div class="metric-val">
              <span>${serverP50 > 0 ? `${serverP50} ms` : '—'}</span>
              <span class="metric-sub">GPU forward</span>
            </div>
          </div>

          <div class="metric-tile">
            <span class="metric-label">Round Trip p50</span>
            <div class="metric-val">
              <span>${rttP50 > 0 ? `${rttP50} ms` : '—'}</span>
              <span class="metric-sub">end-to-end HTTP</span>
            </div>
          </div>

          <div class="metric-tile">
            <span class="metric-label">Wall Time</span>
            <div class="metric-val">
              <span>${this.wallElapsedSec > 0 ? `${this.wallElapsedSec} s` : '0.0 s'}</span>
              <span class="metric-sub">
                ${this.wallElapsedSec > 0 && this.completedItems > 0
                  ? `${(this.completedItems / this.wallElapsedSec).toFixed(1)} items/s`
                  : `${this.concurrency}x parallel`}
              </span>
            </div>
          </div>
        </div>

        <!-- Live Results Table -->
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>#</th>
                <th>State</th>
                <th>Question</th>
                <th>Type</th>
                <th>Predicted</th>
                <th>Expected</th>
                <th>p / conf</th>
                <th>Server ms</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              ${visibleRows.map((r) => {
                const rowIdxLabel =
                  r.totalSlotsForItem > 1 ? `${r.itemIndex}.${r.slotIndex}` : `${r.itemIndex}`;
                const stateLabel =
                  r.state === 'pass'
                    ? '✓ PASS'
                    : r.state === 'miss'
                      ? '✗ MISS'
                      : r.state === 'running'
                        ? '⚡ RUNNING'
                        : r.state === 'queued'
                          ? '⏳ QUEUED'
                          : r.state === 'error'
                            ? '⚠ ERROR'
                            : 'READY';
                const confPct =
                  r.confidence !== undefined ? `${(r.confidence * 100).toFixed(1)}%` : '—';
                const confWidth = r.confidence !== undefined ? Math.round(r.confidence * 100) : 0;
                const confColor =
                  r.state === 'miss'
                    ? '#ef4444'
                    : (r.confidence || 0) >= 0.85
                      ? '#16a34a'
                      : '#f59e0b';

                return html`
                  <tr>
                    <td class="col-idx">${rowIdxLabel}</td>
                    <td>
                      <span class="state-pill ${r.state}">${stateLabel}</span>
                    </td>
                    <td>
                      <div class="q-cell" title="${r.item.preview}">
                        <div class="q-head">
                          <span class="q-slot">${r.slot.question}</span>
                          <span class="q-domain">${r.item.domain} · ${r.item.tier}</span>
                        </div>
                        <div class="q-preview">${r.item.preview}</div>
                      </div>
                    </td>
                    <td>
                      <span class="type-pill">${r.slot.type}</span>
                    </td>
                    <td>
                      ${r.predicted !== undefined
                        ? html`
                            <span class="val-pill ${r.state === 'pass' ? 'match' : 'mismatch'}">
                              ${r.predicted}
                            </span>
                          `
                        : html`<span style="color: var(--text-muted)">—</span>`}
                    </td>
                    <td>
                      <span class="val-pill expected">${r.slot.expected}</span>
                    </td>
                    <td>
                      <div class="conf-cell">
                        <div class="conf-top">
                          <span class="conf-pct">${confPct}</span>
                          ${r.entropy !== undefined
                            ? html`<span class="conf-ent">H=${r.entropy.toFixed(2)}</span>`
                            : ''}
                        </div>
                        <div class="conf-bar">
                          <div
                            class="conf-fill"
                            style="width: ${confWidth}%; background: ${confColor};"
                          ></div>
                        </div>
                      </div>
                    </td>
                    <td class="ms-cell">
                      ${r.serverMs !== undefined
                        ? html`
                            <div class="ms-primary">${r.serverMs} ms</div>
                            <div class="ms-sub">RTT ${r.roundTripMs} ms</div>
                          `
                        : html`<span style="color: var(--text-muted)">—</span>`}
                    </td>
                    <td>
                      <button
                        class="inspect-btn"
                        title="Open this item in the Decision Studio"
                        @click=${() => this.openRowInStudio(r)}
                      >
                        Studio ↗
                      </button>
                    </td>
                  </tr>
                `;
              })}
            </tbody>
          </table>
        </div>
      </div>
    `;
  }
}
