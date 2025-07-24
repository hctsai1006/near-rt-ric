package e2

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
	"github.com/sirupsen/logrus"
)

// WorkerPool manages concurrent message processing for E2 interface
type WorkerPool struct {
	config  *config.E2Config
	logger  *logrus.Logger
	metrics *monitoring.MetricsCollector

	// Worker management
	workerCount   int
	workers       []*Worker
	messageQueue  chan *models.E2Message
	resultChannel chan *models.MessageResult

	// Message handlers
	messageHandler MessageHandler

	// Control and synchronization
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  atomic.Bool

	// Statistics
	stats *models.WorkerPoolStats
}

// Worker represents a single worker in the pool
type Worker struct {
	id           int
	pool         *WorkerPool
	messageQueue chan *models.E2Message
	logger       *logrus.Entry
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// MessageHandler defines the interface for handling E2 messages
type MessageHandler interface {
	HandleE2SetupRequest(connectionID, nodeID string, msg *models.E2SetupRequest) (*models.E2SetupResponse, error)
	HandleE2SetupResponse(connectionID, nodeID string, msg *models.E2SetupResponse) error
	HandleE2SetupFailure(connectionID, nodeID string, msg *models.E2SetupFailure) error
	HandleRICSubscriptionRequest(connectionID, nodeID string, msg *models.RICSubscriptionRequest) (*models.RICSubscriptionResponse, error)
	HandleRICSubscriptionResponse(connectionID, nodeID string, msg *models.RICSubscriptionResponse) error
	HandleRICSubscriptionFailure(connectionID, nodeID string, msg *models.RICSubscriptionFailure) error
	HandleRICSubscriptionDeleteRequest(connectionID, nodeID string, msg *models.RICSubscriptionDeleteRequest) (*models.RICSubscriptionDeleteResponse, error)
	HandleRICSubscriptionDeleteResponse(connectionID, nodeID string, msg *models.RICSubscriptionDeleteResponse) error
	HandleRICIndication(connectionID, nodeID string, msg *models.RICIndication) error
	HandleRICControlRequest(connectionID, nodeID string, msg *models.RICControlRequest) (*models.RICControlAck, error)
	HandleRICControlAck(connectionID, nodeID string, msg *models.RICControlAck) error
	HandleRICControlFailure(connectionID, nodeID string, msg *models.RICControlFailure) error
}

// NewWorkerPool creates a new worker pool for E2 message processing
func NewWorkerPool(config *config.E2Config, logger *logrus.Logger, metrics *monitoring.MetricsCollector) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	// Determine worker count
	workerCount := config.WorkerPool.Size
	if workerCount <= 0 {
		workerCount = runtime.NumCPU() * 2 // Default: 2 workers per CPU core
	}

	// Determine queue size
	queueSize := config.WorkerPool.QueueSize
	if queueSize <= 0 {
		queueSize = 1000 // Default queue size
	}

	pool := &WorkerPool{
		config:        config,
				logger:        logger.WithField("component", "worker-pool"),
		metrics:       metrics,
		workerCount:   workerCount,
		workers:       make([]*Worker, workerCount),
		messageQueue:  make(chan *models.E2Message, queueSize),
		resultChannel: make(chan *models.MessageResult, queueSize),
		ctx:           ctx,
		cancel:        cancel,
		stats:         &models.WorkerPoolStats{},
	}

	return pool
}

// SubmitMessage submits a message for processing
func (wp *WorkerPool) SubmitMessage(msg *models.E2Message) error {
	if !wp.running.Load() {
		return fmt.Errorf("worker pool is not running")
	}

	wp.stats.TotalMessages.Add(1)
	wp.stats.QueueSize.Store(int32(len(wp.messageQueue)))

	select {
	case wp.messageQueue <- msg:
		wp.logger.WithFields(logrus.Fields{
			"message_id":   msg.MessageID,
			"message_type": msg.MessageType.String(),
			"node_id":      msg.NodeID,
			"queue_size":   len(wp.messageQueue),
		}).Debug("Message submitted to worker pool")
		return nil
	case <-time.After(5 * time.Second):
		wp.stats.FailedMessages.Add(1)
		return fmt.Errorf("message queue is full, could not submit message")
	}
}

// processMessage processes a single E2 message
func (w *Worker) processMessage(msg *models.E2Message) {
	start := time.Now()
	result := &models.MessageResult{
		MessageID: msg.MessageID,
		WorkerID:  w.id,
	}

	w.logger.WithFields(logrus.Fields{
		"message_id":   msg.MessageID,
		"message_type": msg.MessageType.String(),
		"node_id":      msg.NodeID,
	}).Debug("Processing E2 message")

	// Decode message based on type
	var response interface{}
	var err error

	switch msg.MessageType {
	case models.E2SetupRequestMsg:
		response, err = w.handleE2SetupRequest(msg)
	case models.E2SetupResponseMsg:
		err = w.handleE2SetupResponse(msg)
	case models.E2SetupFailureMsg:
		err = w.handleE2SetupFailure(msg)
	case models.RICSubscriptionRequestMsg:
		response, err = w.handleRICSubscriptionRequest(msg)
	case models.RICSubscriptionResponseMsg:
		err = w.handleRICSubscriptionResponse(msg)
	case models.RICSubscriptionFailureMsg:
		err = w.handleRICSubscriptionFailure(msg)
	case models.RICSubscriptionDeleteRequestMsg:
		response, err = w.handleRICSubscriptionDeleteRequest(msg)
	case models.RICSubscriptionDeleteResponseMsg:
		err = w.handleRICSubscriptionDeleteResponse(msg)
	case models.RICIndicationMsg:
		err = w.handleRICIndication(msg)
	case models.RICControlRequestMsg:
		response, err = w.handleRICControlRequest(msg)
	case models.RICControlAckMsg:
		err = w.handleRICControlAck(msg)
	case models.RICControlFailureMsg:
		err = w.handleRICControlFailure(msg)
	default:
		err = fmt.Errorf("unsupported message type: %s", msg.MessageType.String())
	}

	// Record processing time
	processingTime := time.Since(start)
	result.ProcessingTime = processingTime
	result.Response = response
	result.Error = err
	result.Success = (err == nil)

	// Update statistics
	if err == nil {
		w.pool.stats.ProcessedMessages.Add(1)
	} else {
		w.pool.stats.FailedMessages.Add(1)
		w.logger.WithFields(logrus.Fields{
			"message_id":   msg.MessageID,
			"message_type": msg.MessageType.String(),
			"error":        err,
			"duration":     processingTime,
		}).Error("Failed to process E2 message")
	}

	// Update average latency
	currentAvg := time.Duration(w.pool.stats.AverageLatency.Load())
	newAvg := (currentAvg + processingTime) / 2
	w.pool.stats.AverageLatency.Store(uint64(newAvg))

	// Send result
	select {
	case w.pool.resultChannel <- result:
	default:
		w.logger.Warn("Result channel full, dropping result")
	}

	w.logger.WithFields(logrus.Fields{
		"message_id":   msg.MessageID,
		"message_type": msg.MessageType.String(),
		"success":      result.Success,
		"duration":     processingTime,
	}).Debug("E2 message processing completed")
}

// handleE2SetupRequest handles E2 Setup Request messages
func (w *Worker) handleE2SetupRequest(msg *models.E2Message) (interface{}, error) {
	// Decode ASN.1 message
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode E2 setup request: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return nil, fmt.Errorf("expected initiating message for E2 setup request")
	}

	// Extract E2SetupRequest from PDU value
	setupReq, ok := pdu.InitiatingMessage.Value.(*models.E2SetupRequest)
	if !ok {
		return nil, fmt.Errorf("failed to extract E2SetupRequest from PDU")
	}

	return w.pool.messageHandler.HandleE2SetupRequest(msg.ConnectionID, msg.NodeID, setupReq)
}

// handleE2SetupResponse handles E2 Setup Response messages
func (w *Worker) handleE2SetupResponse(msg *models.E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode E2 setup response: %w", err)
	}

	if pdu.SuccessfulOutcome == nil {
		return fmt.Errorf("expected successful outcome for E2 setup response")
	}

	setupResp, ok := pdu.SuccessfulOutcome.Value.(*models.E2SetupResponse)
	if !ok {
		return fmt.Errorf("failed to extract E2SetupResponse from PDU")
	}

	return w.pool.messageHandler.HandleE2SetupResponse(msg.ConnectionID, msg.NodeID, setupResp)
}

// handleE2SetupFailure handles E2 Setup Failure messages
func (w *Worker) handleE2SetupFailure(msg *models.E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode E2 setup failure: %w", err)
	}

	if pdu.UnsuccessfulOutcome == nil {
		return fmt.Errorf("expected unsuccessful outcome for E2 setup failure")
	}

	setupFailure, ok := pdu.UnsuccessfulOutcome.Value.(*models.E2SetupFailure)
	if !ok {
		return fmt.Errorf("failed to extract E2SetupFailure from PDU")
	}

	return w.pool.messageHandler.HandleE2SetupFailure(msg.ConnectionID, msg.NodeID, setupFailure)
}

// handleRICSubscriptionRequest handles RIC Subscription Request messages
func (w *Worker) handleRICSubscriptionRequest(msg *models.E2Message) (interface{}, error) {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode RIC subscription request: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return nil, fmt.Errorf("expected initiating message for RIC subscription request")
	}

	subReq, ok := pdu.InitiatingMessage.Value.(*models.RICSubscriptionRequest)
	if !ok {
		return nil, fmt.Errorf("failed to extract RICSubscriptionRequest from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionRequest(msg.ConnectionID, msg.NodeID, subReq)
}

// handleRICSubscriptionResponse handles RIC Subscription Response messages
func (w *Worker) handleRICSubscriptionResponse(msg *models.E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC subscription response: %w", err)
	}

	if pdu.SuccessfulOutcome == nil {
		return fmt.Errorf("expected successful outcome for RIC subscription response")
	}

	subResp, ok := pdu.SuccessfulOutcome.Value.(*models.RICSubscriptionResponse)
	if !ok {
		return fmt.Errorf("failed to extract RICSubscriptionResponse from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionResponse(msg.ConnectionID, msg.NodeID, subResp)
}

// handleRICSubscriptionFailure handles RIC Subscription Failure messages
func (w *Worker) handleRICSubscriptionFailure(msg *models.E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC subscription failure: %w", err)
	}

	if pdu.UnsuccessfulOutcome == nil {
		return fmt.Errorf("expected unsuccessful outcome for RIC subscription failure")
	}

	subFailure, ok := pdu.UnsuccessfulOutcome.Value.(*models.RICSubscriptionFailure)
	if !ok {
		return fmt.Errorf("failed to extract RICSubscriptionFailure from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionFailure(msg.ConnectionID, msg.NodeID, subFailure)
}

// handleRICSubscriptionDeleteRequest handles RIC Subscription Delete Request messages
func (w *Worker) handleRICSubscriptionDeleteRequest(msg *models.E2Message) (interface{}, error) {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode RIC subscription delete request: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return nil, fmt.Errorf("expected initiating message for RIC subscription delete request")
	}

	delReq, ok := pdu.InitiatingMessage.Value.(*models.RICSubscriptionDeleteRequest)
	if !ok {
		return nil, fmt.Errorf("failed to extract RICSubscriptionDeleteRequest from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionDeleteRequest(msg.ConnectionID, msg.NodeID, delReq)
}

// handleRICSubscriptionDeleteResponse handles RIC Subscription Delete Response messages
func (w *Worker) handleRICSubscriptionDeleteResponse(msg *models.E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC subscription delete response: %w", err)
	}

	if pdu.SuccessfulOutcome == nil {
		return fmt.Errorf("expected successful outcome for RIC subscription delete response")
	}

	delResp, ok := pdu.SuccessfulOutcome.Value.(*models.RICSubscriptionDeleteResponse)
	if !ok {
		return fmt.Errorf("failed to extract RICSubscriptionDeleteResponse from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionDeleteResponse(msg.ConnectionID, msg.NodeID, delResp)
}

// handleRICIndication handles RIC Indication messages
func (w *Worker) handleRICIndication(msg *models.E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC indication: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return fmt.Errorf("expected initiating message for RIC indication")
	}

	indication, ok := pdu.InitiatingMessage.Value.(*models.RICIndication)
	if !ok {
		return fmt.Errorf("failed to extract RICIndication from PDU")
	}

	return w.pool.messageHandler.HandleRICIndication(msg.ConnectionID, msg.NodeID, indication)
}

// handleRICControlRequest handles RIC Control Request messages
func (w *Worker) handleRICControlRequest(msg *models.E2Message) (interface{}, error) {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode RIC control request: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return nil, fmt.Errorf("expected initiating message for RIC control request")
	}

	controlReq, ok := pdu.InitiatingMessage.Value.(*models.RICControlRequest)
	if !ok {
		return nil, fmt.Errorf("failed to extract RICControlRequest from PDU")
	}

	return w.pool.messageHandler.HandleRICControlRequest(msg.ConnectionID, msg.NodeID, controlReq)
}

// handleRICControlAck handles RIC Control Acknowledge messages
func (w *Worker) handleRICControlAck(msg *models.E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC control ack: %w", err)
	}

	if pdu.SuccessfulOutcome == nil {
		return fmt.Errorf("expected successful outcome for RIC control ack")
	}

	controlAck, ok := pdu.SuccessfulOutcome.Value.(*models.RICControlAck)
	if !ok {
		return fmt.Errorf("failed to extract RICControlAck from PDU")
	}

	return w.pool.messageHandler.HandleRICControlAck(msg.ConnectionID, msg.NodeID, controlAck)
}

// handleRICControlFailure handles RIC Control Failure messages
func (w *Worker) handleRICControlFailure(msg *models.E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC control failure: %w", err)
	}

	if pdu.UnsuccessfulOutcome == nil {
		return fmt.Errorf("expected unsuccessful outcome for RIC control failure")
	}

	controlFailure, ok := pdu.UnsuccessfulOutcome.Value.(*models.RICControlFailure)
	if !ok {
		return fmt.Errorf("failed to extract RICControlFailure from PDU")
	}

	return w.pool.messageHandler.HandleRICControlFailure(msg.ConnectionID, msg.NodeID, controlFailure)
}

// processResults processes message results from workers
func (wp *WorkerPool) processResults() {
	defer wp.wg.Done()

	wp.logger.Debug("Starting result processor")

	for {
		select {
		case <-wp.ctx.Done():
			wp.logger.Debug("Result processor stopping")
			return

		case result, ok := <-wp.resultChannel:
			if !ok {
				wp.logger.Debug("Result channel closed, processor stopping")
				return
			}

			wp.processResult(result)
		}
	}
}

// processResult processes a single message result
func (wp *WorkerPool) processResult(result *models.MessageResult) {
	// Update metrics
	wp.metrics.E2Metrics.MessageLatencySeconds.WithLabelValues("unknown", "process").Observe(result.ProcessingTime.Seconds())

	if result.Success {
		wp.logger.WithFields(logrus.Fields{
			"message_id": result.MessageID,
			"worker_id":  result.WorkerID,
			"duration":   result.ProcessingTime,
		}).Debug("Message processing result: success")
	} else {
		wp.logger.WithFields(logrus.Fields{
			"message_id": result.MessageID,
			"worker_id":  result.WorkerID,
			"duration":   result.ProcessingTime,
			"error":      result.Error,
		}).Warn("Message processing result: failure")
	}

	// TODO: Handle response messages if needed
	// For example, send response back to the SCTP connection
}

// statisticsCollector periodically collects worker pool statistics
func (wp *WorkerPool) statisticsCollector() {
	defer wp.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case <-ticker.C:
			wp.collectStatistics()
		}
	}
}

// collectStatistics collects and logs worker pool statistics
func (wp *WorkerPool) collectStatistics() {
	stats := wp.stats
	avgLatency := time.Duration(stats.AverageLatency.Load())

	wp.logger.WithFields(logrus.Fields{
		"total_messages":     stats.TotalMessages.Load(),
		"processed_messages": stats.ProcessedMessages.Load(),
		"failed_messages":    stats.FailedMessages.Load(),
		"queue_size":         stats.QueueSize.Load(),
		"active_workers":     stats.ActiveWorkers.Load(),
		"average_latency":    avgLatency,
	}).Info("Worker pool statistics")
}