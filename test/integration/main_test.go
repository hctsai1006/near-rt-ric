package integration

import (
	"os"
	"testing"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	// Set up logger
	logrus.SetLevel(logrus.DebugLevel)

	// Load configuration
	_, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Run tests
	exitCode := m.Run()

	// Exit
	os.Exit(exitCode)
}