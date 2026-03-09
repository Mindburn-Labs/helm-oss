/**
 * HELM EffectBoundary Substrate Example — TypeScript
 *
 * Demonstrates implementing the EffectBoundary contract in TypeScript
 * using the OpenAPI-defined REST surface.
 */

import { createHash } from "crypto";
import { createServer, IncomingMessage, ServerResponse } from "http";

// --- Types ---

type Verdict = "ALLOW" | "DENY" | "ESCALATE";

interface EffectRequest {
  effect_type: string;
  principal_id: string;
  params?: Record<string, unknown>;
  context?: Record<string, unknown>;
  idempotency_key?: string;
}

interface Receipt {
  receipt_id: string;
  verdict: Verdict;
  reason_code: string;
  reason: string;
  timestamp: string;
  lamport: number;
  principal_id: string;
}

// --- EffectBoundary ---

class EffectBoundary {
  private lamport = 0;
  private receipts: Receipt[] = [];

  submit(req: EffectRequest): {
    verdict: Verdict;
    receipt: Receipt;
    intent: { effect_type: string; allowed: boolean };
  } {
    this.lamport++;
    const { verdict, reasonCode, reason } = this.evaluate(req);

    const receipt = this.createReceipt(
      verdict,
      reasonCode,
      reason,
      req.principal_id,
    );
    this.receipts.push(receipt);

    return {
      verdict,
      receipt,
      intent: {
        effect_type: req.effect_type,
        allowed: verdict === "ALLOW",
      },
    };
  }

  complete(receiptId: string, result: Record<string, unknown>): Receipt {
    this.lamport++;
    const receipt = this.createReceipt(
      "ALLOW",
      "EFFECT_COMPLETED",
      `Effect ${receiptId} completed`,
      "system",
    );
    this.receipts.push(receipt);
    return receipt;
  }

  private evaluate(req: EffectRequest): {
    verdict: Verdict;
    reasonCode: string;
    reason: string;
  } {
    if (req.effect_type === "data_export" && req.params?.data_class === "PII") {
      return {
        verdict: "DENY",
        reasonCode: "POLICY_VIOLATION",
        reason: "PII export denied by policy",
      };
    }

    if (req.effect_type === "financial_transfer") {
      const amount = (req.params?.amount_cents as number) ?? 0;
      if (amount > 1_000_000) {
        return {
          verdict: "ESCALATE",
          reasonCode: "TEMPORAL_INTERVENTION",
          reason: "High value transfer requires approval",
        };
      }
    }

    return {
      verdict: "ALLOW",
      reasonCode: "POLICY_SATISFIED",
      reason: "Effect allowed",
    };
  }

  private createReceipt(
    verdict: Verdict,
    reasonCode: string,
    reason: string,
    principalId: string,
  ): Receipt {
    const ts = new Date().toISOString();
    const content = `${verdict}:${reasonCode}:${ts}:${this.lamport}`;
    const hash = createHash("sha256")
      .update(content)
      .digest("hex")
      .slice(0, 16);

    return {
      receipt_id: `urn:helm:receipt:${hash}`,
      verdict,
      reason_code: reasonCode,
      reason,
      timestamp: ts,
      lamport: this.lamport,
      principal_id: principalId,
    };
  }
}

// --- HTTP Server ---

const boundary = new EffectBoundary();

function readBody(req: IncomingMessage): Promise<string> {
  return new Promise((resolve) => {
    let data = "";
    req.on("data", (chunk) => (data += chunk));
    req.on("end", () => resolve(data));
  });
}

const server = createServer(
  async (req: IncomingMessage, res: ServerResponse) => {
    if (req.method !== "POST") {
      res.writeHead(405);
      res.end(JSON.stringify({ error: "Method not allowed" }));
      return;
    }

    const body = JSON.parse(await readBody(req));

    if (req.url === "/v1/effects") {
      const result = boundary.submit(body as EffectRequest);
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify(result, null, 2));
    } else if (req.url?.match(/^\/v1\/effects\/[^/]+\/complete$/)) {
      const receiptId = req.url.split("/")[3];
      const result = boundary.complete(receiptId, body);
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify(result, null, 2));
    } else {
      res.writeHead(404);
      res.end(JSON.stringify({ error: "Not found" }));
    }
  },
);

console.log("HELM EffectBoundary substrate running on :4001");
server.listen(4001);
