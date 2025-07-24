package integration

import (
	"context"
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/ric"
	"github.com/stretchr/testify/require"
)

func TestRICMain(t *testing.T) {
	cfg, err := config.LoadConfig("../../config/config.yaml")
	require.NoError(t, err)

	server, err := ric.NewRICServer(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		err := server.Start()
		require.NoError(t, err)
	}()

	// Let the server run for a bit
	time.Sleep(5 * time.Second)

	err = server.Stop()
	require.NoError(t, err)
}