import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { HelmMastraSandbox, HelmSandboxDenyError } from './index.js';
import type { SandboxReceipt, SandboxDenial } from './index.js';

// ── Helpers ─────────────────────────────────────────────────────

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function createSandboxResponse(sandboxId = 'sbx-1234'): Response {
  return jsonResponse({ sandboxId });
}

function execResponse(output = 'hello\n', exitCode = 0): Response {
  return jsonResponse({
    output,
    errors: '',
    exitCode,
    timedOut: false,
    durationMs: 42,
  });
}

function governanceApproveResponse(): Response {
  return jsonResponse({
    id: `chatcmpl-${Date.now()}`,
    object: 'chat.completion',
    created: Date.now(),
    model: 'helm-governance',
    choices: [
      {
        index: 0,
        message: {
          role: 'assistant',
          content: null,
          tool_calls: [
            {
              id: `call_${Date.now()}`,
              type: 'function',
              function: { name: 'sandbox_exec', arguments: '{}' },
            },
          ],
        },
        finish_reason: 'tool_calls',
      },
    ],
  });
}

function governanceDenyResponse(reason = 'Sandbox exec denied by HELM governance'): Response {
  return jsonResponse({
    id: `chatcmpl-${Date.now()}`,
    object: 'chat.completion',
    created: Date.now(),
    model: 'helm-governance',
    choices: [
      {
        index: 0,
        message: { role: 'assistant', content: reason },
        finish_reason: 'stop',
      },
    ],
  });
}

function helmApiErrorResponse(
  status: number,
  reasonCode = 'DENY_POLICY_VIOLATION',
  message = 'denied',
): Response {
  return jsonResponse(
    {
      error: {
        message,
        type: 'permission_denied',
        code: 'DENY',
        reason_code: reasonCode,
      },
    },
    status,
  );
}

// ── Tests ───────────────────────────────────────────────────────

describe('HelmMastraSandbox', () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal('fetch', fetchSpy);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  function makeSandbox(overrides: Record<string, unknown> = {}) {
    return new HelmMastraSandbox({
      baseUrl: 'http://helm:8080',
      daytonaApiKey: 'dtn-test-key',
      daytonaUrl: 'http://daytona:3000',
      ...overrides,
    });
  }

  // ── Sandbox creation ───────────────────────────────────
  describe('init', () => {
    it('creates a Daytona sandbox with correct auth', async () => {
      fetchSpy.mockResolvedValue(createSandboxResponse());
      const sandbox = makeSandbox();

      await sandbox.init();

      expect(fetchSpy).toHaveBeenCalledWith(
        'http://daytona:3000/sandbox',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            Authorization: 'Bearer dtn-test-key',
          }),
        }),
      );
    });

    it('throws on Daytona creation failure', async () => {
      fetchSpy.mockResolvedValue(new Response('fail', { status: 500 }));
      const sandbox = makeSandbox();

      await expect(sandbox.init()).rejects.toThrow('Daytona sandbox creation failed: 500');
    });
  });

  // ── Governance-approved execution ──────────────────────
  describe('exec (approved)', () => {
    it('executes command through HELM governance then Daytona', async () => {
      // Call 1: Daytona create (auto-init)
      // Call 2: HELM governance
      // Call 3: Daytona exec
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse('sbx-1'))
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse('world\n', 0));

      const sandbox = makeSandbox();
      const result = await sandbox.exec({ command: ['echo', 'world'] });

      expect(result.stdout).toBe('world\n');
      expect(result.exitCode).toBe(0);
      expect(result.timedOut).toBe(false);
    });

    it('sends governance intent with sandbox provider metadata', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse('sbx-meta'))
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse());

      const sandbox = makeSandbox();
      await sandbox.exec({ command: ['ls', '-la'] });

      // Call index 1 is HELM governance
      const helmBody = JSON.parse(fetchSpy.mock.calls[1][1].body);
      const intent = JSON.parse(helmBody.messages[0].content);
      expect(intent.type).toBe('sandbox_exec_intent');
      expect(intent.provider).toBe('daytona');
      expect(intent.sandbox_id).toBe('sbx-meta');
      expect(intent.command).toEqual(['ls', '-la']);
    });
  });

  // ── Denial path ────────────────────────────────────────
  describe('exec (denied)', () => {
    it('throws HelmSandboxDenyError when governance denies', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceDenyResponse('Not allowed'));

      const sandbox = makeSandbox();
      await expect(sandbox.exec({ command: ['rm', '-rf', '/'] })).rejects.toThrow(
        HelmSandboxDenyError,
      );
    });

    it('denial error contains command and reason', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceDenyResponse('Blocked by policy'));

      const sandbox = makeSandbox();
      try {
        await sandbox.exec({ command: ['dangerous'] });
        expect.unreachable('should have thrown');
      } catch (e) {
        const err = e as HelmSandboxDenyError;
        expect(err.denial.command).toEqual(['dangerous']);
        expect(err.denial.reasonCode).toBe('DENY_POLICY_VIOLATION');
        expect(err.name).toBe('HelmSandboxDenyError');
      }
    });

    it('invokes onDeny callback', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceDenyResponse());

      const denials: SandboxDenial[] = [];
      const sandbox = makeSandbox({ onDeny: (d: SandboxDenial) => denials.push(d) });

      try { await sandbox.exec({ command: ['test'] }); } catch { /* expected */ }
      expect(denials).toHaveLength(1);
    });

    it('never reaches Daytona on denial', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceDenyResponse());

      const sandbox = makeSandbox();
      try { await sandbox.exec({ command: ['test'] }); } catch { /* expected */ }

      // Only 2 calls: Daytona create + HELM governance. No Daytona exec.
      expect(fetchSpy).toHaveBeenCalledTimes(2);
    });
  });

  // ── Fail-closed behavior ───────────────────────────────
  describe('fail-closed', () => {
    it('throws on HELM API 500 when failClosed=true (default)', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(helmApiErrorResponse(500));

      const sandbox = makeSandbox();
      await expect(sandbox.exec({ command: ['echo'] })).rejects.toThrow();
    });

    it('throws on HELM network error when failClosed=true', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockRejectedValueOnce(new Error('ECONNREFUSED'));

      const sandbox = makeSandbox();
      await expect(sandbox.exec({ command: ['echo'] })).rejects.toThrow('ECONNREFUSED');
    });
  });

  // ── Receipt collection ─────────────────────────────────
  describe('receipt collection', () => {
    it('collects receipt with deterministic request hash', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse('out', 0));

      const sandbox = makeSandbox();
      const result = await sandbox.exec({ command: ['echo', 'out'] });

      expect(result.receipt).toBeDefined();
      expect(result.receipt.requestHash).toMatch(/^sha256:[a-f0-9]{64}$/);
      expect(result.receipt.outputHash).toMatch(/^sha256:[a-f0-9]{64}$/);
      expect(result.receipt.receipt.status).toBe('APPROVED');
    });

    it('same input produces same request hash', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse('a'))
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse('a'));

      const sandbox = makeSandbox();
      const r1 = await sandbox.exec({ command: ['echo', 'deterministic'] });
      const r2 = await sandbox.exec({ command: ['echo', 'deterministic'] });

      expect(r1.receipt.requestHash).toBe(r2.receipt.requestHash);
    });

    it('different output produces different output hash', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse('output_A'))
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse('output_B'));

      const sandbox = makeSandbox();
      const r1 = await sandbox.exec({ command: ['echo', 'A'] });
      const r2 = await sandbox.exec({ command: ['echo', 'B'] });

      expect(r1.receipt.outputHash).not.toBe(r2.receipt.outputHash);
    });

    it('invokes onReceipt callback', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse());

      const receipts: SandboxReceipt[] = [];
      const sandbox = makeSandbox({ onReceipt: (r: SandboxReceipt) => receipts.push(r) });

      await sandbox.exec({ command: ['echo'] });
      expect(receipts).toHaveLength(1);
    });

    it('getReceipts returns accumulated receipts', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse())
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse());

      const sandbox = makeSandbox();
      await sandbox.exec({ command: ['cmd1'] });
      await sandbox.exec({ command: ['cmd2'] });

      expect(sandbox.getReceipts()).toHaveLength(2);
    });

    it('clearReceipts empties the collection', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse());

      const sandbox = makeSandbox();
      await sandbox.exec({ command: ['echo'] });
      expect(sandbox.getReceipts()).toHaveLength(1);

      sandbox.clearReceipts();
      expect(sandbox.getReceipts()).toHaveLength(0);
    });

    it('receipt lamport clock increments', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse())
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse());

      const sandbox = makeSandbox();
      await sandbox.exec({ command: ['echo', 'a'] });
      await sandbox.exec({ command: ['echo', 'b'] });

      const receipts = sandbox.getReceipts();
      expect(receipts[0].receipt.lamport_clock).toBe(0);
      expect(receipts[1].receipt.lamport_clock).toBe(1);
    });
  });

  // ── Sandbox lifecycle ──────────────────────────────────
  describe('lifecycle', () => {
    it('auto-initializes on first exec', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse('auto-init'))
        .mockResolvedValueOnce(governanceApproveResponse())
        .mockResolvedValueOnce(execResponse());

      const sandbox = makeSandbox();
      // Don't call init() explicitly
      await sandbox.exec({ command: ['echo'] });

      // First call should be Daytona create
      expect(fetchSpy.mock.calls[0][0]).toBe('http://daytona:3000/sandbox');
    });

    it('destroy() sends DELETE and clears sandbox ID', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse('destroy-me'))
        .mockResolvedValueOnce(new Response(null, { status: 204 }));

      const sandbox = makeSandbox();
      await sandbox.init();
      await sandbox.destroy();

      const deleteCall = fetchSpy.mock.calls[1];
      expect(deleteCall[0]).toBe('http://daytona:3000/sandbox/destroy-me');
      expect(deleteCall[1].method).toBe('DELETE');
    });

    it('destroy() is idempotent when not initialized', async () => {
      const sandbox = makeSandbox();
      await sandbox.destroy(); // should not throw
      expect(fetchSpy).not.toHaveBeenCalled();
    });
  });

  // ── File operations ────────────────────────────────────
  describe('file operations', () => {
    it('writeFile sends PUT with content', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse('fs-test'))
        .mockResolvedValueOnce(new Response(null, { status: 200 }));

      const sandbox = makeSandbox();
      await sandbox.init();
      await sandbox.writeFile('/app/test.py', 'print("hello")');

      const writeCall = fetchSpy.mock.calls[1];
      expect(writeCall[0]).toContain('/sandbox/fs-test/filesystem');
      expect(writeCall[0]).toContain('path=%2Fapp%2Ftest.py');
      expect(writeCall[1].method).toBe('PUT');
      expect(writeCall[1].body).toBe('print("hello")');
    });

    it('readFile sends GET and returns content', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse('fs-read'))
        .mockResolvedValueOnce(new Response('file content', { status: 200 }));

      const sandbox = makeSandbox();
      await sandbox.init();
      const content = await sandbox.readFile('/app/out.txt');

      expect(content).toBe('file content');
      const readCall = fetchSpy.mock.calls[1];
      expect(readCall[0]).toContain('path=%2Fapp%2Fout.txt');
      expect(readCall[1].method).toBe('GET');
    });

    it('writeFile throws on failure', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(new Response('fail', { status: 500 }));

      const sandbox = makeSandbox();
      await sandbox.init();

      await expect(sandbox.writeFile('/bad', 'data')).rejects.toThrow(
        'Daytona write file failed: 500',
      );
    });

    it('readFile throws on failure', async () => {
      fetchSpy
        .mockResolvedValueOnce(createSandboxResponse())
        .mockResolvedValueOnce(new Response('fail', { status: 404 }));

      const sandbox = makeSandbox();
      await sandbox.init();

      await expect(sandbox.readFile('/missing')).rejects.toThrow(
        'Daytona read file failed: 404',
      );
    });
  });
});
