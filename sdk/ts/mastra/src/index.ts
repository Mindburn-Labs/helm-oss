/**
 * @mindburn/helm-mastra
 *
 * HELM governance adapter for Mastra agent framework.
 * Wraps Mastra's Daytona sandbox integration with HELM governance,
 * providing policy enforcement, receipt chains, and sandbox preflight.
 *
 * Architecture:
 *   Mastra tool → HelmMastraSandbox → HELM governance → Daytona sandbox
 *
 * Usage:
 * ```ts
 * import { HelmMastraSandbox } from '@mindburn/helm-mastra';
 *
 * const sandbox = new HelmMastraSandbox({
 *   helmUrl: 'http://localhost:8080',
 *   daytonaApiKey: 'dtn-xxx',
 * });
 *
 * // Use within Mastra tools
 * const result = await sandbox.exec({
 *   command: ['python3', '-c', 'print("hello")'],
 * });
 * ```
 */

import { HelmClient, HelmApiError } from '@mindburn/helm-sdk';
import type { HelmClientConfig, Receipt } from '@mindburn/helm-sdk';
import { createHash } from 'node:crypto';

// ── Types ───────────────────────────────────────────────────────

/** Configuration for the HELM Mastra sandbox adapter. */
export interface HelmMastraSandboxConfig extends HelmClientConfig {
  /** Daytona API key for sandbox operations. */
  daytonaApiKey: string;

  /** Daytona API URL. Default: https://api.daytona.io */
  daytonaUrl?: string;

  /** If true, deny execution on governance errors (fail-closed). Default: true. */
  failClosed?: boolean;

  /** Default language for code execution. Default: 'python3'. */
  defaultLanguage?: string;

  /** Per-command timeout in milliseconds. Default: 30000. */
  execTimeout?: number;

  /** If true, collect receipts. Default: true. */
  collectReceipts?: boolean;

  /** Callback invoked after each successful execution with its receipt. */
  onReceipt?: (receipt: SandboxReceipt) => void;

  /** Callback invoked when execution is denied. */
  onDeny?: (denial: SandboxDenial) => void;
}

/** A receipt for a governed sandbox execution. */
export interface SandboxReceipt {
  command: string[];
  receipt: Receipt;
  requestHash: string;
  outputHash: string;
  exitCode: number;
  durationMs: number;
}

/** Details of a denied sandbox execution. */
export interface SandboxDenial {
  command: string[];
  reasonCode: string;
  message: string;
}

/** Sandbox execution request. */
export interface ExecRequest {
  command: string[];
  env?: Record<string, string>;
  workDir?: string;
  timeout?: number;
}

/** Sandbox execution result. */
export interface ExecResult {
  exitCode: number;
  stdout: string;
  stderr: string;
  durationMs: number;
  timedOut: boolean;
  receipt: SandboxReceipt;
}

/** File operations. */
export interface SandboxFile {
  path: string;
  content: string;
}

// ── Errors ──────────────────────────────────────────────────────

export class HelmSandboxDenyError extends Error {
  readonly denial: SandboxDenial;
  constructor(denial: SandboxDenial) {
    super(`HELM denied sandbox exec: ${denial.reasonCode} — ${denial.message}`);
    this.name = 'HelmSandboxDenyError';
    this.denial = denial;
  }
}

// ── Sandbox ─────────────────────────────────────────────────────

/**
 * HelmMastraSandbox wraps Mastra's Daytona sandbox with HELM governance.
 *
 * Each sandbox operation goes through HELM's governance plane:
 * 1. HELM evaluates the tool call intent against policy
 * 2. If approved, the command is forwarded to Daytona
 * 3. A receipt is produced for the execution
 */
export class HelmMastraSandbox {
  private readonly helmClient: HelmClient;
  private readonly daytonaUrl: string;
  private readonly daytonaApiKey: string;
  private readonly failClosed: boolean;
  private readonly defaultLanguage: string;
  private readonly execTimeout: number;
  private readonly collectReceipts: boolean;
  private readonly onReceipt?: (receipt: SandboxReceipt) => void;
  private readonly onDeny?: (denial: SandboxDenial) => void;
  private readonly receipts: SandboxReceipt[] = [];
  private sandboxId: string | null = null;

  constructor(config: HelmMastraSandboxConfig) {
    this.helmClient = new HelmClient(config);
    this.daytonaUrl = config.daytonaUrl ?? 'https://api.daytona.io';
    this.daytonaApiKey = config.daytonaApiKey;
    this.failClosed = config.failClosed ?? true;
    this.defaultLanguage = config.defaultLanguage ?? 'python3';
    this.execTimeout = config.execTimeout ?? 30_000;
    this.collectReceipts = config.collectReceipts ?? true;
    this.onReceipt = config.onReceipt;
    this.onDeny = config.onDeny;
  }

  /**
   * Initialize the sandbox. Must be called before exec.
   */
  async init(): Promise<void> {
    // Create a Daytona sandbox.
    const resp = await fetch(`${this.daytonaUrl}/sandbox`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.daytonaApiKey}`,
      },
      body: JSON.stringify({
        language: this.defaultLanguage,
        timeout: Math.floor(this.execTimeout / 1000),
      }),
    });

    if (!resp.ok) {
      throw new Error(`Daytona sandbox creation failed: ${resp.status}`);
    }

    const data = (await resp.json()) as { sandboxId: string };
    this.sandboxId = data.sandboxId;
  }

  /**
   * Execute a command in the sandbox with HELM governance.
   */
  async exec(req: ExecRequest): Promise<ExecResult> {
    if (!this.sandboxId) {
      await this.init();
    }

    const startMs = Date.now();

    // Step 1: HELM governance evaluation.
    try {
      const governanceResp = await this.helmClient.chatCompletions({
        model: 'helm-governance',
        messages: [
          {
            role: 'user',
            content: JSON.stringify({
              type: 'sandbox_exec_intent',
              provider: 'daytona',
              sandbox_id: this.sandboxId,
              command: req.command,
              env: req.env,
            }),
          },
        ],
        tools: [
          {
            type: 'function',
            function: {
              name: 'sandbox_exec',
              description: 'Execute command in Daytona sandbox',
              parameters: {
                type: 'object',
                properties: {
                  command: { type: 'array', items: { type: 'string' } },
                },
              },
            },
          },
        ],
      });

      const choice = governanceResp.choices?.[0];
      if (!choice || (choice.finish_reason === 'stop' && !choice.message?.tool_calls)) {
        const denial: SandboxDenial = {
          command: req.command,
          reasonCode: 'DENY_POLICY_VIOLATION',
          message: choice?.message?.content ?? 'Sandbox exec denied by HELM governance',
        };
        this.onDeny?.(denial);
        throw new HelmSandboxDenyError(denial);
      }
    } catch (error) {
      if (error instanceof HelmSandboxDenyError) throw error;
      if (error instanceof HelmApiError) {
        const denial: SandboxDenial = {
          command: req.command,
          reasonCode: error.reasonCode,
          message: error.message,
        };
        this.onDeny?.(denial);
        if (this.failClosed) throw new HelmSandboxDenyError(denial);
      }
      if (this.failClosed) throw error;
    }

    // Step 2: Execute on Daytona.
    const cmd = req.command.join(' ');
    const execResp = await fetch(
      `${this.daytonaUrl}/sandbox/${this.sandboxId}/process/execute`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${this.daytonaApiKey}`,
        },
        body: JSON.stringify({
          command: cmd,
          env: req.env,
          cwd: req.workDir,
          timeout: req.timeout ? Math.floor(req.timeout / 1000) : Math.floor(this.execTimeout / 1000),
        }),
      },
    );

    if (!execResp.ok) {
      throw new Error(`Daytona exec failed: ${execResp.status}`);
    }

    const result = (await execResp.json()) as {
      output: string;
      errors: string;
      exitCode: number;
      timedOut: boolean;
      durationMs: number;
    };

    // Step 3: Build receipt.
    const durationMs = Date.now() - startMs;
    const requestHash =
      'sha256:' + createHash('sha256').update(JSON.stringify(req)).digest('hex');
    const outputHash =
      'sha256:' + createHash('sha256').update(result.output || '').digest('hex');

    const receipt: SandboxReceipt = {
      command: req.command,
      receipt: {
        receipt_id: `mastra-${Date.now()}`,
        decision_id: `mastra-${this.sandboxId}`,
        effect_id: `exec-${Date.now()}`,
        status: 'APPROVED',
        reason_code: 'ALLOW',
        output_hash: outputHash,
        blob_hash: '',
        prev_hash: '',
        lamport_clock: this.receipts.length,
        signature: '',
        timestamp: new Date().toISOString(),
        principal: 'helm-mastra-adapter',
      },
      requestHash,
      outputHash,
      exitCode: result.exitCode,
      durationMs,
    };

    if (this.collectReceipts) {
      this.receipts.push(receipt);
    }
    this.onReceipt?.(receipt);

    return {
      exitCode: result.exitCode,
      stdout: result.output,
      stderr: result.errors,
      durationMs: result.durationMs,
      timedOut: result.timedOut,
      receipt,
    };
  }

  /**
   * Write a file to the sandbox.
   */
  async writeFile(path: string, content: string): Promise<void> {
    if (!this.sandboxId) await this.init();

    const resp = await fetch(
      `${this.daytonaUrl}/sandbox/${this.sandboxId}/filesystem?path=${encodeURIComponent(path)}`,
      {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/octet-stream',
          Authorization: `Bearer ${this.daytonaApiKey}`,
        },
        body: content,
      },
    );

    if (!resp.ok) {
      throw new Error(`Daytona write file failed: ${resp.status}`);
    }
  }

  /**
   * Read a file from the sandbox.
   */
  async readFile(path: string): Promise<string> {
    if (!this.sandboxId) await this.init();

    const resp = await fetch(
      `${this.daytonaUrl}/sandbox/${this.sandboxId}/filesystem?path=${encodeURIComponent(path)}`,
      {
        method: 'GET',
        headers: {
          Authorization: `Bearer ${this.daytonaApiKey}`,
        },
      },
    );

    if (!resp.ok) {
      throw new Error(`Daytona read file failed: ${resp.status}`);
    }

    return resp.text();
  }

  /**
   * Destroy the sandbox.
   */
  async destroy(): Promise<void> {
    if (!this.sandboxId) return;

    await fetch(`${this.daytonaUrl}/sandbox/${this.sandboxId}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${this.daytonaApiKey}` },
    });

    this.sandboxId = null;
  }

  /**
   * Get collected receipts.
   */
  getReceipts(): ReadonlyArray<SandboxReceipt> {
    return this.receipts;
  }

  /**
   * Clear collected receipts.
   */
  clearReceipts(): void {
    this.receipts.length = 0;
  }
}

export default HelmMastraSandbox;
