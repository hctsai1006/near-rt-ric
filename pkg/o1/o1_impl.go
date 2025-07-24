package o1

import (
	"context"
	"fmt"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

type o1Interface struct {
	config        *config.O1Config
	logger        *logrus.Entry
	metrics       *monitoring.MetricsCollector
	netconfServer NetconfServerInterface
	fcapsManager  FCAPSManagerInterface
	running       bool
	startTime     time.Time
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewO1Interface(cfg *config.O1Config, baseLogger *logrus.Logger, metrics *monitoring.MetricsCollector) (O1Interface, error) {
	ctx, cancel := context.WithCancel(context.Background())

	netconfServer, err := NewNetconfServer(cfg, baseLogger, metrics)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create NETCONF server: %w", err)
	}

	fcapsManager := NewFCAPSManager(cfg, baseLogger, metrics)

	return &o1Interface{
		config:        cfg,
		logger:        baseLogger.WithField("component", "o1-interface"),
		metrics:       metrics,
		netconfServer: netconfServer,
		fcapsManager:  fcapsManager,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

func (o *o1Interface) Start() error {
	if o.running {
		return fmt.Errorf("O1 interface already running")
	}
	o.logger.Info("Starting O1 interface")
	if err := o.netconfServer.Start(o.ctx); err != nil {
		return fmt.Errorf("failed to start NETCONF server: %w", err)
	}
	if err := o.fcapsManager.Start(o.ctx); err != nil {
		return fmt.Errorf("failed to start FCAPS manager: %w", err)
	}
	o.running = true
	o.startTime = time.Now()
	return nil
}

func (o *o1Interface) Stop() error {
	o.logger.Info("Stopping O1 interface")
	o.cancel()
	if err := o.netconfServer.Stop(o.ctx); err != nil {
		return fmt.Errorf("failed to stop NETCONF server: %w", err)
	}
	if err := o.fcapsManager.Stop(o.ctx); err != nil {
		return fmt.Errorf("failed to stop FCAPS manager: %w", err)
	}
	o.running = false
	return nil
}

func (o *o1Interface) HealthCheck() O1HealthCheck {
	return O1HealthCheck{}
}

func (o *o1Interface) GetStatistics() O1Statistics {
	return O1Statistics{}
}

func (o *o1Interface) SendNotification(notification O1Event) error {
	return fmt.Errorf("not implemented")
}

func (o *o1Interface) UpdateConfig(change ConfigurationChange) error {
	return fmt.Errorf("not implemented")
}

func (o *o1Interface) GetConfig(filter Filter) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (o *o1Interface) GetAlarmList() ([]Alarm, error) {
	return nil, fmt.Errorf("not implemented")
}

func (o *o1Interface) GetPerformanceMetrics(filter PerformanceFilter) ([]PerformanceMetric, error) {
	return nil, fmt.Errorf("not implemented")
}
