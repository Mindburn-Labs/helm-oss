package arc_test

import (
	"context"
	"os"
	"testing"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/arc"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/arc/connectors"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/artifacts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/metering"
)

// MockMeter implements metering.Meter for testing.
type MockMeter struct{}

func (m *MockMeter) Record(ctx context.Context, event metering.Event) error         { return nil }
func (m *MockMeter) RecordBatch(ctx context.Context, events []metering.Event) error { return nil }
func (m *MockMeter) GetUsage(ctx context.Context, tenantID string, period metering.Period) (*metering.Usage, error) {
	return nil, nil
}
func (m *MockMeter) GetUsageByType(ctx context.Context, tenantID string, eventType metering.EventType, period metering.Period) (int64, error) {
	return 0, nil
}

func TestIngestionWeb(t *testing.T) {
	// 1. Setup Store
	tmpDir, err := os.MkdirTemp("", "arc-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	store, err := artifacts.NewFileStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Setup Service
	svc := arc.NewIngestionService(store, &MockMeter{})

	// 3. Register Connectors
	eu := connectors.NewEUConnector()
	us := connectors.NewUSConnector()

	svc.RegisterConnector(eu)
	svc.RegisterConnector(us)

	// 4. Test EU Ingestion (Should Fail Closed)
	ctx := context.Background()
	_, _, err = svc.Ingest(ctx, "eu-eurlex", "32024R1689")
	if err == nil {
		t.Fatal("EU Ingest should have failed (Real Integration Missing)")
	}

	// 5. Test US Ingestion (Should Fail Closed)
	_, _, err = svc.Ingest(ctx, "us-federal-register", "2024-12345")
	if err == nil {
		t.Fatal("US Ingest should have failed (Real Integration Missing)")
	}

	t.Logf("Ingestion Verification Passed. Both connectors failed closed as expected.")
}
