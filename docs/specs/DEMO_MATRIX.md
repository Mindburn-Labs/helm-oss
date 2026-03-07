# HELM Mandatory Demo Matrix

> 10 combinations that MUST pass before every release.

## Demo Matrix

| #   | Client         | Framework              | Auth         | Jurisdiction | Expected Artifacts          |
| --- | -------------- | ---------------------- | ------------ | ------------ | --------------------------- |
| 1   | Gemini CLI     | Google ADK             | Google OAuth | Global       | Proof Report + EvidencePack |
| 2   | Claude Code    | OpenAI Agents (Python) | MCP Header   | EU GDPR      | Proof Report + EvidencePack |
| 3   | Claude Desktop | LangGraph              | MCP Header   | US Baseline  | Proof Report + EvidencePack |
| 4   | Cursor         | Mastra                 | MCP Header   | Global       | Proof Report + EvidencePack |
| 5   | VS Code        | LangChain              | MCP Header   | UK DPA       | Proof Report + EvidencePack |
| 6   | Qwen Code      | OpenAI Agents (JS)     | MCP Header   | CN PIPL      | Proof Report + EvidencePack |
| 7   | Codex CLI      | Direct SDK (Go)        | API Key      | JP APPI      | Proof Report + EvidencePack |
| 8   | Windsurf       | Direct SDK (TS)        | MCP Header   | AU Privacy   | Proof Report + EvidencePack |
| 9   | Continue.dev   | AutoGen                | MCP Header   | BR LGPD      | Proof Report + EvidencePack |
| 10  | Zed            | Direct SDK (Python)    | MCP Header   | CA PIPEDA    | Proof Report + EvidencePack |

## Per-Demo Requirements

Each demo MUST produce:

1. **Proof Report** (`proof-report.html`) — verifiable HTML with embedded JSON capsule
2. **EvidencePack** (`evidence-pack.tar`) — receipts + decisions + active bundles
3. **Verify Command** — `helm verify <evidence-pack>` returns exit 0
4. **Compatibility Metadata** — written to `compatibility-registry.json`

## CI Gating

```yaml
# .github/workflows/demo-matrix.yml
jobs:
  demo-matrix:
    strategy:
      matrix:
        demo: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    steps:
      - run: helm demo run --matrix ${{ matrix.demo }}
      - run: helm verify dist/evidence-pack-${{ matrix.demo }}.tar
      - uses: actions/upload-artifact@v4
        with:
          name: demo-${{ matrix.demo }}
          path: |
            dist/proof-report-${{ matrix.demo }}.html
            dist/evidence-pack-${{ matrix.demo }}.tar
```

## Demo Runner

```bash
# Run a single demo
helm demo run --matrix 1

# Run all 10 demos
helm demo run --matrix all

# Run with custom jurisdiction
helm demo run --matrix 2 --jurisdiction sg-pdpa
```
