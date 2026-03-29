/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Git diff viewer component.
 *
 * Fetches and displays the git diff and status for an agent's worktree.
 * Shows file-level summary and syntax-highlighted unified diff output
 * with collapsible file sections.
 */

import { LitElement, html, css, nothing } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import { apiFetch } from '../../client/api.js';

interface GitStatusResponse {
  branch?: string;
  commits?: GitCommit[];
  files?: GitFileStatus[];
  [key: string]: unknown;
}

interface GitCommit {
  hash?: string;
  subject?: string;
  author?: string;
  date?: string;
  [key: string]: unknown;
}

interface GitFileStatus {
  path?: string;
  status?: string;
  [key: string]: unknown;
}

interface FileDiff {
  header: string;
  fileName: string;
  additions: number;
  deletions: number;
  lines: DiffLine[];
}

interface DiffLine {
  type: 'addition' | 'deletion' | 'context' | 'header' | 'hunk';
  content: string;
}

function parseDiff(raw: string): FileDiff[] {
  const files: FileDiff[] = [];
  if (!raw) return files;

  const fileChunks = raw.split(/^diff --git /m);

  for (const chunk of fileChunks) {
    if (!chunk.trim()) continue;

    const fullHeader = 'diff --git ' + chunk.split('\n')[0];
    const fileMatch = chunk.match(/^(?:a\/(.+?)\s+b\/(.+?)$|(.+?)\s+(.+?)$)/m);
    const fileName = fileMatch
      ? (fileMatch[2] || fileMatch[4] || '').replace(/^b\//, '')
      : 'unknown';

    let additions = 0;
    let deletions = 0;
    const lines: DiffLine[] = [];

    const rawLines = chunk.split('\n');
    for (let i = 1; i < rawLines.length; i++) {
      const line = rawLines[i];
      if (
        line.startsWith('index ') ||
        line.startsWith('---') ||
        line.startsWith('+++') ||
        line.startsWith('old mode') ||
        line.startsWith('new mode') ||
        line.startsWith('new file') ||
        line.startsWith('deleted file') ||
        line.startsWith('similarity index') ||
        line.startsWith('rename from') ||
        line.startsWith('rename to') ||
        line.startsWith('Binary files')
      ) {
        lines.push({ type: 'header', content: line });
      } else if (line.startsWith('@@')) {
        lines.push({ type: 'hunk', content: line });
      } else if (line.startsWith('+')) {
        additions++;
        lines.push({ type: 'addition', content: line });
      } else if (line.startsWith('-')) {
        deletions++;
        lines.push({ type: 'deletion', content: line });
      } else {
        lines.push({ type: 'context', content: line });
      }
    }

    files.push({
      header: fullHeader,
      fileName,
      additions,
      deletions,
      lines,
    });
  }

  return files;
}

@customElement('scion-git-diff-viewer')
export class ScionGitDiffViewer extends LitElement {
  @property()
  agentId = '';

  @state() private loading = false;
  @state() private error: string | null = null;
  @state() private loaded = false;
  @state() private diffRaw = '';
  @state() private fileDiffs: FileDiff[] = [];
  @state() private gitStatus: GitStatusResponse | null = null;
  @state() private collapsedFiles = new Set<string>();

  static override styles = css`
    :host {
      display: block;
    }

    /* Toolbar */
    .toolbar {
      display: flex;
      align-items: center;
      justify-content: flex-end;
      gap: 0.75rem;
      margin-bottom: 1rem;
    }

    /* Summary bar */
    .summary {
      display: flex;
      align-items: center;
      gap: 1rem;
      padding: 0.75rem 1rem;
      background: var(--scion-bg-subtle, #f1f5f9);
      border: 1px solid var(--scion-border, #e2e8f0);
      border-radius: 0.375rem;
      margin-bottom: 1rem;
      font-size: 0.8125rem;
      flex-wrap: wrap;
    }
    .summary-item {
      display: inline-flex;
      align-items: center;
      gap: 0.375rem;
      color: var(--scion-text-secondary, #475569);
    }
    .summary-item.additions {
      color: var(--scion-success-700, #15803d);
    }
    .summary-item.deletions {
      color: var(--scion-danger-700, #b91c1c);
    }
    .summary-item.branch {
      font-weight: 600;
      color: var(--scion-primary-700, #1d4ed8);
    }

    /* Commits list */
    .commits {
      margin-bottom: 1rem;
    }
    .commits-title {
      font-size: 0.8125rem;
      font-weight: 600;
      color: var(--scion-text-secondary, #475569);
      margin-bottom: 0.5rem;
    }
    .commit-row {
      display: flex;
      align-items: baseline;
      gap: 0.75rem;
      padding: 0.25rem 0;
      font-size: 0.8125rem;
      border-bottom: 1px solid var(--scion-border, #e2e8f0);
    }
    .commit-hash {
      font-family: var(--scion-font-mono, monospace);
      font-size: 0.75rem;
      color: var(--scion-primary-600, #2563eb);
      flex-shrink: 0;
    }
    .commit-subject {
      flex: 1;
      color: var(--scion-text-primary, #1e293b);
    }
    .commit-meta {
      font-size: 0.75rem;
      color: var(--scion-text-muted, #64748b);
      flex-shrink: 0;
    }

    /* File section */
    .file-section {
      border: 1px solid var(--scion-border, #e2e8f0);
      border-radius: 0.375rem;
      margin-bottom: 0.75rem;
      overflow: hidden;
    }
    .file-header {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.5rem 0.75rem;
      background: var(--scion-bg-subtle, #f1f5f9);
      border-bottom: 1px solid var(--scion-border, #e2e8f0);
      cursor: pointer;
      user-select: none;
      font-size: 0.8125rem;
    }
    .file-header:hover {
      background: var(--scion-neutral-100, #e2e8f0);
    }
    .file-name {
      font-family: var(--scion-font-mono, monospace);
      font-weight: 600;
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .file-stats {
      display: inline-flex;
      gap: 0.5rem;
      font-size: 0.75rem;
      flex-shrink: 0;
    }
    .file-stats .add {
      color: var(--scion-success-700, #15803d);
      font-weight: 600;
    }
    .file-stats .del {
      color: var(--scion-danger-700, #b91c1c);
      font-weight: 600;
    }
    .chevron {
      font-size: 0.75rem;
      transition: transform 0.15s ease;
      flex-shrink: 0;
    }
    .chevron.collapsed {
      transform: rotate(-90deg);
    }

    /* Diff lines */
    .diff-body {
      overflow-x: auto;
      font-family: var(--scion-font-mono, monospace);
      font-size: 0.75rem;
      line-height: 1.5;
    }
    .diff-line {
      padding: 0 0.75rem;
      white-space: pre;
      min-height: 1.5em;
    }
    .diff-line.addition {
      background: var(--scion-success-50, #f0fdf4);
      color: var(--scion-success-800, #166534);
    }
    .diff-line.deletion {
      background: var(--scion-danger-50, #fef2f2);
      color: var(--scion-danger-800, #991b1b);
    }
    .diff-line.hunk {
      background: var(--scion-primary-50, #eff6ff);
      color: var(--scion-primary-700, #1d4ed8);
      font-style: italic;
    }
    .diff-line.header {
      color: var(--scion-text-muted, #64748b);
      font-style: italic;
    }
    .diff-line.context {
      color: var(--scion-text-secondary, #475569);
    }

    /* Empty / Loading / Error */
    .state-msg {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 3rem 2rem;
      color: var(--scion-text-muted, #64748b);
      gap: 0.75rem;
    }
    .state-msg sl-spinner {
      font-size: 1.5rem;
    }
    .state-msg sl-icon {
      font-size: 2rem;
      opacity: 0.4;
    }
  `;

  override disconnectedCallback(): void {
    super.disconnectedCallback();
  }

  /** Called by the parent when the changes tab is first shown. */
  loadDiff(): void {
    if (this.loaded) return;
    this.loaded = true;
    void this.fetchAll();
  }

  private async fetchAll(): Promise<void> {
    if (!this.agentId) return;
    this.loading = true;
    this.error = null;

    try {
      const [statusRes, diffRes] = await Promise.all([
        apiFetch(`/api/v1/agents/${this.agentId}/git-status`),
        apiFetch(`/api/v1/agents/${this.agentId}/git-diff?base=main`),
      ]);

      if (!statusRes.ok) {
        const errData = (await statusRes.json().catch(() => ({}))) as {
          error?: { message?: string };
          message?: string;
        };
        throw new Error(
          (errData.error as { message?: string })?.message ||
            errData.message ||
            `HTTP ${statusRes.status}`
        );
      }

      if (!diffRes.ok) {
        const errData = (await diffRes.json().catch(() => ({}))) as {
          error?: { message?: string };
          message?: string;
        };
        throw new Error(
          (errData.error as { message?: string })?.message ||
            errData.message ||
            `HTTP ${diffRes.status}`
        );
      }

      this.gitStatus = (await statusRes.json()) as GitStatusResponse;

      const diffData = (await diffRes.json()) as { diff?: string; [key: string]: unknown };
      this.diffRaw = typeof diffData === 'string' ? diffData : (diffData.diff ?? '');
      this.fileDiffs = parseDiff(this.diffRaw);
    } catch (err) {
      this.error = err instanceof Error ? err.message : 'Failed to fetch diff';
    } finally {
      this.loading = false;
    }
  }

  private handleRefresh(): void {
    this.loaded = false;
    this.loadDiff();
  }

  private toggleFile(fileName: string): void {
    if (this.collapsedFiles.has(fileName)) {
      this.collapsedFiles.delete(fileName);
    } else {
      this.collapsedFiles.add(fileName);
    }
    this.requestUpdate();
  }

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  override render() {
    return html`
      ${this.renderToolbar()}
      ${this.renderContent()}
    `;
  }

  private renderToolbar() {
    return html`
      <div class="toolbar">
        <sl-button
          size="small"
          variant="default"
          ?loading=${this.loading}
          ?disabled=${this.loading}
          @click=${this.handleRefresh}
        >
          <sl-icon slot="prefix" name="arrow-clockwise"></sl-icon>
          Refresh
        </sl-button>
      </div>
    `;
  }

  private renderContent() {
    if (this.loading) {
      return html`
        <div class="state-msg">
          <sl-spinner></sl-spinner>
          <span>Loading changes...</span>
        </div>
      `;
    }

    if (this.error) {
      return html`
        <div class="state-msg">
          <sl-icon name="exclamation-triangle"></sl-icon>
          <span>${this.error}</span>
          <sl-button size="small" @click=${this.handleRefresh}>Retry</sl-button>
        </div>
      `;
    }

    if (!this.loaded) {
      return html`
        <div class="state-msg">
          <sl-icon name="file-diff"></sl-icon>
          <span>Select this tab to load changes</span>
        </div>
      `;
    }

    if (this.fileDiffs.length === 0) {
      return html`
        <div class="state-msg">
          <sl-icon name="check-circle"></sl-icon>
          <span>No changes found</span>
        </div>
      `;
    }

    const totalAdditions = this.fileDiffs.reduce((s, f) => s + f.additions, 0);
    const totalDeletions = this.fileDiffs.reduce((s, f) => s + f.deletions, 0);

    return html`
      ${this.renderSummary(totalAdditions, totalDeletions)}
      ${this.renderCommits()}
      ${this.fileDiffs.map((f) => this.renderFileSection(f))}
    `;
  }

  private renderSummary(totalAdditions: number, totalDeletions: number) {
    const branch = this.gitStatus?.branch;
    return html`
      <div class="summary">
        ${branch
          ? html`<span class="summary-item branch">
              <sl-icon name="diagram-2"></sl-icon>
              ${branch}
            </span>`
          : nothing}
        <span class="summary-item">
          <sl-icon name="file-earmark-diff"></sl-icon>
          ${this.fileDiffs.length} file${this.fileDiffs.length !== 1 ? 's' : ''} changed
        </span>
        <span class="summary-item additions">+${totalAdditions}</span>
        <span class="summary-item deletions">-${totalDeletions}</span>
      </div>
    `;
  }

  private renderCommits() {
    const commits = this.gitStatus?.commits;
    if (!commits || commits.length === 0) return nothing;

    return html`
      <div class="commits">
        <div class="commits-title">Commits</div>
        ${commits.map(
          (c) => html`
            <div class="commit-row">
              <span class="commit-hash">${(c.hash ?? '').substring(0, 7)}</span>
              <span class="commit-subject">${c.subject ?? ''}</span>
              <span class="commit-meta">${c.author ?? ''}${c.date ? ` - ${c.date}` : ''}</span>
            </div>
          `
        )}
      </div>
    `;
  }

  private renderFileSection(file: FileDiff) {
    const collapsed = this.collapsedFiles.has(file.fileName);
    return html`
      <div class="file-section">
        <div class="file-header" @click=${() => this.toggleFile(file.fileName)}>
          <sl-icon
            name="chevron-down"
            class="chevron ${collapsed ? 'collapsed' : ''}"
          ></sl-icon>
          <span class="file-name">${file.fileName}</span>
          <span class="file-stats">
            <span class="add">+${file.additions}</span>
            <span class="del">-${file.deletions}</span>
          </span>
        </div>
        ${collapsed
          ? nothing
          : html`
              <div class="diff-body">
                ${file.lines.map(
                  (line) =>
                    html`<div class="diff-line ${line.type}">${line.content}</div>`
                )}
              </div>
            `}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'scion-git-diff-viewer': ScionGitDiffViewer;
  }
}
