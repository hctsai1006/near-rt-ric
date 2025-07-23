package o1

import (
	"github.com/hctsai1006/near-rt-ric/pkg/o1/accounting"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/config"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/fault"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/netconf"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/performance"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/security"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/yang"
)

// O1Server represents the O1 interface server
type O1Server struct {
	NetconfServer *netconf.Server
	YangManager   *yang.Manager
	FaultMgr      fault.FaultManager
	ConfigMgr     config.ConfigurationManager
	PerfMgr       performance.PerformanceManager
	SecurityMgr   security.SecurityManager
	AccountMgr    accounting.AccountingManager
}

// NewO1Server creates a new O1 server
func NewO1Server(yangManager *yang.Manager, logger *logrus.Logger) *O1Server {
	return &O1Server{
		YangManager: yangManager,
		ConfigMgr:   config.NewConfigurationManager(yangManager, logger),
	}
}
