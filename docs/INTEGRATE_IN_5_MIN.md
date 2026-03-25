---
title: INTEGRATE_IN_5_MIN
---

# Integrate HELM in 5 Minutes

No trust required. Every step produces verifiable output.

---

## 1. Start HELM

```bash
docker compose up -d
curl -s http://localhost:8080/healthz   # → OK
```

## 2. Swap base_url (Python)

```diff
- client = openai.OpenAI()
+ client = openai.OpenAI(base_url="http://localhost:8080/v1")
```

## 3. Make a tool call

```python
import openai

client = openai.OpenAI(base_url="http://localhost:8080/v1")
response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "List files in /tmp"}]
)
print(response.choices[0].message.content)
```

## 4. See a deny

```bash
curl -s http://localhost:8080/v1/tools/execute \
  -H 'Content-Type: application/json' \
  -d '{"tool":"unknown_tool","args":{"bad_field":true}}' | jq .
```

**Expected:**

```json
{
  "status": "DENIED",
  "reason_code": "ERR_TOOL_NOT_FOUND",
  "receipt_hash": "sha256:..."
}
```

## 5. Export EvidencePack

```bash
./bin/helm export --evidence ./data/evidence --out pack.tar
ls -la pack.tar
```

**Expected:** deterministic `pack.tar` file.

## 6. Verify offline

```bash
./bin/helm verify --bundle pack.tar
```

**Expected:** `verification: PASS` — no network required, air-gapped safe.

---

**Done.** You now have a governed tool-call loop with cryptographic receipts and offline-verifiable evidence.

Next:

- [QUICKSTART.md](QUICKSTART.md) — annotated 8-step walkthrough
- [DEMO.md](DEMO.md) — copy-paste commands for HN/Reddit
- [SDK docs](sdks/00_INDEX.md) — typed clients for 5 languages
