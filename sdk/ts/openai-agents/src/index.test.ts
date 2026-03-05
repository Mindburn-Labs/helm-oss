import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { HelmToolProxy, HelmToolDenyError } from "./index.js";
import type {
  ToolDefinition,
  GovernedTool,
  ToolCallReceipt,
  ToolCallDenial,
} from "./index.js";

// ── Helpers ─────────────────────────────────────────────────────

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function approveResponse(
  toolName: string,
  args: Record<string, unknown> = {},
): Response {
  return jsonResponse({
    id: `chatcmpl-${Date.now()}`,
    object: "chat.completion",
    created: Date.now(),
    model: "helm-governance",
    choices: [
      {
        index: 0,
        message: {
          role: "assistant",
          content: null,
          tool_calls: [
            {
              id: `call_${Date.now()}`,
              type: "function",
              function: {
                name: toolName,
                arguments: JSON.stringify(args),
              },
            },
          ],
        },
        finish_reason: "tool_calls",
      },
    ],
  });
}

function denyResponse(
  reason = "Tool call denied by HELM governance",
): Response {
  return jsonResponse({
    id: `chatcmpl-${Date.now()}`,
    object: "chat.completion",
    created: Date.now(),
    model: "helm-governance",
    choices: [
      {
        index: 0,
        message: { role: "assistant", content: reason },
        finish_reason: "stop",
      },
    ],
  });
}

function helmApiErrorResponse(
  status: number,
  reasonCode = "DENY_POLICY_VIOLATION",
  message = "denied",
): Response {
  return jsonResponse(
    {
      error: {
        message,
        type: "permission_denied",
        code: "DENY",
        reason_code: reasonCode,
      },
    },
    status,
  );
}

function makeTool(
  name: string,
  run?: (args: Record<string, unknown>) => Promise<unknown>,
): ToolDefinition {
  return {
    type: "function",
    function: {
      name,
      description: `${name} tool`,
      parameters: { type: "object", properties: { input: { type: "string" } } },
    },
    run: run ?? (async (args) => `result:${JSON.stringify(args)}`),
  };
}

// ── Tests ───────────────────────────────────────────────────────

describe("HelmToolProxy", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  // ── Metadata preservation ──────────────────────────────
  describe("metadata preservation", () => {
    it("preserves tool function schema in governed tool", () => {
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const tool = makeTool("get_weather");
      const governed = proxy.wrapTool(tool);

      expect(governed.type).toBe("function");
      expect(governed.function.name).toBe("get_weather");
      expect(governed.function.description).toBe("get_weather tool");
      expect(governed.function.parameters).toEqual(tool.function.parameters);
    });

    it("preserves reference to original tool", () => {
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const tool = makeTool("calculator");
      const governed = proxy.wrapTool(tool);

      expect(governed._original).toBe(tool);
    });

    it("wraps multiple tools preserving all schemas", () => {
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const tools = [makeTool("a"), makeTool("b"), makeTool("c")];
      const governed = proxy.wrapTools(tools);

      expect(governed).toHaveLength(3);
      expect(governed.map((g) => g.function.name)).toEqual(["a", "b", "c"]);
    });
  });

  // ── Approval path ──────────────────────────────────────
  describe("approval path", () => {
    it("executes tool when HELM approves", async () => {
      fetchSpy.mockResolvedValue(approveResponse("run_code"));
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const tool = makeTool(
        "run_code",
        async (args) => `executed:${args.input}`,
      );
      const governed = proxy.wrapTool(tool);

      const result = await governed.run({ input: "hello" });
      expect(result).toBe("executed:hello");
    });

    it("sends tool call intent as structured message", async () => {
      fetchSpy.mockResolvedValue(approveResponse("search"));
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const governed = proxy.wrapTool(makeTool("search"));

      await governed.run({ input: "query" });

      const body = JSON.parse(fetchSpy.mock.calls[0][1].body);
      expect(body.model).toBe("helm-governance");
      expect(body.messages[0].role).toBe("user");
      const intent = JSON.parse(body.messages[0].content);
      expect(intent.type).toBe("tool_call_intent");
      expect(intent.tool).toBe("search");
      expect(intent.arguments).toEqual({ input: "query" });
    });
  });

  // ── Denial path ────────────────────────────────────────
  describe("denial path", () => {
    it("throws HelmToolDenyError when kernel denies", async () => {
      fetchSpy.mockResolvedValue(denyResponse("Policy violation: no writes"));
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const governed = proxy.wrapTool(makeTool("write_file"));

      await expect(governed.run({ path: "/etc/passwd" })).rejects.toThrow(
        HelmToolDenyError,
      );
    });

    it("denial error contains reason code and tool name", async () => {
      fetchSpy.mockResolvedValue(denyResponse("Budget exceeded"));
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const governed = proxy.wrapTool(makeTool("expensive_tool"));

      try {
        await governed.run({});
        expect.unreachable("should have thrown");
      } catch (e) {
        const err = e as HelmToolDenyError;
        expect(err.denial.toolName).toBe("expensive_tool");
        expect(err.denial.reasonCode).toBe("DENY_POLICY_VIOLATION");
        expect(err.name).toBe("HelmToolDenyError");
      }
    });

    it("invokes onDeny callback on denial", async () => {
      fetchSpy.mockResolvedValue(denyResponse());
      const denials: ToolCallDenial[] = [];
      const proxy = new HelmToolProxy({
        baseUrl: "http://h",
        onDeny: (d) => denials.push(d),
      });
      const governed = proxy.wrapTool(makeTool("blocked_tool"));

      try {
        await governed.run({ x: 1 });
      } catch {
        /* expected */
      }

      expect(denials).toHaveLength(1);
      expect(denials[0].toolName).toBe("blocked_tool");
    });

    it("never executes underlying tool on denial", async () => {
      fetchSpy.mockResolvedValue(denyResponse());
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const runSpy = vi.fn();
      const tool = makeTool("guarded", runSpy);
      const governed = proxy.wrapTool(tool);

      try {
        await governed.run({});
      } catch {
        /* expected */
      }

      expect(runSpy).not.toHaveBeenCalled();
    });
  });

  // ── Fail-closed behavior ───────────────────────────────
  describe("fail-closed", () => {
    it("throws on HELM API error when failClosed=true (default)", async () => {
      fetchSpy.mockResolvedValue(helmApiErrorResponse(500));
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const governed = proxy.wrapTool(makeTool("tool"));

      await expect(governed.run({})).rejects.toThrow(HelmToolDenyError);
    });

    it("falls through to direct execution when failClosed=false", async () => {
      fetchSpy.mockResolvedValue(helmApiErrorResponse(500));
      const proxy = new HelmToolProxy({
        baseUrl: "http://h",
        failClosed: false,
      });
      const governed = proxy.wrapTool(makeTool("tool", async () => "direct"));

      const result = await governed.run({});
      expect(result).toBe("direct");
    });

    it("does NOT fall through on denial even when failClosed=false", async () => {
      fetchSpy.mockResolvedValue(denyResponse());
      const proxy = new HelmToolProxy({
        baseUrl: "http://h",
        failClosed: false,
      });
      const governed = proxy.wrapTool(makeTool("tool"));

      // Denial is NOT an API error — it is a governance decision, must always throw
      await expect(governed.run({})).rejects.toThrow(HelmToolDenyError);
    });

    it("throws on network error when failClosed=true", async () => {
      fetchSpy.mockRejectedValue(new Error("ECONNREFUSED"));
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const governed = proxy.wrapTool(makeTool("tool"));

      await expect(governed.run({})).rejects.toThrow();
    });
  });

  // ── Receipt collection ─────────────────────────────────
  describe("receipt collection", () => {
    it("collects receipts for approved calls", async () => {
      fetchSpy.mockResolvedValue(approveResponse("tool1"));
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const governed = proxy.wrapTool(makeTool("tool1"));

      await governed.run({ a: 1 });

      const receipts = proxy.getReceipts();
      expect(receipts).toHaveLength(1);
      expect(receipts[0].toolName).toBe("tool1");
      expect(receipts[0].args).toEqual({ a: 1 });
      expect(receipts[0].receipt.status).toBe("APPROVED");
      expect(receipts[0].durationMs).toBeGreaterThanOrEqual(0);
    });

    it("invokes onReceipt callback", async () => {
      fetchSpy.mockResolvedValue(approveResponse("tool1"));
      const receipts: ToolCallReceipt[] = [];
      const proxy = new HelmToolProxy({
        baseUrl: "http://h",
        onReceipt: (r) => receipts.push(r),
      });
      const governed = proxy.wrapTool(makeTool("tool1"));

      await governed.run({});

      expect(receipts).toHaveLength(1);
    });

    it("does not collect receipts when collectReceipts=false", async () => {
      fetchSpy.mockResolvedValue(approveResponse("tool1"));
      const proxy = new HelmToolProxy({
        baseUrl: "http://h",
        collectReceipts: false,
      });
      const governed = proxy.wrapTool(makeTool("tool1"));

      await governed.run({});

      expect(proxy.getReceipts()).toHaveLength(0);
    });

    it("clearReceipts empties the collection", async () => {
      fetchSpy.mockResolvedValue(approveResponse("tool1"));
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const governed = proxy.wrapTool(makeTool("tool1"));

      await governed.run({});
      expect(proxy.getReceipts()).toHaveLength(1);

      proxy.clearReceipts();
      expect(proxy.getReceipts()).toHaveLength(0);
    });
  });

  // ── Concurrent tool calls ──────────────────────────────
  describe("concurrency ordering", () => {
    it("maintains receipt ordering across concurrent calls", async () => {
      let callIndex = 0;
      fetchSpy.mockImplementation(async () => {
        callIndex++;
        return approveResponse(`tool_${callIndex}`);
      });

      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const tools = ["a", "b", "c"].map((n) => proxy.wrapTool(makeTool(n)));

      await Promise.all(tools.map((t) => t.run({})));

      const receipts = proxy.getReceipts();
      expect(receipts).toHaveLength(3);
      // All three tools should have receipts
      expect(new Set(receipts.map((r) => r.toolName))).toEqual(
        new Set(["a", "b", "c"]),
      );
    });
  });

  // ── Error cases ────────────────────────────────────────
  describe("error cases", () => {
    it("throws if tool has no run implementation", async () => {
      fetchSpy.mockResolvedValue(approveResponse("no_run"));
      const proxy = new HelmToolProxy({ baseUrl: "http://h" });
      const tool: ToolDefinition = {
        type: "function",
        function: { name: "no_run", description: "test" },
        // No run method
      };
      const governed = proxy.wrapTool(tool);

      await expect(governed.run({})).rejects.toThrow(
        "has no run implementation",
      );
    });
  });
});
