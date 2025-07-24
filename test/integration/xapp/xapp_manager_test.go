package xapp_test

import (
	"context"
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/xapp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type XAppManagerTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	xappManager xapp.XAppManager
}

func (suite *XAppManagerTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cfg := &xapp.XAppFrameworkConfig{
		ConflictDetection:   true,
		HealthCheckInterval: 30 * time.Second,
		MetricsInterval:     60 * time.Second,
		Namespace:           "default",
	}

	repo := xapp.NewMemoryRepository()
	orch := xapp.NewDummyOrchestrator(logger.WithField("component", "dummy-orchestrator"))
	reg := xapp.NewDummyRegistry(logger.WithField("component", "dummy-registry"))

	suite.xappManager = xapp.NewXAppManager(repo, orch, reg, cfg, logger)
}

func (suite *XAppManagerTestSuite) TearDownSuite() {
	suite.xappManager.Cleanup()
	suite.cancel()
}

func (suite *XAppManagerTestSuite) TestDeployUndeploy() {
	suite.T().Log("Testing xApp deployment")
	descriptor := &xapp.XAppDescriptor{
		Name:    "test-xapp",
		Version: "1.0.0",
		Image:   "test-image",
	}

	instance, err := suite.xappManager.Deploy(descriptor, nil)
	suite.NoError(err)
	suite.NotNil(instance)
	suite.T().Logf("Deployed xApp instance %s", instance.InstanceID)

	suite.T().Log("Testing xApp undeployment")
	err = suite.xappManager.Undeploy(string(instance.InstanceID))
	suite.NoError(err)
	suite.T().Log("Undeployed xApp instance")
}

func TestXAppManagerSuite(t *testing.T) {
	suite.Run(t, new(XAppManagerTestSuite))
}