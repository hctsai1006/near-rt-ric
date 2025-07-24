//go:build linux
// +build linux

package e2_test

import (
	"net"
	"testing"

	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
	"github.com/ishidawataru/sctp"
	"github.com/onosproject/onos-lib-go/pkg/asn1/aper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2FullFlow(t *testing.T) {
	// SCTP server setup
	addr, err := sctp.ResolveSCTPAddr("sctp", "127.0.0.1:0")
	require.NoError(t, err)
	ln, err := sctp.ListenSCTP("sctp", addr)
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			buf := make([]byte, 1500)
			n, err := conn.Read(buf)
			if err != nil {
				return
			}

			var pdu models.E2AP_PDU
			err = aper.Unmarshal(buf[:n], &pdu, nil, models.E2AP_PDU_TypeMaps)
			if err != nil {
				return
			}

			if pdu.InitiatingMessage != nil {
				switch pdu.InitiatingMessage.ProcedureCode {
				case models.ProcedureCodeE2Setup:
					resp := &models.E2SetupResponse{
						TransactionID: 1,
					}
					encodedResp, err := e2.EncodeE2SetupResponse(resp)
					if err != nil {
						return
					}
					conn.Write(encodedResp)
				case models.ProcedureCodeRICSubscription:
					resp := &models.RICSubscriptionResponse{
						RICRequestID: models.RICrequestID{
							RICrequestorID: 1,
							RICinstanceID:  1,
						},
					}
					encodedResp, err := e2.EncodeRICSubscriptionResponse(resp)
					if err != nil {
						return
					}
					conn.Write(encodedResp)
				}
			}
		}
	}()

	// Client
	clientAddr := ln.Addr()
	clientConn, err := sctp.DialSCTP("sctp", nil, clientAddr.(*sctp.SCTPAddr))
	require.NoError(t, err)
	defer clientConn.Close()

	t.Run("TestE2Setup", func(t *testing.T) {
		// E2 Setup Procedure
		setupReq := &models.E2SetupRequest{
			TransactionID: 1,
		}
		encodedReq, err := e2.EncodeE2SetupRequest(setupReq)
		require.NoError(t, err)

		_, err = clientConn.Write(encodedReq)
		require.NoError(t, err)

		// Read response
		buf := make([]byte, 1500)
		n, err := clientConn.Read(buf)
		require.NoError(t, err)
		assert.True(t, n > 0)

		// Decode and validate response
		setupResp, err := e2.DecodeE2SetupResponse(buf[:n])
		require.NoError(t, err)
		assert.NotNil(t, setupResp)
		assert.Equal(t, int64(1), setupResp.TransactionID)
	})

	t.Run("TestSubscription", func(t *testing.T) {
		// Subscription Procedure
		subscriptionReq := &models.RICSubscriptionRequest{
			RICrequestID: &models.RICrequestID{RICrequestorID: 1, RICinstanceID: 1},
		}
		encodedReq, err := e2.EncodeSubscriptionRequest(subscriptionReq)
		require.NoError(t, err)

		_, err = clientConn.Write(encodedReq)
		require.NoError(t, err)

		// Read response
		buf := make([]byte, 1500)
		n, err := clientConn.Read(buf)
		require.NoError(t, err)
		assert.True(t, n > 0)

		// Decode and validate response
		subResp, err := e2.DecodeRICSubscriptionResponse(buf[:n])
		require.NoError(t, err)
		assert.NotNil(t, subResp)
		assert.Equal(t, 1, subResp.RICRequestID.RICrequestorID)
		assert.Equal(t, 1, subResp.RICRequestID.RICinstanceID)
	})
}
