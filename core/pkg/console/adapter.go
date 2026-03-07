package console

import (
	"context"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/manifest"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/pack"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/registry"
)

// RegistryAdapter bridges the Bundle Registry to the PackRegistry interface.
type RegistryAdapter struct {
	reg registry.Registry
}

// NewRegistryAdapter creates a new adapter.
func NewRegistryAdapter(reg registry.Registry) *RegistryAdapter {
	return &RegistryAdapter{reg: reg}
}

// GetPack retrieves a pack by ID (mapped from Bundle Name).
func (a *RegistryAdapter) GetPack(ctx context.Context, id string) (*pack.Pack, error) {
	bundle, err := a.reg.Get(id)
	if err != nil {
		return nil, err
	}

	// Convert Bundle to Pack
	capabilities := make([]string, len(bundle.Manifest.Capabilities))
	for i, c := range bundle.Manifest.Capabilities {
		capabilities[i] = c.Name
	}

	p := &pack.Pack{
		PackID: bundle.Manifest.Name, // Using Name as ID for compatibility
		Manifest: pack.PackManifest{
			PackID:       bundle.Manifest.Name,
			Version:      bundle.Manifest.Version,
			Name:         bundle.Manifest.Name,
			Description:  bundle.Manifest.Description,
			Capabilities: capabilities,
		},
		CreatedAt: time.Now(), // Bundle doesn't store CreatedAt in struct easily accessible? using Now for adapter
	}

	// Signature mapping
	if bundle.Signature != "" {
		p.Manifest.Signatures = []pack.Signature{
			{
				SignerID:  "bundle-registry",
				Signature: bundle.Signature,
				SignedAt:  time.Now(), // Unknown
				Algorithm: "ed25519",  // Assumed
			},
		}
	}

	return p, nil
}

// FindByCapability finds packs by capability.
// Note: O(N) scan over bundles. Acceptable at current registry scale.
func (a *RegistryAdapter) FindByCapability(ctx context.Context, capability string) ([]pack.Pack, error) {
	bundles := a.reg.List()
	var packs []pack.Pack

	for _, b := range bundles {
		for _, cap := range b.Manifest.Capabilities {
			if cap.Name == capability {
				// Convert to Pack (reused logic)
				caps := make([]string, len(b.Manifest.Capabilities))
				for i, c := range b.Manifest.Capabilities {
					caps[i] = c.Name
				}
				p := pack.Pack{
					PackID: b.Manifest.Name,
					Manifest: pack.PackManifest{
						PackID:       b.Manifest.Name,
						Version:      b.Manifest.Version,
						Name:         b.Manifest.Name,
						Description:  b.Manifest.Description,
						Capabilities: caps,
					},
				}
				packs = append(packs, p)
				break
			}
		}
	}
	return packs, nil
}

// ListVersions lists versions.
// Bundle registry returns latest version via Get, and all latest via List.
// We might not have access to history in this interface yet.
func (a *RegistryAdapter) ListVersions(ctx context.Context, packName string) ([]pack.PackVersion, error) {
	// Stub implementation: Returns current version only
	bundle, err := a.reg.Get(packName)
	if err != nil {
		return nil, err
	}

	return []pack.PackVersion{
		{
			PackName:   bundle.Manifest.Name,
			Version:    bundle.Manifest.Version,
			ReleasedAt: time.Now(),
		},
	}, nil
}

// PublishPack adapts pack.Pack to manifest.Bundle and registers it.
func (a *RegistryAdapter) PublishPack(ctx context.Context, p *pack.Pack) error {
	// Map Capabilities
	caps := make([]manifest.CapabilityConfig, len(p.Manifest.Capabilities))
	for i, c := range p.Manifest.Capabilities {
		caps[i] = manifest.CapabilityConfig{
			Name:        c,
			Description: "Auto-mapped from Pack",
		}
	}

	bundle := &manifest.Bundle{
		Manifest: manifest.Module{
			Name:         p.Manifest.Name,
			Version:      p.Manifest.Version,
			Description:  p.Manifest.Description,
			Capabilities: caps,
		},
		CompiledAt: p.CreatedAt.Format(time.RFC3339),
		PowerDelta: 10, // Default for now
	}

	// Signatures: Bundle supports a single signature slot.
	// Take the first Ed25519 signature if available.
	if len(p.Manifest.Signatures) > 0 {
		bundle.Signature = p.Manifest.Signatures[0].Signature
	}

	return a.reg.Register(bundle)
}
