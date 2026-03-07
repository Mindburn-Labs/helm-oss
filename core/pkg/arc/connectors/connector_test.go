package connectors_test

import (
	"context"
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/arc/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Invariant: production connectors must never return stubbed regulatory data.
// If the real integration isn't wired, they MUST fail closed.
func TestUSConnector_FailClosedWhenIntegrationMissing(t *testing.T) {
	c := connectors.NewUSConnector()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	content, mime, meta, err := c.Fetch(ctx, "DOC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "integration missing")
	assert.Nil(t, content)
	assert.Equal(t, "", mime)
	assert.Nil(t, meta)
}

func TestEUConnector_FailClosedWhenIntegrationMissing(t *testing.T) {
	c := connectors.NewEUConnector()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	content, mime, meta, err := c.Fetch(ctx, "ELI-456")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "integration missing")
	assert.Nil(t, content)
	assert.Equal(t, "", mime)
	assert.Nil(t, meta)
}
