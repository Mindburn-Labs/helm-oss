package evidencepack

import (
	"encoding/json"
	"fmt"
	"time"
)

// Builder constructs evidence packs deterministically.
// All entries are collected, sorted, hashed, and then archived.
type Builder struct {
	packID     string
	actorDID   string
	intentID   string
	policyHash string
	createdAt  time.Time
	entries    map[string]entryData
}

type entryData struct {
	content     []byte
	contentType string
}

// NewBuilder creates a new evidence pack builder.
func NewBuilder(packID, actorDID, intentID, policyHash string) *Builder {
	return &Builder{
		packID:     packID,
		actorDID:   actorDID,
		intentID:   intentID,
		policyHash: policyHash,
		createdAt:  time.Now().UTC(),
		entries:    make(map[string]entryData),
	}
}

// WithCreatedAt overrides the creation timestamp (for deterministic testing).
func (b *Builder) WithCreatedAt(t time.Time) *Builder {
	b.createdAt = t
	return b
}

// AddReceipt adds a receipt as a JSON file to the pack.
func (b *Builder) AddReceipt(name string, receipt interface{}) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal receipt %s: %w", name, err)
	}
	b.entries["receipts/"+name+".json"] = entryData{
		content:     data,
		contentType: "application/json",
	}
	return nil
}

// AddPolicyDecision adds a policy decision document to the pack.
func (b *Builder) AddPolicyDecision(name string, decision interface{}) error {
	data, err := json.MarshalIndent(decision, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal policy decision %s: %w", name, err)
	}
	b.entries["policy/"+name+".json"] = entryData{
		content:     data,
		contentType: "application/json",
	}
	return nil
}

// AddToolTranscript adds a tool execution transcript.
func (b *Builder) AddToolTranscript(name string, transcript interface{}) error {
	data, err := json.MarshalIndent(transcript, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal transcript %s: %w", name, err)
	}
	b.entries["transcripts/"+name+".json"] = entryData{
		content:     data,
		contentType: "application/json",
	}
	return nil
}

// AddRawEntry adds raw bytes as a file in the pack.
func (b *Builder) AddRawEntry(path, contentType string, data []byte) {
	b.entries[path] = entryData{
		content:     data,
		contentType: contentType,
	}
}

// Build constructs the manifest and returns all entries ready for archiving.
func (b *Builder) Build() (*Manifest, map[string][]byte, error) {
	if len(b.entries) == 0 {
		return nil, nil, fmt.Errorf("evidence pack has no entries")
	}

	var manifestEntries []ManifestEntry
	contentMap := make(map[string][]byte, len(b.entries))

	for path, entry := range b.entries {
		hash := HashContent(entry.content)
		manifestEntries = append(manifestEntries, ManifestEntry{
			Path:        path,
			ContentHash: hash,
			Size:        int64(len(entry.content)),
			ContentType: entry.contentType,
		})
		contentMap[path] = entry.content
	}

	manifest := &Manifest{
		Version:    ManifestVersion,
		PackID:     b.packID,
		CreatedAt:  b.createdAt,
		ActorDID:   b.actorDID,
		IntentID:   b.intentID,
		PolicyHash: b.policyHash,
		Entries:    manifestEntries,
	}

	manifestHash, err := ComputeManifestHash(manifest)
	if err != nil {
		return nil, nil, fmt.Errorf("compute manifest hash: %w", err)
	}
	manifest.ManifestHash = manifestHash

	// Add the manifest itself to the content map
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal manifest: %w", err)
	}
	contentMap["manifest.json"] = manifestJSON

	return manifest, contentMap, nil
}
