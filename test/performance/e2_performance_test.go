package performance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/sirupsen/logrus"
)

func BenchmarkE2Performance(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cfg := &config.E2Config{
		ListenAddress: "127.0.0.1",
		ListenPort:    36421,
		SCTP: config.SCTPConfig{
			ConnectionTimeout: 30,
		},
	}

	e2Interface := e2.NewE2Interface(fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort))

	go func() {
		if err := e2Interface.Start(context.Background()); err != nil {
			b.Fatalf("E2 interface start error: %v", err)
		}
	}()
	defer e2Interface.Stop()

	time.Sleep(2 * time.Second)

	// This is a placeholder for a performance test.
	// In a real implementation, you would create a client and send a high volume of messages.
	b.Run("E2SetupRequest", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Placeholder for sending an E2 setup request
		}
	})
}
