package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfile_US(t *testing.T) {
	profilesDir := locateProfiles(t)
	p, err := LoadProfile(profilesDir, "us")
	if err != nil {
		t.Fatalf("LoadProfile(us): %v", err)
	}
	if p.Name != "United States" {
		t.Errorf("expected name 'United States', got %q", p.Name)
	}
	if p.Encryption != "AES-256-GCM" {
		t.Errorf("expected AES-256-GCM, got %q", p.Encryption)
	}
	if p.IsIslandMode() {
		t.Error("US should not be island mode")
	}
}

func TestLoadProfile_EU_GDPR(t *testing.T) {
	profilesDir := locateProfiles(t)
	p, err := LoadProfile(profilesDir, "eu")
	if err != nil {
		t.Fatalf("LoadProfile(eu): %v", err)
	}
	if p.PIIHandling != "strict" {
		t.Errorf("EU should have strict PII handling, got %q", p.PIIHandling)
	}
	if !p.RightToErasure {
		t.Error("EU should have right to erasure")
	}
	if !p.Ceremony.RequireChallenge {
		t.Error("EU should require ceremony challenge")
	}
}

func TestLoadProfile_RU_IslandMode(t *testing.T) {
	profilesDir := locateProfiles(t)
	p, err := LoadProfile(profilesDir, "ru")
	if err != nil {
		t.Fatalf("LoadProfile(ru): %v", err)
	}
	if !p.IsIslandMode() {
		t.Error("RU should default to island mode")
	}
	if !p.CryptoPolicy.RequireNationalCrypto {
		t.Error("RU should require national crypto")
	}
}

func TestLoadProfile_CN_SM4(t *testing.T) {
	profilesDir := locateProfiles(t)
	p, err := LoadProfile(profilesDir, "cn")
	if err != nil {
		t.Fatalf("LoadProfile(cn): %v", err)
	}
	if p.Encryption != "SM4" {
		t.Errorf("CN should use SM4, got %q", p.Encryption)
	}
	if !p.IsIslandMode() {
		t.Error("CN should default to island mode")
	}
}

func TestLoadAllProfiles(t *testing.T) {
	profilesDir := locateProfiles(t)
	profiles, err := LoadAllProfiles(profilesDir)
	if err != nil {
		t.Fatalf("LoadAllProfiles: %v", err)
	}
	if len(profiles) < 4 {
		t.Errorf("expected at least 4 profiles, got %d", len(profiles))
	}
	for code, p := range profiles {
		if p.Name == "" {
			t.Errorf("profile %s has empty name", code)
		}
	}
}

func TestIsAllowed_Allowlist(t *testing.T) {
	p := &RegionalProfile{
		Networking: NetworkingConfig{
			OutboundMode: "allowlist",
			Allowlist:    []string{"api.openai.com"},
		},
	}
	if !p.IsAllowed("api.openai.com") {
		t.Error("should allow api.openai.com")
	}
	if p.IsAllowed("evil.com") {
		t.Error("should deny evil.com")
	}
}

func TestIsAllowed_IslandMode(t *testing.T) {
	p := &RegionalProfile{
		Networking: NetworkingConfig{
			IslandMode: true,
		},
	}
	if p.IsAllowed("api.openai.com") {
		t.Error("island mode should deny all")
	}
}

func locateProfiles(t *testing.T) string {
	t.Helper()
	// Try to find profiles directory relative to this test file
	candidates := []string{
		"profiles",
		"../config/profiles",
		filepath.Join(os.Getenv("GOPATH"), "src/github.com/Mindburn-Labs/helm-oss/core/pkg/config/profiles"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Try to find from working directory
	wd, _ := os.Getwd()
	p := filepath.Join(wd, "profiles")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	t.Skip("profiles directory not found")
	return ""
}
