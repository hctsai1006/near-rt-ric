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
	messageQueue  chan *E2Message
	resultChannel chan *MessageResult

	// Message handlers
	messageHandler MessageHandler

	// Control and synchronization
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  atomic.Bool

	// Statistics
	stats *WorkerPoolStats
}

// Worker represents a single worker in the pool
type Worker struct {
	id           int
	pool         *WorkerPool
	messageQueue chan *E2Message
	logger       *logrus.Entry
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// MessageHandler defines the interface for handling E2 messages
type MessageHandler interface {
	HandleE2SetupRequest(connectionID, nodeID string, msg *E2SetupRequest) (*E2SetupResponse, error)
	HandleE2SetupResponse(connectionID, nodeID string, msg *E2SetupResponse) error
	HandleE2SetupFailure(connectionID, nodeID string, msg *E2SetupFailure) error
	HandleRICSubscriptionRequest(connectionID, nodeID string, msg *RICSubscriptionRequest) (*RICSubscriptionResponse, error)
	HandleRICSubscriptionResponse(connectionID, nodeID string, msg *RICSubscriptionResponse) error
	HandleRICSubscriptionFailure(connectionID, nodeID string, msg *RICSubscriptionFailure) error
	HandleRICSubscriptionDeleteRequest(connectionID, nodeID string, msg *RICSubscriptionDeleteRequest) (*RICSubscriptionDeleteResponse, error)
	HandleRICSubscriptionDeleteResponse(connectionID, nodeID string, msg *RICSubscriptionDeleteResponse) error
	HandleRICIndication(connectionID, nodeID string, msg *RICIndication) error
	HandleRICControlRequest(connectionID, nodeID string, msg *RICControlRequest) (*RICControlAck, error)
	HandleRICControlAck(connectionID, nodeID string, msg *RICControlAck) error
	HandleRICControlFailure(connectionID, nodeID string, msg *RICControlFailure) error
}

// MessageResult represents the result of message processing
type MessageResult struct {
	MessageID    string
	Success      bool
	Error        error
	Response     interface{}
	ProcessingTime time.Duration
	WorkerID     int
}

// WorkerPoolStats contains statistics for the worker pool
type WorkerPoolStats struct {
	TotalMessages     atomic.Uint64
	ProcessedMessages atomic.Uint64
	FailedMessages    atomic.Uint64
	QueueSize         atomic.Int32
	ActiveWorkers     atomic.Int32
	AverageLatency    atomic.Uint64 // in nanoseconds
}

// WorkerPoolConfig contains configuration for the worker pool
type WorkerPoolConfig struct {
	WorkerCount       int
	QueueSize         int
	MessageTimeout    time.Duration
	ShutdownTimeout   time.Duration
	StatsInterval     time.Duration
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
		logger:        logger.WithField("component", "worker-pool").Logger,
		metrics:       metrics,
		workerCount:   workerCount,
		workers:       make([]*Worker, workerCount),
		messageQueue:  make(chan *E2Message, queueSize),
		resultChannel: make(chan *MessageResult, queueSize),
		ctx:           ctx,
		cancel:        cancel,
		stats:         &WorkerPoolStats{},
	}

	return pool
}

// SetMessageHandler sets the message handler for the worker pool
func (wp *WorkerPool) SetMessageHandler(handler MessageHandler) {
	wp.messageHandler = handler
}

// Start starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context) error {
	if !wp.running.CompareAndSwap(false, true) {
		return fmt.Errorf("worker pool is already running")
	}

	if wp.messageHandler == nil {
		wp.running.Store(false)
		return fmt.Errorf("message handler must be set before starting worker pool")
	}

	wp.logger.WithFields(logrus.Fields{
		"worker_count": wp.workerCount,
		"queue_size":   cap(wp.messageQueue),
	}).Info("Starting E2 worker pool")

	// Start workers
	for i := 0; i < wp.workerCount; i++ {
		worker := &Worker{
			id:           i,
			pool:         wp,
			messageQueue: wp.messageQueue,
			logger:       wp.logger.WithField("worker_id", i),
			ctx:          wp.ctx,
		}
		worker.ctx, worker.cancel = context.WithCancel(wp.ctx)
		wp.workers[i] = worker

		wp.wg.Add(1)
		go worker.run()
	}

	// Start result processor
	wp.wg.Add(1)
	go wp.processResults()

	// Start statistics collector
	wp.wg.Add(1)
	go wp.statisticsCollector()

	wp.logger.WithField("worker_count", wp.workerCount).Info("E2 worker pool started successfully")
	return nil
}

// Stop stops the worker pool gracefully
func (wp *WorkerPool) Stop(ctx context.Context) error {
	if !wp.running.CompareAndSwap(true, false) {
		return nil
	}

	wp.logger.Info("Stopping E2 worker pool")

	// Close message queue to signal workers to stop accepting new messages
	close(wp.messageQueue)

	// Cancel context to stop all workers
	wp.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.logger.Info("E2 worker pool stopped successfully")
	case <-ctx.Done():
		wp.logger.Warn("E2 worker pool shutdown timeout")
	}

	// Close result channel
	close(wp.resultChannel)

	return nil
}

// SubmitMessage submits a message for processing
func (wp *WorkerPool) SubmitMessage(msg *E2Message) error {
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

// GetStats returns current worker pool statistics
func (wp *WorkerPool) GetStats() *WorkerPoolStats {
	return wp.stats
}

// GetQueueSize returns the current size of the message queue
func (wp *WorkerPool) GetQueueSize() int {
	return len(wp.messageQueue)
}

// run executes the worker main loop
func (w *Worker) run() {
	defer w.pool.wg.Done()
	defer w.cancel()

	w.logger.Debug("Worker started")
	w.pool.stats.ActiveWorkers.Add(1)

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Debug("Worker stopping")
			w.pool.stats.ActiveWorkers.Add(-1)
			return

		case msg, ok := <-w.messageQueue:
			if !ok {
				w.logger.Debug("Message queue closed, worker stopping")
				w.pool.stats.ActiveWorkers.Add(-1)
				return
			}

			// Process the message
			w.processMessage(msg)
		}
	}
}

// processMessage processes a single E2 message
func (w *Worker) processMessage(msg *E2Message) {
	start := time.Now()
	result := &MessageResult{
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
	case E2SetupRequestMsg:
		response, err = w.handleE2SetupRequest(msg)
	case E2SetupResponseMsg:
		err = w.handleE2SetupResponse(msg)
	case E2SetupFailureMsg:
		err = w.handleE2SetupFailure(msg)
	case RICSubscriptionRequestMsg:
		response, err = w.handleRICSubscriptionRequest(msg)
	case RICSubscriptionResponseMsg:
		err = w.handleRICSubscriptionResponse(msg)
	case RICSubscriptionFailureMsg:
		err = w.handleRICSubscriptionFailure(msg)
	case RICSubscriptionDeleteRequestMsg:
		response, err = w.handleRICSubscriptionDeleteRequest(msg)
	case RICSubscriptionDeleteResponseMsg:
		err = w.handleRICSubscriptionDeleteResponse(msg)
	case RICIndicationMsg:
		err = w.handleRICIndication(msg)
	case RICControlRequestMsg:
		response, err = w.handleRICControlRequest(msg)
	case RICControlAckMsg:
		err = w.handleRICControlAck(msg)
	case RICControlFailureMsg:
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
func (w *Worker) handleE2SetupRequest(msg *E2Message) (interface{}, error) {
	// Decode ASN.1 message
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode E2 setup request: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return nil, fmt.Errorf("expected initiating message for E2 setup request")
	}

	// Extract E2SetupRequest from PDU value
	setupReq, ok := pdu.InitiatingMessage.Value.(*E2SetupRequest)
	if !ok {
		return nil, fmt.Errorf("failed to extract E2SetupRequest from PDU")
	}

	return w.pool.messageHandler.HandleE2SetupRequest(msg.ConnectionID, msg.NodeID, setupReq)
}

// handleE2SetupResponse handles E2 Setup Response messages
func (w *Worker) handleE2SetupResponse(msg *E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode E2 setup response: %w", err)
	}

	if pdu.SuccessfulOutcome == nil {
		return fmt.Errorf("expected successful outcome for E2 setup response")
	}

	setupResp, ok := pdu.SuccessfulOutcome.Value.(*E2SetupResponse)
	if !ok {
		return fmt.Errorf("failed to extract E2SetupResponse from PDU")
	}

	return w.pool.messageHandler.HandleE2SetupResponse(msg.ConnectionID, msg.NodeID, setupResp)
}

// handleE2SetupFailure handles E2 Setup Failure messages
func (w *Worker) handleE2SetupFailure(msg *E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode E2 setup failure: %w", err)
	}

	if pdu.UnsuccessfulOutcome == nil {
		return fmt.Errorf("expected unsuccessful outcome for E2 setup failure")
	}

	setupFailure, ok := pdu.UnsuccessfulOutcome.Value.(*E2SetupFailure)
	if !ok {
		return fmt.Errorf("failed to extract E2SetupFailure from PDU")
	}

	return w.pool.messageHandler.HandleE2SetupFailure(msg.ConnectionID, msg.NodeID, setupFailure)
}

// handleRICSubscriptionRequest handles RIC Subscription Request messages
func (w *Worker) handleRICSubscriptionRequest(msg *E2Message) (interface{}, error) {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode RIC subscription request: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return nil, fmt.Errorf("expected initiating message for RIC subscription request")
	}

	subReq, ok := pdu.InitiatingMessage.Value.(*RICSubscriptionRequest)
	if !ok {
		return nil, fmt.Errorf("failed to extract RICSubscriptionRequest from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionRequest(msg.ConnectionID, msg.NodeID, subReq)
}

// handleRICSubscriptionResponse handles RIC Subscription Response messages
func (w *Worker) handleRICSubscriptionResponse(msg *E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC subscription response: %w", err)
	}

	if pdu.SuccessfulOutcome == nil {
		return fmt.Errorf("expected successful outcome for RIC subscription response")
	}

	subResp, ok := pdu.SuccessfulOutcome.Value.(*RICSubscriptionResponse)
	if !ok {
		return fmt.Errorf("failed to extract RICSubscriptionResponse from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionResponse(msg.ConnectionID, msg.NodeID, subResp)
}

// handleRICSubscriptionFailure handles RIC Subscription Failure messages
func (w *Worker) handleRICSubscriptionFailure(msg *E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC subscription failure: %w", err)
	}

	if pdu.UnsuccessfulOutcome == nil {
		return fmt.Errorf("expected unsuccessful outcome for RIC subscription failure")
	}

	subFailure, ok := pdu.UnsuccessfulOutcome.Value.(*RICSubscriptionFailure)
	if !ok {
		return fmt.Errorf("failed to extract RICSubscriptionFailure from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionFailure(msg.ConnectionID, msg.NodeID, subFailure)
}

// handleRICSubscriptionDeleteRequest handles RIC Subscription Delete Request messages
func (w *Worker) handleRICSubscriptionDeleteRequest(msg *E2Message) (interface{}, error) {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode RIC subscription delete request: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return nil, fmt.Errorf("expected initiating message for RIC subscription delete request")
	}

	delReq, ok := pdu.InitiatingMessage.Value.(*RICSubscriptionDeleteRequest)
	if !ok {
		return nil, fmt.Errorf("failed to extract RICSubscriptionDeleteRequest from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionDeleteRequest(msg.ConnectionID, msg.NodeID, delReq)
}

// handleRICSubscriptionDeleteResponse handles RIC Subscription Delete Response messages
func (w *Worker) handleRICSubscriptionDeleteResponse(msg *E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC subscription delete response: %w", err)
	}

	if pdu.SuccessfulOutcome == nil {
		return fmt.Errorf("expected successful outcome for RIC subscription delete response")
	}

	delResp, ok := pdu.SuccessfulOutcome.Value.(*RICSubscriptionDeleteResponse)
	if !ok {
		return fmt.Errorf("failed to extract RICSubscriptionDeleteResponse from PDU")
	}

	return w.pool.messageHandler.HandleRICSubscriptionDeleteResponse(msg.ConnectionID, msg.NodeID, delResp)
}

// handleRICIndication handles RIC Indication messages
func (w *Worker) handleRICIndication(msg *E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC indication: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return fmt.Errorf("expected initiating message for RIC indication")
	}

	indication, ok := pdu.InitiatingMessage.Value.(*RICIndication)
	if !ok {
		return fmt.Errorf("failed to extract RICIndication from PDU")
	}

	return w.pool.messageHandler.HandleRICIndication(msg.ConnectionID, msg.NodeID, indication)
}

// handleRICControlRequest handles RIC Control Request messages
func (w *Worker) handleRICControlRequest(msg *E2Message) (interface{}, error) {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode RIC control request: %w", err)
	}

	if pdu.InitiatingMessage == nil {
		return nil, fmt.Errorf("expected initiating message for RIC control request")
	}

	controlReq, ok := pdu.InitiatingMessage.Value.(*RICControlRequest)
	if !ok {
		return nil, fmt.Errorf("failed to extract RICControlRequest from PDU")
	}

	return w.pool.messageHandler.HandleRICControlRequest(msg.ConnectionID, msg.NodeID, controlReq)
}

// handleRICControlAck handles RIC Control Acknowledge messages
func (w *Worker) handleRICControlAck(msg *E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC control ack: %w", err)
	}

	if pdu.SuccessfulOutcome == nil {
		return fmt.Errorf("expected successful outcome for RIC control ack")
	}

	controlAck, ok := pdu.SuccessfulOutcome.Value.(*RICControlAck)
	if !ok {
		return fmt.Errorf("failed to extract RICControlAck from PDU")
	}

	return w.pool.messageHandler.HandleRICControlAck(msg.ConnectionID, msg.NodeID, controlAck)
}

// handleRICControlFailure handles RIC Control Failure messages
func (w *Worker) handleRICControlFailure(msg *E2Message) error {
	pdu, err := w.pool.messageHandler.(*E2Interface).codec.DecodeE2AP_PDU(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to decode RIC control failure: %w", err)
	}

	if pdu.UnsuccessfulOutcome == nil {
		return fmt.Errorf("expected unsuccessful outcome for RIC control failure")
	}

	controlFailure, ok := pdu.UnsuccessfulOutcome.Value.(*RICControlFailure)
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
func (wp *WorkerPool) processResult(result *MessageResult) {
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