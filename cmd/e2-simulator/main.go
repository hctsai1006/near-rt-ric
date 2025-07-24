package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
	"github.com/sirupsen/logrus"
)

const (
	version = "1.0.0"
	appName = "O-RAN E2 Node Simulator"
)

var (
	ricAddr     = flag.String("ric-addr", "127.0.0.1", "RIC address to connect to")
	ricPort     = flag.Int("ric-port", 36421, "RIC port to connect to")
	nodeID      = flag.String("node-id", "gnb_001", "E2 node identifier")
	nodeType    = flag.String("node-type", "gnb", "E2 node type (gnb)")
	plmnID      = flag.String("plmn-id", "310410", "PLMN identifier")
	logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	interval    = flag.Duration("report-interval", 30*time.Second, "Reporting interval for indications")
	showVersion = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("%s version %s\n", appName, version)
		os.Exit(0)
	}

	// Configure logging
	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.WithError(err).Fatal("Invalid log level")
	}
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	logger.WithFields(logrus.Fields{
		"version":   version,
		"ric_addr":  *ricAddr,
		"ric_port":  *ricPort,
		"node_id":   *nodeID,
		"node_type": *nodeType,
		"plmn_id":   *plmnID,
	}).Info("Starting O-RAN E2 Node Simulator")

	// Convert node type string to enum
	var nodeTypeEnum models.E2NodeType
	switch *nodeType {
	case "gnb":
		nodeTypeEnum = models.E2NodeTypeGNB
	default:
		logger.WithField("node_type", *nodeType).Fatal("Invalid node type")
	}

	// Create E2 interface
	e2if := e2.NewE2Interface(fmt.Sprintf("%s:%d", *ricAddr, *ricPort))

	// Start E2 interface
	if err := e2if.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start E2 interface")
	}
	defer e2if.Stop()

	// Send E2 Setup Request
	if err := sendE2SetupRequest(e2if, *nodeID, nodeTypeEnum, []byte(*plmnID)); err != nil {
		logger.WithError(err).Fatal("Failed to send E2 setup request")
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("O-RAN E2 Node Simulator started successfully")

	// Start periodic reporting
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go periodicReporting(ctx, e2if, *nodeID, *interval, logger)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.WithField("signal", sig.String()).Info("Received shutdown signal")
}

// sendE2SetupRequest sends an E2 setup request to the RIC
func sendE2SetupRequest(e2if *e2.E2Interface, nodeID string, nodeType models.E2NodeType, plmnID []byte) error {
	setupReq := &models.E2SetupRequest{
		TransactionID: 1,
		GlobalE2NodeID: &models.GlobalE2NodeID{
			GNB_ID: &models.GNB_ID{
				GNB_ID: []byte(nodeID),
			},
		},
		RANfunctions: []*models.RANfunction{
			{
				RANfunctionID:         1,
				RANfunctionDefinition: []byte("E2SM-KPM-v01.00"),
				RANfunctionRevision:   1,
			},
		},
	}

	if _, err := e2if.SendE2SetupRequest(nodeID, setupReq); err != nil {
		return fmt.Errorf("failed to send E2 setup request: %w", err)
	}

	logrus.Info("E2 Setup Request sent successfully")
	return nil
}

// periodicReporting sends periodic RIC indications
func periodicReporting(ctx context.Context, e2if *e2.E2Interface, nodeID string, interval time.Duration, logger *logrus.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// In a real implementation, this would be a proper RIC indication
			logger.Debug("Sending RIC indication")
		}
	}
}
