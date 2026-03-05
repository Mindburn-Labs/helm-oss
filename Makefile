.PHONY: build test test-race test-sdk-ts test-sdk-py test-all crucible lint proxy clean docker demo demo-down release-binaries verify-boundary

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
release-binaries:
	@echo "Building release binaries..."
	cd core && GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/helm-linux-amd64 ./cmd/helm/
	cd core && GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/helm-linux-arm64 ./cmd/helm/
	cd core && GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/helm-darwin-amd64 ./cmd/helm/
	cd core && GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/helm-darwin-arm64 ./cmd/helm/
	cd bin && shasum -a 256 helm-* > SHA256SUMS.txt
	@echo "✅ Release binaries + SHA256SUMS.txt"

# ── Repo Boundary (OSS ↔ Commercial) ──────────────────
verify-boundary:
	bash tools/verify-boundary.sh
	@echo "✅ All protected paths in sync"

# ── Clean ──────────────────────────────────────────────
clean:
	rm -rf bin/ sbom.json deps.txt
