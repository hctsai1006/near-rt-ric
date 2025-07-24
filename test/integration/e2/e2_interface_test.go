package e2_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type E2InterfaceTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	e2Interface *e2.E2Interface
}

func (suite *E2InterfaceTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cfg := &config.E2Config{
		ListenAddress: "127.0.0.1",
		ListenPort:    36421,
	}

	suite.e2Interface = e2.NewE2Interface(fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort))

	go func() {
		if err := suite.e2Interface.Start(suite.ctx); err != nil {
			suite.T().Logf("E2 interface start error: %v", err)
		}
	}()
}

func (suite *E2InterfaceTestSuite) TearDownSuite() {
	suite.e2Interface.Stop()
	suite.cancel()
}

func (suite *E2InterfaceTestSuite) TestE2Setup() {
	// This is a placeholder for a test that would send an E2 setup request
	// and verify the response.
	req := &models.E2SetupRequest{}
	_, err := suite.e2Interface.SendE2SetupRequest("test-node", req)
	suite.NoError(err)
}

func TestE2InterfaceSuite(t *testing.T) {
	suite.Run(t, new(E2InterfaceTestSuite))
}