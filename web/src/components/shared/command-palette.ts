/**
 * Command Palette — Cmd+K / Ctrl+K quick navigation and actions.
 *
 * Provides fuzzy search across agents, groves, and navigation.
 * Follows the same pattern as VS Code / Linear command palettes.
 */

import { LitElement, html, css, nothing } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { stateManager } from '../../client/state.js';

interface CommandItem {
  id: string;
  label: string;
  description?: string;
  category: 'navigation' | 'agent' | 'grove' | 'action';
  icon: string;
  action: () => void;
}

@customElement('scion-command-palette')
export class ScionCommandPalette extends LitElement {
  @state()
  private open = false;

  @state()
  private query = '';

  @state()
  private selectedIndex = 0;

  @state()
  private items: CommandItem[] = [];

  private boundKeyDown = this.handleGlobalKeyDown.bind(this);

  static override styles = css`
    .overlay {
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.5);
      backdrop-filter: blur(4px);
      z-index: 9999;
      display: flex;
      justify-content: center;
      padding-top: 20vh;
    }

    .palette {
      width: min(36rem, 90vw);
      max-height: 24rem;
      background: var(--scion-surface, #ffffff);
      border: 1px solid var(--scion-border, #e2e8f0);
      border-radius: var(--scion-radius-lg, 0.75rem);
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }

    .search-input {
      width: 100%;
      padding: 0.875rem 1rem;
      border: none;
      border-bottom: 1px solid var(--scion-border, #e2e8f0);
      font-size: 1rem;
      font-family: inherit;
      background: transparent;
      color: var(--scion-text, #1e293b);
      outline: none;
    }

    .search-input::placeholder {
      color: var(--scion-text-muted, #94a3b8);
    }

    .results {
      overflow-y: auto;
      flex: 1;
    }

    .result-item {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.625rem 1rem;
      cursor: pointer;
      transition: background 100ms ease;
    }

    .result-item:hover,
    .result-item.selected {
      background: var(--scion-bg-subtle, #f1f5f9);
    }

    .result-item.selected {
      background: #eff6ff;
    }

    .result-icon {
      font-size: 1rem;
      min-width: 1.5rem;
      text-align: center;
      color: var(--scion-text-muted, #64748b);
    }

    .result-label {
      font-weight: 500;
      color: var(--scion-text, #1e293b);
    }

    .result-desc {
      font-size: 0.8125rem;
      color: var(--scion-text-muted, #64748b);
      margin-left: auto;
    }

    .category-header {
      padding: 0.375rem 1rem;
      font-size: 0.6875rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--scion-text-muted, #94a3b8);
      background: var(--scion-bg-subtle, #f8fafc);
    }

    .shortcut-hint {
      padding: 0.5rem 1rem;
      font-size: 0.75rem;
      color: var(--scion-text-muted, #94a3b8);
      border-top: 1px solid var(--scion-border, #e2e8f0);
      display: flex;
      gap: 1rem;
    }

    kbd {
      display: inline-block;
      padding: 0.0625rem 0.25rem;
      font-size: 0.6875rem;
      border: 1px solid var(--scion-border, #e2e8f0);
      border-radius: 0.25rem;
      background: var(--scion-bg-subtle, #f8fafc);
      font-family: inherit;
    }

    .no-results {
      padding: 2rem;
      text-align: center;
      color: var(--scion-text-muted, #94a3b8);
    }
  `;

  override connectedCallback(): void {
    super.connectedCallback();
    document.addEventListener('keydown', this.boundKeyDown);
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    document.removeEventListener('keydown', this.boundKeyDown);
  }

  private handleGlobalKeyDown(e: KeyboardEvent): void {
    // Cmd+K or Ctrl+K to open
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault();
      this.toggle();
      return;
    }

    if (!this.open) return;

    // Escape to close
    if (e.key === 'Escape') {
      e.preventDefault();
      this.close();
      return;
    }

    // Arrow navigation
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      this.selectedIndex = Math.min(this.selectedIndex + 1, this.items.length - 1);
      return;
    }

    if (e.key === 'ArrowUp') {
      e.preventDefault();
      this.selectedIndex = Math.max(this.selectedIndex - 1, 0);
      return;
    }

    // Enter to execute
    if (e.key === 'Enter') {
      e.preventDefault();
      if (this.items[this.selectedIndex]) {
        this.items[this.selectedIndex].action();
        this.close();
      }
      return;
    }
  }

  private toggle(): void {
    if (this.open) {
      this.close();
    } else {
      this.openPalette();
    }
  }

  private openPalette(): void {
    this.open = true;
    this.query = '';
    this.selectedIndex = 0;
    this.buildItems();
    this.updateComplete.then(() => {
      const input = this.shadowRoot?.querySelector('.search-input') as HTMLInputElement | null;
      input?.focus();
    });
  }

  private close(): void {
    this.open = false;
    this.query = '';
    this.items = [];
  }

  private handleInput(e: Event): void {
    this.query = (e.target as HTMLInputElement).value;
    this.selectedIndex = 0;
    this.buildItems();
  }

  private buildItems(): void {
    const all: CommandItem[] = [];

    // Navigation commands
    all.push(
      { id: 'nav-home', label: 'Home', category: 'navigation', icon: '\u{1F3E0}', action: () => { window.location.href = '/'; } },
      { id: 'nav-agents', label: 'All Agents', category: 'navigation', icon: '\u{1F916}', action: () => { window.location.href = '/agents'; } },
      { id: 'nav-groves', label: 'All Groves', category: 'navigation', icon: '\u{1F333}', action: () => { window.location.href = '/groves'; } },
      { id: 'nav-settings', label: 'Settings', category: 'navigation', icon: '\u{2699}', action: () => { window.location.href = '/settings'; } },
    );

    // Agents from state manager
    const agents = stateManager.getAgents();
    for (const agent of agents.slice(0, 20)) {
      all.push({
        id: `agent-${agent.id}`,
        label: agent.name || agent.id,
        description: agent.phase || '',
        category: 'agent',
        icon: '\u{1F916}',
        action: () => { window.location.href = `/agents/${agent.id}`; },
      });
    }

    // Groves from state manager
    const groves = stateManager.getGroves();
    for (const grove of groves.slice(0, 10)) {
      all.push({
        id: `grove-${grove.id}`,
        label: grove.name || grove.id,
        description: `${grove.agentCount || 0} agents`,
        category: 'grove',
        icon: '\u{1F333}',
        action: () => { window.location.href = `/groves/${grove.id}`; },
      });
    }

    // Filter by query
    if (this.query) {
      const q = this.query.toLowerCase();
      this.items = all.filter(item =>
        item.label.toLowerCase().includes(q) ||
        (item.description && item.description.toLowerCase().includes(q))
      );
    } else {
      this.items = all;
    }
  }

  override render() {
    if (!this.open) return nothing;

    return html`
      <div class="overlay" @click=${(e: Event) => { if (e.target === e.currentTarget) this.close(); }}>
        <div class="palette">
          <input class="search-input"
            type="text"
            placeholder="Search agents, groves, or navigate..."
            .value=${this.query}
            @input=${this.handleInput}
          />

          <div class="results">
            ${this.items.length === 0
              ? html`<div class="no-results">No results for "${this.query}"</div>`
              : this.renderGrouped()
            }
          </div>

          <div class="shortcut-hint">
            <span><kbd>\u2191</kbd><kbd>\u2193</kbd> navigate</span>
            <span><kbd>Enter</kbd> select</span>
            <span><kbd>Esc</kbd> close</span>
          </div>
        </div>
      </div>
    `;
  }

  private renderGrouped() {
    const groups = new Map<string, CommandItem[]>();
    for (const item of this.items) {
      const group = groups.get(item.category) || [];
      group.push(item);
      groups.set(item.category, group);
    }

    const categoryLabels: Record<string, string> = {
      navigation: 'Navigation',
      agent: 'Agents',
      grove: 'Groves',
      action: 'Actions',
    };

    let globalIdx = 0;
    const fragments: unknown[] = [];

    for (const [cat, items] of groups) {
      fragments.push(html`<div class="category-header">${categoryLabels[cat] || cat}</div>`);
      for (const item of items) {
        const idx = globalIdx++;
        fragments.push(html`
          <div class="result-item ${idx === this.selectedIndex ? 'selected' : ''}"
            @click=${() => { item.action(); this.close(); }}
            @mouseenter=${() => { this.selectedIndex = idx; }}>
            <span class="result-icon">${item.icon}</span>
            <span class="result-label">${item.label}</span>
            ${item.description ? html`<span class="result-desc">${item.description}</span>` : nothing}
          </div>
        `);
      }
    }

    return fragments;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'scion-command-palette': ScionCommandPalette;
  }
}
