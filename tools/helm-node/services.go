package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Mindburn-Labs/helm/core/pkg/api"
	"github.com/Mindburn-Labs/helm/core/pkg/artifacts"
	"github.com/Mindburn-Labs/helm/core/pkg/authz"
	"github.com/Mindburn-Labs/helm/core/pkg/boundary"
	"github.com/Mindburn-Labs/helm/core/pkg/budget"
	"github.com/Mindburn-Labs/helm/core/pkg/config"
	"github.com/Mindburn-Labs/helm/core/pkg/credentials"
	helmcrypto "github.com/Mindburn-Labs/helm/core/pkg/crypto"
	"github.com/Mindburn-Labs/helm/core/pkg/evidence"
	"github.com/Mindburn-Labs/helm/core/pkg/guardian"
	"github.com/Mindburn-Labs/helm/core/pkg/identity"
	"github.com/Mindburn-Labs/helm/core/pkg/kernelruntime"
	"github.com/Mindburn-Labs/helm/core/pkg/merkle"
	"github.com/Mindburn-Labs/helm/core/pkg/metering"
	"github.com/Mindburn-Labs/helm/core/pkg/observability"
	"github.com/Mindburn-Labs/helm/core/pkg/runtime/obligation"
	"github.com/Mindburn-Labs/helm/core/pkg/runtime/sandbox"
	"github.com/Mindburn-Labs/helm/core/pkg/sdk"
	"github.com/Mindburn-Labs/helm/core/pkg/tenants"
	"github.com/Mindburn-Labs/helm/core/pkg/tiers"
)

// Services holds all initialized subsystems for the HELM runtime.
type Services struct {
	// --- Infrastructure ---
	Config        *config.Config
	Observability *observability.Provider

	// --- Identity & Auth ---
	Identity identity.KeySet
	Tenants  tenants.Provisioner
	Authz    *authz.Engine
	Creds    *credentials.Handler

	// --- Budget & Metering ---
	BudgetStore    budget.Storage
	BudgetEnforcer *budget.SimpleEnforcer
	Tiers          *tiers.Tier

	// --- Memory ---
	MemoryAPI *api.MemoryService

	// --- Kernel & Execution ---
	BoundaryEnforcer *boundary.PerimeterEnforcer
	MerkleTree       *merkle.MerkleTree
	Sandbox          sandbox.Sandbox
	Obligation       *obligation.ObligationEngine

	// --- UX & Console Subsystems ---
	Evidence *evidence.DefaultExporter

	// --- Operational ---
	SDK *sdk.PackBuilder

	// --- Cross-cutting ---
	KernelRT *kernelruntime.Server

	// --- Security ---
	Guardian *guardian.Guardian
}

// NewServices initializes all subsystems.
func NewServices(ctx context.Context, db *sql.DB, artStore artifacts.Store, meter metering.Meter, logger *slog.Logger) (*Services, error) {
	s := &Services{}

	// --- 1. Config ---
	s.Config = config.Load()
	logger.Info("subsystem ready", "component"," Config loaded")

	// --- 2. Observability ---
	obsCfg := observability.DefaultConfig()
	obs, err := observability.New(ctx, obsCfg)
	if err != nil {
		logger.Warn("Observability init skipped (no OTLP endpoint)", "error", err)
	} else {
		s.Observability = obs
		logger.Info("subsystem ready", "component"," Observability provider initialized")
	}

	// --- 3. Identity ---
	ks, err := identity.NewInMemoryKeySet()
	if err != nil {
		return nil, fmt.Errorf("identity: %w", err)
	}
	s.Identity = ks
	logger.Info("subsystem ready", "component"," Identity KeySet initialized")

	// --- 4. Tenants ---
	prov := tenants.NewPostgresProvisioner(db)
	if err := prov.Init(ctx); err != nil {
		logger.Warn("Tenants table init (may already exist)", "error", err)
	}
	s.Tenants = prov
	logger.Info("subsystem ready", "component"," Tenant Provisioner initialized")

	// --- 5. Authorization ---
	s.Authz = authz.NewEngine()
	logger.Info("subsystem ready", "component"," ReBAC Authorization Engine initialized")

	// --- 6. Credentials ---
	credKeyHex := os.Getenv("CREDENTIALS_ENCRYPTION_KEY")
	if credKeyHex == "" {
		logger.Warn("CREDENTIALS_ENCRYPTION_KEY not set — credentials store DISABLED. Set a 64-char hex key for production.")
	} else {
		encKey, hexErr := hex.DecodeString(credKeyHex)
		if hexErr != nil || len(encKey) != 32 {
			logger.Warn("CREDENTIALS_ENCRYPTION_KEY invalid (must be 64 hex chars / 32 bytes) — credentials store DISABLED")
		} else {
			credStore, err := credentials.NewStore(db, encKey)
			if err != nil {
				logger.Warn("Credentials store init", "error", err)
			} else {
				s.Creds = credentials.NewHandler(credStore)
				logger.Info("subsystem ready", "component"," Credentials Handler initialized")
			}
		}
	}

	// --- 7. Budget ---
	s.BudgetStore = budget.NewPostgresStorage(db)
	s.BudgetEnforcer = budget.NewSimpleEnforcer(s.BudgetStore)
	logger.Info("subsystem ready", "component"," Budget Enforcer initialized (Postgres)")

	// --- 8. Tiers ---
	s.Tiers = &tiers.Free
	logger.Info("subsystem ready", "component"," Tier definitions loaded")

	// --- 9. Memory ---
	s.MemoryAPI = api.NewMemoryService()
	logger.Info("subsystem ready", "component"," Memory Service initialized (stub)")

	// --- 10. Sandbox ---
	sandboxConfig := sandbox.SandboxConfig{
		MemoryLimitBytes: 64 * 1024 * 1024, // 64MB
		CPUTimeLimit:     500 * time.Millisecond,
		NetworkEnabled:   false,
	}
	s.Sandbox, err = sandbox.NewWasiSandbox(ctx, artStore, sandboxConfig)
	if err != nil {
		return nil, fmt.Errorf("sandbox init: %w", err)
	}
	logger.Info("subsystem ready", "component"," Sandbox initialized")

	// --- 11. Boundary ---
	defaultBoundaryPolicy := &boundary.PerimeterPolicy{
		Version:  "1.0",
		PolicyID: "default",
		Name:     "HELM Default Perimeter",
	}
	perimEnforcer, err := boundary.NewPerimeterEnforcer(defaultBoundaryPolicy)
	if err != nil {
		logger.Warn("Boundary enforcer init", "error", err)
	} else {
		s.BoundaryEnforcer = perimEnforcer
	}
	logger.Info("subsystem ready", "component"," Boundary Perimeter Enforcer initialized")

	// --- 12. Merkle ---
	initData := map[string]interface{}{"init": "helm-genesis"}
	mt, _ := merkle.BuildMerkleTree(initData)
	s.MerkleTree = mt
	logger.Info("subsystem ready", "component"," Merkle Tree initialized")

	// --- 13. Obligation ---
	obligationStore := obligation.NewMemoryStore()
	s.Obligation = obligation.NewObligationEngine(obligationStore)
	logger.Info("subsystem ready", "component"," Obligation Engine initialized")

	// --- 14. Evidence ---
	evidenceKey := os.Getenv("EVIDENCE_SIGNING_KEY")
	if evidenceKey == "" {
		evidenceKey = "helm-evidence-bundle"
		logger.Warn("EVIDENCE_SIGNING_KEY not set — using default seed (not safe for production)")
	}
	evidenceSigner, err := helmcrypto.NewEd25519Signer(evidenceKey)
	if err != nil {
		return nil, fmt.Errorf("evidence signer init: %w", err)
	}
	s.Evidence = evidence.NewExporter(evidenceSigner, evidenceSigner.KeyID)
	logger.Info("subsystem ready", "component"," Evidence Exporter initialized")

	// --- 15. SDK ---
	s.SDK = sdk.NewPack("helm-sdk", "1.0.0", "HELM SDK")
	logger.Info("subsystem ready", "component"," SDK initialized")

	// --- 16. Kernel Runtime ---
	s.KernelRT = kernelruntime.New(s.Config)
	logger.Info("subsystem ready", "component"," KernelRuntime initialized")

	logger.Info("subsystem ready", "component"," All subsystems initialized successfully")
	return s, nil
}
