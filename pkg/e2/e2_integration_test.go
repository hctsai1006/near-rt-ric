package e2

import (
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2SetupProcedure(t *testing.T) {
	// Setup E2 interface
	e2Interface := NewE2Interface(":36421")
	require.NoError(t, e2Interface.Start())
	defer e2Interface.Stop()

	// Create E2 Setup Request
	setupReq := &models.E2SetupRequest{
		GlobalE2NodeID: &models.GlobalE2NodeID{
			GNB_ID: &models.GNB_ID{
				GNB_ID: []byte{0x12, 0x34, 0x56},
			},
		},
		RANfunctions: []*models.RANfunction{
			{
				RANfunctionID:         1,
				RANfunctionDefinition: []byte("KPM_function_definition"),
				RANfunctionRevision:   1,
			},
		},
	}

	// Test ASN.1 encoding
	encoded, err := EncodeE2SetupRequest(setupReq)
	require.NoError(t, err)
	assert.Greater(t, len(encoded), 0)

	// Test SCTP transmission
	response, err := e2Interface.SendE2SetupRequest("test_node", setupReq)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, models.E2_SETUP_RESPONSE, response.ProcedureCode)
}

func TestRICSubscriptionProcedure(t *testing.T) {
	e2Interface := NewE2Interface(":36422") // Use a different port to avoid conflict
	require.NoError(t, e2Interface.Start())
	defer e2Interface.Stop()

	// Test subscription request
	subReq := &models.RICsubscriptionRequest{
		RICrequestID: &models.RICrequestID{
			RICrequestorID: 1,
			RICinstanceID:  1,
		},
		RANfunctionID: 1,
		RICsubscriptionDetails: &models.RICsubscriptionDetails{
			RICeventTriggerDefinition: []byte("trigger_definition"),
			RICactions: []*models.RICaction{
				{
					RICactionID:   1,
					RICactionType: models.Report,
				},
			},
		},
	}

	response, err := e2Interface.CreateSubscription("test_node", subReq)
	require.NoError(t, err)
	assert.Equal(t, models.RIC_SUBSCRIPTION_RESPONSE, response.ProcedureCode)
}

func TestE2PerformanceRequirements(t *testing.T) {
	e2Interface := NewE2Interface(":36423") // Use a different port
	require.NoError(t, e2Interface.Start())
	defer e2Interface.Stop()

	// Test latency requirement: < 10ms for E2 message processing
	start := time.Now()

	setupReq := &models.E2SetupRequest{
		GlobalE2NodeID: &models.GlobalE2NodeID{
			GNB_ID: &models.GNB_ID{
				GNB_ID: []byte{0xAB, 0xCD, 0xEF},
			},
		},
	}

	_, err := e2Interface.SendE2SetupRequest("perf_test_node", setupReq)
	require.NoError(t, err)

	latency := time.Since(start)
	assert.Less(t, latency, 10*time.Millisecond, "E2 message processing should be < 10ms")
}
