.PHONY: build test test-race test-sdk-ts test-sdk-py test-all crucible lint proxy clean docker demo demo-down release-binaries verify-boundary onboard demo-cli mcp-pack mcp-install release-all

# ── Build ──────────────────────────────────────────────
build:
	cd core && go build -o ../bin/helm ./cmd/helm/
	go build -C apps/helm-node -o ../../bin/helm-node .
	@echo "✅ bin/helm + bin/helm-node"

# ── Test ───────────────────────────────────────────────
test:
	cd core && go test ./pkg/... -count=1

test-race:
	cd core && go test ./pkg/... -count=1 -race

test-sdk-ts:
	cd sdk/ts && npm test -- --run

test-sdk-py:
	cd sdk/python && pip install -q '.[dev]' && pytest -v

test-cli:
	cd packages/mindburn-helm-cli && npm test -- --run

verify-fixtures:
	@echo "Verifying golden fixtures..."
	@cd packages/mindburn-helm-cli && npx tsx ../../../scripts/verify-fixture-roots.mts
	@echo "Golden fixture roots verified"

test-all: test test-sdk-ts test-sdk-py test-cli verify-fixtures

# ── Crucible (adversarial + conformance + use cases) ──
crucible: test
	bash scripts/usecases/run_all.sh
	@echo "✅ Crucible passed"

# ── Lint ───────────────────────────────────────────────
lint:
	cd core && go vet ./...

# ── Proxy (quick-start) ───────────────────────────────
proxy: build
	./bin/helm proxy --upstream https://api.openai.com/v1

# ── Docker ─────────────────────────────────────────────
docker:
	docker build -t helm:latest .

docker-up:
	docker compose up -d

# ── Demo (DigitalOcean / any Docker host) ──────────────
demo:
	docker compose -f docker-compose.demo.yml up -d --build
	@echo ""
	@echo "✅ HELM demo running"
	@echo "   API:    http://localhost:8080"
	@echo "   Health: http://localhost:8080/health"
	@echo ""

demo-down:
	docker compose -f docker-compose.demo.yml down

demo-reset:
	bash deploy/demo-reset.sh

# ── SBOM ───────────────────────────────────────────────
sbom: build
	bash scripts/ci/generate_sbom.sh
	@echo "✅ sbom.json (CycloneDX) + deps.txt generated"

# ── Provenance ─────────────────────────────────────────
provenance:
	cd core && CGO_ENABLED=0 go build -ldflags="-s -w \
		-X main.version=0.1.1 \
		-X main.commit=$$(git rev-parse HEAD) \
		-X main.buildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
		-o ../bin/helm ./cmd/helm/
	shasum -a 256 bin/helm > bin/helm.sha256
	@echo "✅ Provenance build: bin/helm + bin/helm.sha256"

# ── Release Binaries (cross-compile) ──────────────────
VERSION ?= 0.2.0
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

release-binaries:
	@echo "Building release binaries (v$(VERSION))..."
	cd core && GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../bin/helm-linux-amd64 ./cmd/helm/
	cd core && GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../bin/helm-linux-arm64 ./cmd/helm/
	cd core && GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../bin/helm-darwin-amd64 ./cmd/helm/
	cd core && GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../bin/helm-darwin-arm64 ./cmd/helm/
	cd core && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../bin/helm-windows-amd64.exe ./cmd/helm/
	cd bin && shasum -a 256 helm-* > SHA256SUMS.txt
	@echo "✅ Release binaries + SHA256SUMS.txt (v$(VERSION))"

# ── Quickstart (one-command onboard + demo) ────────────
onboard: build
	./bin/helm onboard --yes

demo-cli: build
	./bin/helm demo company --template starter

# ── MCP ────────────────────────────────────────────────
mcp-pack: build
	./bin/helm mcp pack --client claude-desktop --out dist/helm.mcpb
	@echo "✅ dist/helm.mcpb (MCPB bundle)"

mcp-install: build
	./bin/helm mcp install --client claude-code
	@echo "✅ helm-mcp-plugin/ (Claude Code plugin)"

# ── Full Release (all artifacts) ───────────────────────
release-all: release-binaries sbom mcp-pack
	@mkdir -p dist
	cp bin/helm-* dist/
	cp bin/SHA256SUMS.txt dist/
	@echo "✅ Full release in dist/ (binaries + SBOM + MCPB)"

# ── Repo Boundary (OSS ↔ Commercial) ──────────────────
verify-boundary:
	bash tools/verify-boundary.sh
	@echo "✅ All protected paths in sync"

# ── Clean ──────────────────────────────────────────────
clean:
	rm -rf bin/ dist/ sbom.json deps.txt helm-mcp-plugin/

