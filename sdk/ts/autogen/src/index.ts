/**
 * HELM governance adapter for AutoGen.
 *
 * Wraps AutoGen tool functions with HELM governance. Intercepts tool calls
 * through HELM's policy plane before execution.
 *
 * @example
 * ```typescript
 * import { HelmAutoGenGovernor } from '@mindburn/helm-autogen';
 *
 * const governor = new HelmAutoGenGovernor({ helmUrl: 'http://localhost:8080' });
 * const governed = governor.governTools(myTools);
 * ```
 */

export interface HelmAutoGenConfig {
  helmUrl: string;
  apiKey?: string;
  failClosed?: boolean;
  collectReceipts?: boolean;
  timeout?: number;
}

export interface ToolCallReceipt {
  toolName: string;
  args: Record<string, unknown>;
  receiptId: string;
  decision: 'APPROVED' | 'DENIED';
  reasonCode: string;
  durationMs: number;
  requestHash: string;
  outputHash: string;
}

export interface ToolCallDenial {
  toolName: string;
  args: Record<string, unknown>;
  reasonCode: string;
  message: string;
}

export class HelmToolDenyError extends Error {
  denial: ToolCallDenial;
  constructor(denial: ToolCallDenial) {
    super(`HELM denied "${denial.toolName}": ${denial.reasonCode} — ${denial.message}`);
    this.name = 'HelmToolDenyError';
    this.denial = denial;
  }
}

export class HelmAutoGenGovernor {
  private config: Required<HelmAutoGenConfig>;
  private _receipts: ToolCallReceipt[] = [];
  private _onReceipt?: (receipt: ToolCallReceipt) => void;
  private _onDeny?: (denial: ToolCallDenial) => void;

  constructor(config: HelmAutoGenConfig) {
    this.config = {
      helmUrl: config.helmUrl,
      apiKey: config.apiKey ?? '',
      failClosed: config.failClosed ?? true,
      collectReceipts: config.collectReceipts ?? true,
      timeout: config.timeout ?? 30000,
    };
  }

  onReceipt(cb: (receipt: ToolCallReceipt) => void): this {
    this._onReceipt = cb;
    return this;
  }

  onDeny(cb: (denial: ToolCallDenial) => void): this {
    this._onDeny = cb;
    return this;
  }

  get receipts(): ToolCallReceipt[] {
    return [...this._receipts];
  }

  clearReceipts(): void {
    this._receipts = [];
  }

  /**
   * Wrap a function-based tool with HELM governance.
   */
  governTool<T extends (...args: any[]) => any>(
    toolName: string,
    fn: T,
  ): T {
    const governor = this;
    const governed = async function (...args: any[]) {
      const toolArgs = args[0] && typeof args[0] === 'object' ? args[0] : { input: args };
      const startMs = performance.now();

      try {
        const response = await governor.evaluateIntent(toolName, toolArgs);
        const choices = response.choices ?? [];
        if (
          !choices.length ||
          (choices[0].finish_reason === 'stop' && !choices[0].message?.tool_calls?.length)
        ) {
          const denial: ToolCallDenial = {
            toolName,
            args: toolArgs,
            reasonCode: 'DENY_POLICY_VIOLATION',
            message: 'Denied by HELM governance',
          };
          governor._onDeny?.(denial);
          throw new HelmToolDenyError(denial);
        }

        const result = await fn(...args);
        const durationMs = performance.now() - startMs;

        const receipt: ToolCallReceipt = {
          toolName,
          args: toolArgs,
          receiptId: response.id ?? '',
          decision: 'APPROVED',
          reasonCode: 'ALLOW',
          durationMs,
          requestHash: `sha256:${await sha256(JSON.stringify(toolArgs))}`,
          outputHash: `sha256:${await sha256(String(result))}`,
        };

        if (governor.config.collectReceipts) governor._receipts.push(receipt);
        governor._onReceipt?.(receipt);
        return result;
      } catch (e) {
        if (e instanceof HelmToolDenyError) throw e;
        if (governor.config.failClosed) {
          throw new HelmToolDenyError({
            toolName,
            args: toolArgs,
            reasonCode: 'ERROR_INTERNAL',
            message: String(e),
          });
        }
        return fn(...args);
      }
    };
    return governed as unknown as T;
  }

  governTools(
    tools: Array<{ name: string; fn: (...args: any[]) => any }>,
  ): Array<{ name: string; fn: (...args: any[]) => any }> {
    return tools.map((t) => ({ name: t.name, fn: this.governTool(t.name, t.fn) }));
  }

  private async evaluateIntent(toolName: string, args: Record<string, unknown>): Promise<any> {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (this.config.apiKey) headers['Authorization'] = `Bearer ${this.config.apiKey}`;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const resp = await fetch(`${this.config.helmUrl}/v1/chat/completions`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          model: 'helm-governance',
          messages: [
            {
              role: 'user',
              content: JSON.stringify({
                type: 'tool_call_intent',
                tool: toolName,
                arguments: args,
              }),
            },
          ],
          tools: [{ type: 'function', function: { name: toolName } }],
        }),
        signal: controller.signal,
      });

      if (!resp.ok) {
        const body = await resp.json().catch(() => ({}));
        const err = (body as any).error ?? {};
        throw new HelmToolDenyError({
          toolName,
          args,
          reasonCode: err.reason_code ?? 'ERROR_INTERNAL',
          message: err.message ?? resp.statusText,
        });
      }

      return resp.json();
    } finally {
      clearTimeout(timeoutId);
    }
  }
}

async function sha256(data: string): Promise<string> {
  if (typeof globalThis.crypto?.subtle !== 'undefined') {
    const buffer = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(data));
    return Array.from(new Uint8Array(buffer))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
  }
  // Node.js fallback
  const { createHash } = await import('node:crypto');
  return createHash('sha256').update(data).digest('hex');
}
