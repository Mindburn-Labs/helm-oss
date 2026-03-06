// Package bundles implements policy bundle loading, signing, and verification.
//
// Per HELM Standard v1.2 — Policy Bundles (Phase 2-4):
//
//	Policy bundles are external YAML files defining governance rules,
//	compliance profiles, and jurisdiction constraints. They are loaded
//	at runtime, enabling compliance-as-code without kernel recompilation.
//
//	Phase 2: Bundle loader (this file)
//	Phase 3: Bundle registry
//	Phase 4: Bundle signing (bundle_signer.go)
//	Phase 5: Bundle distribution
package bundles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// PolicyBundle is a loadable policy definition.
type PolicyBundle struct {
	// Header
	ID          string    `yaml:"id"          json:"id"`
	Name        string    `yaml:"name"        json:"name"`
	Version     string    `yaml:"version"     json:"version"`
	Description string    `yaml:"description" json:"description"`
	Author      string    `yaml:"author"      json:"author"`
	CreatedAt   time.Time `yaml:"created_at"  json:"created_at"`

	// Classification
	Kind     BundleKind `yaml:"kind"        json:"kind"` // compliance, jurisdiction, industry
	Regime   string     `yaml:"regime"      json:"regime,omitempty"`
	Region   string     `yaml:"region"      json:"region,omitempty"`
	Industry string     `yaml:"industry"    json:"industry,omitempty"`

	// Policy Rules
	Rules []PolicyRule `yaml:"rules" json:"rules"`

	// Metadata
	ContentHash string `yaml:"-" json:"content_hash"`
}

// BundleKind classifies a policy bundle.
type BundleKind string

const (
	BundleCompliance   BundleKind = "compliance"
	BundleJurisdiction BundleKind = "jurisdiction"
	BundleIndustry     BundleKind = "industry"
)

// PolicyRule is a single rule within a bundle.
type PolicyRule struct {
	ID         string            `yaml:"id"          json:"id"`
	Effect     string            `yaml:"effect"      json:"effect"` // allow, deny, escalate
	Condition  string            `yaml:"condition"   json:"condition"`
	Engine     string            `yaml:"engine"      json:"engine"` // cel, rego, cedar
	Priority   int               `yaml:"priority"    json:"priority"`
	ReasonCode string            `yaml:"reason_code" json:"reason_code,omitempty"`
	Tags       []string          `yaml:"tags"        json:"tags,omitempty"`
	Metadata   map[string]string `yaml:"metadata"    json:"metadata,omitempty"`
}

// BundleLoader loads policy bundles from the filesystem.
type BundleLoader struct {
	bundles []*PolicyBundle
}

// NewBundleLoader creates a new bundle loader.
func NewBundleLoader() *BundleLoader {
	return &BundleLoader{
		bundles: make([]*PolicyBundle, 0),
	}
}

// LoadFromFile reads a single policy bundle YAML file.
func (l *BundleLoader) LoadFromFile(path string) (*PolicyBundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundle %s: %w", path, err)
	}

	var bundle PolicyBundle
	if err := yaml.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse bundle %s: %w", path, err)
	}

	// Validate required fields
	if bundle.ID == "" {
		return nil, fmt.Errorf("bundle %s: missing required field 'id'", path)
	}
	if bundle.Name == "" {
		return nil, fmt.Errorf("bundle %s: missing required field 'name'", path)
	}
	if bundle.Kind == "" {
		return nil, fmt.Errorf("bundle %s: missing required field 'kind'", path)
	}

	// Compute content hash
	canonical, err := json.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("bundle %s: cannot compute hash: %w", path, err)
	}
	h := sha256.Sum256(canonical)
	bundle.ContentHash = "sha256:" + hex.EncodeToString(h[:])

	l.bundles = append(l.bundles, &bundle)
	return &bundle, nil
}

// LoadFromDir scans a directory for *.policy.yaml files and loads them all.
func (l *BundleLoader) LoadFromDir(dir string) ([]*PolicyBundle, error) {
	var loaded []*PolicyBundle

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".policy.yaml") || strings.HasSuffix(name, ".policy.yml") {
			bundle, err := l.LoadFromFile(filepath.Join(dir, name))
			if err != nil {
				return nil, err // Fail-closed: any invalid bundle aborts the load
			}
			loaded = append(loaded, bundle)
		}
	}

	return loaded, nil
}

// All returns all loaded bundles.
func (l *BundleLoader) All() []*PolicyBundle {
	return l.bundles
}

// ByKind filters loaded bundles by kind.
func (l *BundleLoader) ByKind(kind BundleKind) []*PolicyBundle {
	var result []*PolicyBundle
	for _, b := range l.bundles {
		if b.Kind == kind {
			result = append(result, b)
		}
	}
	return result
}
