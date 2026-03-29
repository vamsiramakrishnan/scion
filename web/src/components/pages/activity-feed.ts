/**
 * Activity Feed — "Mission Control" view showing real-time agent activity
 * across all agents in a grove. Uses the SSE activity-feed endpoint.
 */

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';

import type { PageData } from '../../shared/types.js';
import '../shared/status-badge.js';

interface ActivityItem {
  id: string;
  timestamp: string;
  type: string;
  agentId?: string;
  agentName?: string;
  groveId?: string;
  phase?: string;
  activity?: string;
  toolName?: string;
  message?: string;
  subject: string;
}

@customElement('scion-page-activity-feed')
export class ScionPageActivityFeed extends LitElement {
  @property({ type: Object })
  pageData: PageData | null = null;

  @property({ type: String })
  groveId = '';

  @state()
  private items: ActivityItem[] = [];

  @state()
  private connected = false;

  @state()
  private error: string | null = null;

  @state()
  private paused = false;

  @state()
  private maxItems = 200;

  private eventSource: EventSource | null = null;

  static override styles = css`
    :host {
      display: block;
      padding: 1rem;
    }

    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 1rem;
    }

    .header h1 {
      font-size: 1.5rem;
      font-weight: 600;
      margin: 0;
    }

    .status-dot {
      display: inline-block;
      width: 8px;
      height: 8px;
      border-radius: 50%;
      margin-right: 0.5rem;
    }

    .status-dot.connected { background: #22c55e; }
    .status-dot.disconnected { background: #ef4444; }

    .feed {
      display: flex;
      flex-direction: column;
      gap: 0.25rem;
      font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
      font-size: 0.8125rem;
      line-height: 1.5;
    }

    .feed-item {
      display: flex;
      gap: 0.75rem;
      padding: 0.375rem 0.75rem;
      border-radius: var(--scion-radius-sm, 0.375rem);
      border-left: 3px solid transparent;
      transition: background 150ms ease;
    }

    .feed-item:hover {
      background: var(--scion-bg-subtle, #f8fafc);
    }

    .feed-item.agent_status { border-left-color: #3b82f6; }
    .feed-item.agent_created { border-left-color: #22c55e; }
    .feed-item.agent_deleted { border-left-color: #ef4444; }
    .feed-item.broker_status { border-left-color: #8b5cf6; }

    .time {
      color: var(--scion-text-muted, #64748b);
      white-space: nowrap;
      min-width: 5.5rem;
    }

    .icon {
      min-width: 1.25rem;
      text-align: center;
    }

    .agent-name {
      color: var(--scion-primary, #3b82f6);
      font-weight: 500;
      min-width: 10rem;
      max-width: 10rem;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .detail {
      color: var(--scion-text, #1e293b);
      flex: 1;
    }

    .phase {
      display: inline-block;
      padding: 0.0625rem 0.375rem;
      border-radius: 0.25rem;
      font-size: 0.6875rem;
      font-weight: 500;
      text-transform: uppercase;
      letter-spacing: 0.03em;
    }

    .phase.running { background: #dcfce7; color: #166534; }
    .phase.starting { background: #dbeafe; color: #1e40af; }
    .phase.stopping { background: #fef3c7; color: #92400e; }
    .phase.stopped { background: #f1f5f9; color: #475569; }
    .phase.error { background: #fee2e2; color: #991b1b; }

    .tool-name {
      color: #8b5cf6;
      font-weight: 500;
    }

    .empty-state {
      text-align: center;
      padding: 3rem;
      color: var(--scion-text-muted, #64748b);
    }

    .controls {
      display: flex;
      gap: 0.5rem;
      align-items: center;
    }
  `;

  override connectedCallback(): void {
    super.connectedCallback();
    if (this.groveId) {
      this.connect();
    }
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.disconnect();
  }

  private connect(): void {
    if (this.eventSource) {
      this.eventSource.close();
    }

    const url = `/api/v1/activity-feed?grove_id=${encodeURIComponent(this.groveId)}`;
    this.eventSource = new EventSource(url);

    this.eventSource.addEventListener('activity', (e: MessageEvent) => {
      if (this.paused) return;
      try {
        const item = JSON.parse(e.data) as ActivityItem;
        this.items = [item, ...this.items].slice(0, this.maxItems);
      } catch {
        // ignore parse errors
      }
    });

    this.eventSource.addEventListener('heartbeat', () => {
      this.connected = true;
    });

    this.eventSource.onopen = () => {
      this.connected = true;
      this.error = null;
    };

    this.eventSource.onerror = () => {
      this.connected = false;
      this.error = 'Connection lost — reconnecting...';
    };
  }

  private disconnect(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    this.connected = false;
  }

  private formatTime(ts: string): string {
    try {
      const d = new Date(ts);
      return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
    } catch {
      return '';
    }
  }

  private getIcon(type: string): string {
    switch (type) {
      case 'agent_status': return '\u{1F527}';
      case 'agent_created': return '\u{2728}';
      case 'agent_deleted': return '\u{1F5D1}';
      case 'broker_status': return '\u{1F4E1}';
      default: return '\u{25CF}';
    }
  }

  private getDetail(item: ActivityItem): string {
    if (item.type === 'agent_created') return 'created';
    if (item.type === 'agent_deleted') return 'deleted';
    if (item.toolName) return `executing ${item.toolName}`;
    if (item.activity) return item.activity;
    if (item.phase) return item.phase;
    if (item.message) return item.message;
    return item.type;
  }

  override render() {
    return html`
      <div class="header">
        <h1>
          <span class="status-dot ${this.connected ? 'connected' : 'disconnected'}"></span>
          Activity Feed
        </h1>
        <div class="controls">
          <sl-button size="small" variant="${this.paused ? 'primary' : 'default'}"
            @click=${() => { this.paused = !this.paused; }}>
            ${this.paused ? 'Resume' : 'Pause'}
          </sl-button>
          <sl-button size="small" variant="default"
            @click=${() => { this.items = []; }}>
            Clear
          </sl-button>
        </div>
      </div>

      ${this.error ? html`<div class="error-banner">${this.error}</div>` : nothing}

      ${this.items.length === 0
        ? html`<div class="empty-state">Waiting for agent activity...</div>`
        : html`
          <div class="feed">
            ${this.items.map(item => html`
              <div class="feed-item ${item.type}">
                <span class="time">${this.formatTime(item.timestamp)}</span>
                <span class="icon">${this.getIcon(item.type)}</span>
                <span class="agent-name">${item.agentName || item.agentId || '—'}</span>
                <span class="detail">
                  ${item.phase ? html`<span class="phase ${item.phase}">${item.phase}</span> ` : nothing}
                  ${item.toolName ? html`<span class="tool-name">${item.toolName}</span> ` : nothing}
                  ${this.getDetail(item)}
                </span>
              </div>
            `)}
          </div>
        `
      }
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'scion-page-activity-feed': ScionPageActivityFeed;
  }
}
