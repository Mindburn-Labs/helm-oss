package arc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/artifacts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/metering"
)

// IngestionService orchestrates the ingestion of authoritative sources.
type IngestionService struct {
	store      artifacts.Store
	meter      metering.Meter
	connectors map[string]SourceConnector
}

// NewIngestionService creates a new IngestionService.
func NewIngestionService(store artifacts.Store, meter metering.Meter) *IngestionService {
	return &IngestionService{
		store:      store,
		meter:      meter,
		connectors: make(map[string]SourceConnector),
	}
}

// RegisterConnector adds a connector to the service.
func (s *IngestionService) RegisterConnector(c SourceConnector) {
	s.connectors[c.ID()] = c
}

// Ingest fetches a regulation from a source, stores it, and returns a receipt.
func (s *IngestionService) Ingest(ctx context.Context, sourceID string, externalID string) (*IngestionReceipt, *SourceArtifact, error) {
	connector, ok := s.connectors[sourceID]
	if !ok {
		return nil, nil, fmt.Errorf("connector not found: %s", sourceID)
	}

	// F11: Validate ExternalID
	if externalID == "" {
		return nil, nil, fmt.Errorf("externalID must not be empty")
	}
	if len(externalID) > 512 {
		return nil, nil, fmt.Errorf("externalID exceeds maximum length (512)")
	}

	// 1. Fetch
	// Note: Connectors manage their own rate limits via BaseConnector
	data, mimeType, meta, err := connector.Fetch(ctx, externalID)
	if err != nil {
		return &IngestionReceipt{
			ReceiptID: uuid.New().String(),
			SourceID:  sourceID,
			Status:    "ERROR",
			Timestamp: time.Now().UTC(),
			Error:     err.Error(),
		}, nil, err
	}

	// 2. Store (Content-Addressed)
	hash, err := s.store.Store(ctx, data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to store artifact: %w", err)
	}

	// 3. Create Artifact Record
	// F8: Use connector version
	connVersion := "1.0.0"
	if bc, ok := connector.(interface{ Version() string }); ok {
		connVersion = bc.Version()
	}

	artifact := &SourceArtifact{
		ArtifactID:      uuid.New().String(),
		ContentHash:     hash,
		SourceID:        sourceID,
		ExternalID:      externalID,
		IngestedAt:      time.Now().UTC(),
		MimeType:        mimeType,
		ConnectorConfig: meta,
		Provenance: &SourceProvenance{
			ConnectorName:    sourceID,
			ConnectorVersion: connVersion,
			RetrievalMethod:  "api",
			RetrievedAt:      time.Now().UTC(),
		},
	}

	// 4. F9: Cost Model based on TrustClass
	bytesIngested := int64(len(data))
	kb := float64(bytesIngested) / 1024.0
	var costPerKB float64
	switch connector.TrustClass() {
	case TrustClassOfficial:
		costPerKB = 0.001 // Official APIs are cheapest
	case TrustClassPartner:
		costPerKB = 0.005 // Partner feeds mid-range
	case TrustClassCommunity:
		costPerKB = 0.010 // Community sources most expensive (quality overhead)
	default:
		costPerKB = 0.010
	}
	cost := kb * costPerKB

	// F10: Use registered EventIngestion type
	if s.meter != nil {
		_ = s.meter.Record(ctx, metering.Event{
			TenantID:  "system",
			EventType: metering.EventIngestion,
			Quantity:  bytesIngested,
			Timestamp: time.Now().UTC(),
			Metadata: map[string]any{
				"source_id": sourceID,
				"url":       externalID,
				"cost_usd":  cost,
			},
		})
	}

	// 5. Receipt
	receipt := &IngestionReceipt{
		ReceiptID:     uuid.New().String(),
		SourceID:      sourceID,
		ArtifactID:    artifact.ArtifactID,
		Status:        "SUCCESS",
		BytesIngested: bytesIngested,
		CostUSD:       cost,
		Timestamp:     time.Now().UTC(),
	}

	return receipt, artifact, nil
}
