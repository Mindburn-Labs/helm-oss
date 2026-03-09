# HELM Compatibility Matrix

## Tiers

| Tier           | Requirements                                 | Badge         |
| -------------- | -------------------------------------------- | ------------- |
| **Compatible** | L1 pass                                      | 🟢 Compatible |
| **Verified**   | L1 + L2 + strict preflight + receipt binding | 🔵 Verified   |
| **Sovereign**  | Verified + L3 degraded-path adversarial      | 🟣 Sovereign  |

## Current Matrix

### Sandbox Providers

| Provider    | Version | Compatible | Verified | Sovereign | Notes                            |
| ----------- | ------- | :--------: | :------: | :-------: | -------------------------------- |
| Mock        | 1.0.0   |     ✅     |    ✅    |    ✅     | Built-in, deterministic          |
| OpenSandbox | latest  |     ✅     |    ✅    |    🔲     | Full preflight + receipt binding |
| E2B         | latest  |     ✅     |    ✅    |    🔲     | Full preflight + receipt binding |
| Daytona     | latest  |     ✅     |    🔲    |    🔲     | Basic integration                |

### Orchestrator Adapters

| Framework          | Language   | Compatible | Verified | Notes                          |
| ------------------ | ---------- | :--------: | :------: | ------------------------------ |
| OpenAI Agents SDK  | Python     |     ✅     |    ✅    | Full governance + EvidencePack |
| OpenAI Agents SDK  | TypeScript |     ✅     |    ✅    | + Responses WS mode            |
| MS Agent Framework | Python     |     ✅     |    ✅    | RC adapter                     |
| MS Agent Framework | .NET       |     ✅     |    🔲    | Minimal example                |
| LangChain          | Python     |     ✅     |    ✅    | Tool wrapper                   |
| Mastra             | TypeScript |     ✅     |    ✅    | + sandbox provider selection   |

### MCP Clients

| Client         | Transport      | Compatible | Notes                        |
| -------------- | -------------- | :--------: | ---------------------------- |
| Claude Code    | stdio (plugin) |     ✅     | Auto-start via `.mcp.json`   |
| Claude Desktop | stdio (.mcpb)  |     ✅     | Cross-platform binary bundle |
| Windsurf       | stdio / HTTP   |     ✅     |                              |
| Codex          | stdio          |     ✅     | `codex mcp add`              |
| VS Code        | stdio          |     ✅     | `.vscode/settings.json`      |
| Cursor         | stdio          |     ✅     | `.cursor/mcp.json`           |

## CI

The compatibility matrix is generated weekly by `.github/workflows/compatibility_matrix.yml`.

```bash
# Run conformance locally
helm conform --level L2 --json

# Run sandbox conformance
helm sandbox conform --provider opensandbox --tier verified --json
```
