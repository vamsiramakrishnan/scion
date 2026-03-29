/**
 * Cost Dashboard — shows token usage and estimated costs per grove.
 * Fetches from GET /api/v1/groves/{groveId}/cost-summary
 */

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';

import type { PageData } from '../../shared/types.js';
import { apiFetch, extractApiError } from '../../client/api.js';

interface ModelCostBreak {
  model: string;
  costUsd: number;
  inputTokens: number;
  outputTokens: number;
  agentCount: number;
}

interface AgentCostBreak {
  agentId: string;
  agentName: string;
  model: string;
  costUsd: number;
  inputTokens: number;
  outputTokens: number;
  turns: number;
}

interface CostSummary {
  totalCostUsd: number;
  totalInputTokens: number;
  totalOutputTokens: number;
  totalTokens: number;
  agentCount: number;
  byModel: ModelCostBreak[];
  byAgent: AgentCostBreak[];
}

@customElement('scion-page-cost-dashboard')
export class ScionPageCostDashboard extends LitElement {
  @property({ type: Object })
  pageData: PageData | null = null;

  @property({ type: String })
  groveId = '';

  @state()
  private loading = true;

  @state()
  private summary: CostSummary | null = null;

  @state()
  private error: string | null = null;

  static override styles = css`
    :host {
      display: block;
      padding: 1rem;
    }

    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 1.5rem;
    }

    .header h1 {
      font-size: 1.5rem;
      font-weight: 600;
      margin: 0;
    }

    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
      gap: 1rem;
      margin-bottom: 2rem;
    }

    .stat-card {
      background: var(--scion-surface, #ffffff);
      border: 1px solid var(--scion-border, #e2e8f0);
      border-radius: var(--scion-radius-lg, 0.75rem);
      padding: 1.25rem;
    }

    .stat-label {
      font-size: 0.75rem;
      font-weight: 500;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--scion-text-muted, #64748b);
      margin-bottom: 0.25rem;
    }

    .stat-value {
      font-size: 1.75rem;
      font-weight: 700;
      color: var(--scion-text, #1e293b);
    }

    .stat-value.cost { color: #16a34a; }

    .section-title {
      font-size: 1.125rem;
      font-weight: 600;
      margin: 1.5rem 0 0.75rem 0;
    }

    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.875rem;
    }

    th {
      text-align: left;
      padding: 0.5rem 0.75rem;
      border-bottom: 2px solid var(--scion-border, #e2e8f0);
      font-weight: 600;
      color: var(--scion-text-muted, #64748b);
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    td {
      padding: 0.625rem 0.75rem;
      border-bottom: 1px solid var(--scion-border, #e2e8f0);
    }

    tr:hover { background: var(--scion-bg-subtle, #f8fafc); }

    .number { text-align: right; font-variant-numeric: tabular-nums; }
    .cost-cell { color: #16a34a; font-weight: 600; }

    .model-badge {
      display: inline-block;
      padding: 0.125rem 0.5rem;
      border-radius: 0.25rem;
      font-size: 0.75rem;
      font-weight: 500;
      background: #ede9fe;
      color: #6d28d9;
    }

    .loading-state, .error-state, .empty-state {
      text-align: center;
      padding: 3rem;
      color: var(--scion-text-muted, #64748b);
    }
  `;

  override connectedCallback(): void {
    super.connectedCallback();
    if (this.groveId) {
      void this.loadCosts();
    }
  }

  private async loadCosts(): Promise<void> {
    this.loading = true;
    this.error = null;
    try {
      const response = await apiFetch(`/api/v1/groves/${encodeURIComponent(this.groveId)}/cost-summary`);
      if (!response.ok) throw new Error(await extractApiError(response, 'Failed to load costs'));
      this.summary = await response.json() as CostSummary;
    } catch (err) {
      this.error = err instanceof Error ? err.message : 'Failed to load';
    } finally {
      this.loading = false;
    }
  }

  private formatTokens(n: number): string {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return n.toString();
  }

  private formatCost(n: number): string {
    if (n === 0) return '$0.00';
    if (n < 0.01) return `$${n.toFixed(4)}`;
    return `$${n.toFixed(2)}`;
  }

  override render() {
    return html`
      <div class="header">
        <h1>Cost Dashboard</h1>
        <sl-button size="small" @click=${() => this.loadCosts()}>
          <sl-icon slot="prefix" name="arrow-clockwise"></sl-icon>
          Refresh
        </sl-button>
      </div>

      ${this.loading ? html`<div class="loading-state"><sl-spinner></sl-spinner><p>Loading costs...</p></div>`
        : this.error ? html`<div class="error-state"><p>${this.error}</p><sl-button @click=${() => this.loadCosts()}>Retry</sl-button></div>`
        : this.summary ? this.renderDashboard()
        : html`<div class="empty-state">No cost data available yet.</div>`}
    `;
  }

  private renderDashboard() {
    const s = this.summary!;
    return html`
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-label">Total Cost</div>
          <div class="stat-value cost">${this.formatCost(s.totalCostUsd)}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Input Tokens</div>
          <div class="stat-value">${this.formatTokens(s.totalInputTokens)}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Output Tokens</div>
          <div class="stat-value">${this.formatTokens(s.totalOutputTokens)}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Agents</div>
          <div class="stat-value">${s.agentCount}</div>
        </div>
      </div>

      ${s.byModel && s.byModel.length > 0 ? html`
        <h2 class="section-title">By Model</h2>
        <table>
          <thead><tr>
            <th>Model</th>
            <th class="number">Cost</th>
            <th class="number">Input</th>
            <th class="number">Output</th>
            <th class="number">Agents</th>
          </tr></thead>
          <tbody>
            ${s.byModel.map(m => html`<tr>
              <td><span class="model-badge">${m.model || 'unknown'}</span></td>
              <td class="number cost-cell">${this.formatCost(m.costUsd)}</td>
              <td class="number">${this.formatTokens(m.inputTokens)}</td>
              <td class="number">${this.formatTokens(m.outputTokens)}</td>
              <td class="number">${m.agentCount}</td>
            </tr>`)}
          </tbody>
        </table>
      ` : nothing}

      ${s.byAgent && s.byAgent.length > 0 ? html`
        <h2 class="section-title">By Agent</h2>
        <table>
          <thead><tr>
            <th>Agent</th>
            <th>Model</th>
            <th class="number">Cost</th>
            <th class="number">Input</th>
            <th class="number">Output</th>
            <th class="number">Turns</th>
          </tr></thead>
          <tbody>
            ${s.byAgent.map(a => html`<tr>
              <td><a href="/agents/${a.agentId}">${a.agentName || a.agentId}</a></td>
              <td><span class="model-badge">${a.model || '—'}</span></td>
              <td class="number cost-cell">${this.formatCost(a.costUsd)}</td>
              <td class="number">${this.formatTokens(a.inputTokens)}</td>
              <td class="number">${this.formatTokens(a.outputTokens)}</td>
              <td class="number">${a.turns}</td>
            </tr>`)}
          </tbody>
        </table>
      ` : nothing}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'scion-page-cost-dashboard': ScionPageCostDashboard;
  }
}
