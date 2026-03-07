package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/crypto"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/manifest"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/metering"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/registry"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store/ledger"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/tenants"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/tiers"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: bootstrap <db_url>")
	}
	dbURL := os.Args[1]

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// 1. Initialize Schemas
	log.Println("[bootstrap] Initializing Schemas...")

	// Ledger
	lgr := ledger.NewPostgresLedger(db)
	if err := lgr.Init(ctx); err != nil {
		log.Fatalf("Failed to init ledger: %v", err)
	}

	// Metering
	meter := metering.NewPostgresMeter(db)
	if err := meter.Init(ctx); err != nil {
		log.Fatalf("Failed to init metering: %v", err)
	}

	// Registry
	reg := registry.NewPostgresRegistry(db)
	if err := reg.Init(ctx); err != nil {
		log.Fatalf("Failed to init registry: %v", err)
	}

	// Tenants
	prov := tenants.NewPostgresProvisioner(db)
	if err := prov.Init(ctx); err != nil {
		log.Fatalf("Failed to init tenants: %v", err)
	}

	log.Println("[bootstrap] Schemas Initialized.")

	var tenantID string

	// 2. Bootstrap helm-org Tenant
	// Check if exists
	existing, err := prov.GetByEmail(ctx, "admin@helm.sh")
	if err == nil && existing != nil {
		log.Printf("[bootstrap] tenant 'helm-org' (admin@helm.sh) already exists (ID: %s)\n", existing.ID)
		tenantID = existing.ID
	} else {
		log.Println("[bootstrap] Creating 'helm-org' tenant...")
		req := tenants.CreateRequest{
			Email: "admin@helm.sh",
			Metadata: map[string]any{
				"org_name": "HELM Organization",
				"type":     "internal",
			},
		}

		tenant, rawKey, err := prov.Create(ctx, req)
		if err != nil {
			log.Fatalf("Failed to create tenant: %v", err)
		}
		log.Printf("[bootstrap] Created 'helm-org' tenant!\n")
		log.Printf("  ID: %s\n", tenant.ID)
		log.Printf("  API Key: %s\n", rawKey)
		tenantID = tenant.ID
	}

	// Update to Enterprise Tier (Always ensure)
	log.Println("[bootstrap] Ensuring Enterprise Tier...")
	_, err = db.ExecContext(ctx, "UPDATE tenants SET tier_id = $1, email_verified = true WHERE id = $2", tiers.TierEnterprise, tenantID)
	if err != nil {
		log.Printf(">> Warning: Failed to upgrade tier: %v\n", err)
	}

	// 3. Seed Ops Packs
	log.Println("[bootstrap] Seeding Ops Packs...")
	packsDir := "/app/packs" // Default container path
	// override if env set
	if p := os.Getenv("PACKS_DIR"); p != "" {
		packsDir = p
	}

	if err := seedPacks(reg, packsDir); err != nil {
		log.Printf(">> Warning: Failed to seed packs: %v\n", err)
	} else {
		log.Println("[bootstrap] Ops Packs Seeded.")
	}

	log.Println("[bootstrap] Bootstrap Complete.")
}

func seedPacks(reg registry.Registry, root string) error {
	// Initialize System Signer (Boot Key) — MUST be provided via environment
	bootKey := os.Getenv("SYSTEM_BOOT_KEY")
	if bootKey == "" {
		return fmt.Errorf("SYSTEM_BOOT_KEY environment variable is required for pack signing (must be a stable secret)")
	}
	signer, err := crypto.NewEd25519Signer(bootKey)
	if err != nil {
		return fmt.Errorf("failed to create system signer: %w", err)
	}

	entries, err := os.ReadDir(root + "/ops")
	if err != nil {
		return err // might not exist
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := fmt.Sprintf("%s/ops/%s/manifest.json", root, e.Name())
		//nolint:gosec // G304: Local manifest read
		content, err := os.ReadFile(manifestPath)
		if err != nil {
			log.Printf("   Skipping %s: no manifest.json\n", e.Name())
			continue
		}

		var m manifest.Module
		if err := json.Unmarshal(content, &m); err != nil {
			log.Printf("   Skipping %s: invalid manifest: %v\n", e.Name(), err)
			continue
		}

		// Create bundle
		bundle := &manifest.Bundle{
			Manifest:   m,
			CompiledAt: time.Now().UTC().String(),
		}

		// SIGN THE BUNDLE
		manifestJSON, _ := json.Marshal(m)
		sig, err := signer.Sign(manifestJSON)
		if err != nil {
			log.Printf("   Failed to sign bundle %s: %v\n", m.Name, err)
			continue
		}
		bundle.Signature = sig

		if err := reg.Register(bundle); err != nil {
			// Ignore if already exists (idempotent), but log warning
			log.Printf("   Warning: Failed to register %s: %v\n", m.Name, err)
		} else {
			log.Printf("  Registered and Signed %s@%s (Sig: %s...)\n", m.Name, m.Version, bundle.Signature[:10])
		}
	}
	return nil
}
