---
title: PORTS
---

# HELM Port Layout

| Port | Service | Context |
|------|---------|---------|
| **8080** | Kernel API (OpenAI-compatible, MCP gateway) | All modes |
| **8081** | Health endpoint (`/healthz`, compat `/health`) | All modes |
| **9090** | Standalone proxy (when running `helm proxy`) | Standalone only |

## Notes

- **Docker Compose**: exposes `8080` and `8081`. The proxy runs inside the kernel on `8080`.
- **Standalone proxy** (`make proxy` / `helm proxy`): listens on `9090`, forwards to upstream.
- **Quickstart examples** use `http://localhost:8080` (Docker) or `http://localhost:9090/v1` (standalone proxy).
