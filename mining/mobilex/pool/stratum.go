// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/wire"
)

// StratumServer implements the Stratum protocol for mobile miners.
type StratumServer struct {
	cfg            *PoolConfig
	chainParams    *chaincfg.Params
	jobManager     *JobManager
	shareValidator *ShareValidator

	// Network
	listener     net.Listener
	clients      map[uint64]*StratumClient
	clientsMu    sync.RWMutex
	nextClientID uint64

	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// StratumClient represents a connected mobile miner.
type StratumClient struct {
	ID     uint64
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer

	// Client state
	subscribed bool
	authorized bool
	workerName string
	userAgent  string

	// Mobile-specific
	deviceType   string  // iOS, Android
	socModel     string  // Snapdragon 8 Gen 3, A17 Pro, etc.
	thermalLimit float64 // Max temperature
	npuCapable   bool

	// Mining state
	currentJob      *MiningJob
	submittedShares uint64
	acceptedShares  uint64
	rejectedShares  uint64
	lastShareTime   time.Time

	// Difficulty
	difficulty float64
	targetDiff float64

	// Metrics
	hashRate    float64
	temperature float64
	powerUsage  float64
}

// StratumMessage represents a Stratum protocol message.
type StratumMessage struct {
	ID     interface{}     `json:"id"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result interface{}     `json:"result,omitempty"`
	Error  *StratumError   `json:"error,omitempty"`
}

// StratumError represents a Stratum protocol error.
type StratumError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// MiningJob represents work for mobile miners.
type MiningJob struct {
	ID               string
	Height           int32
	PreviousHash     string
	CoinbaseValue    int64
	Target           string
	MobileDifficulty float64

	// Mobile-specific
	NPUWork       []byte         // Optional NPU computation work
	ThermalTarget float64        // Thermal compliance target
	WorkSize      WorkSizeConfig // Optimized for device class
}

// WorkSizeConfig defines work parameters for different device classes.
type WorkSizeConfig struct {
	SearchSpace   uint32 // Nonce search space size
	NPUIterations uint32 // How often to run NPU
	CacheSize     uint32 // Working set size
}

// NewStratumServer creates a new Stratum server for mobile miners.
func NewStratumServer(cfg *PoolConfig, chainParams *chaincfg.Params) (*StratumServer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &StratumServer{
		cfg:         cfg,
		chainParams: chainParams,
		clients:     make(map[uint64]*StratumClient),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize job manager
	s.jobManager = NewJobManager(cfg, chainParams)

	// Initialize share validator
	s.shareValidator = NewShareValidator(cfg, chainParams)

	return s, nil
}

// Start begins listening for mobile miner connections.
func (s *StratumServer) Start() error {
	listener, err := net.Listen("tcp", s.cfg.StratumEndpoint)
	if err != nil {
		return fmt.Errorf("failed to start stratum server: %w", err)
	}

	s.listener = listener

	// Start job manager
	s.wg.Add(1)
	go s.jobManager.Start(s.ctx, &s.wg)

	// Start accepting connections
	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Stop gracefully shuts down the Stratum server.
func (s *StratumServer) Stop() {
	s.cancel()

	if s.listener != nil {
		s.listener.Close()
	}

	// Close all client connections
	s.clientsMu.Lock()
	for _, client := range s.clients {
		client.conn.Close()
	}
	s.clientsMu.Unlock()

	s.wg.Wait()
}

// acceptConnections accepts new miner connections.
func (s *StratumServer) acceptConnections() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				continue
			}
		}

		// Create new client
		clientID := atomic.AddUint64(&s.nextClientID, 1)
		client := &StratumClient{
			ID:         clientID,
			conn:       conn,
			reader:     bufio.NewReader(conn),
			writer:     bufio.NewWriter(conn),
			difficulty: s.cfg.InitialDifficulty,
		}

		// Register client
		s.clientsMu.Lock()
		s.clients[clientID] = client
		s.clientsMu.Unlock()

		// Handle client in goroutine
		s.wg.Add(1)
		go s.handleClient(client)
	}
}

// handleClient handles a single miner connection.
func (s *StratumServer) handleClient(client *StratumClient) {
	defer s.wg.Done()
	defer s.removeClient(client)

	// Set connection timeout
	client.conn.SetDeadline(time.Now().Add(s.cfg.ConnectionTimeout))

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// Read message
			line, err := client.reader.ReadBytes('\n')
			if err != nil {
				return
			}

			// Reset deadline
			client.conn.SetDeadline(time.Now().Add(s.cfg.ConnectionTimeout))

			// Parse and handle message
			var msg StratumMessage
			if err := json.Unmarshal(line, &msg); err != nil {
				s.sendError(client, nil, -32700, "Parse error")
				continue
			}

			// Handle method
			if err := s.handleMethod(client, &msg); err != nil {
				s.sendError(client, msg.ID, -32601, err.Error())
			}
		}
	}
}

// handleMethod routes Stratum methods to handlers.
func (s *StratumServer) handleMethod(client *StratumClient, msg *StratumMessage) error {
	switch msg.Method {
	case "mining.subscribe":
		return s.handleSubscribe(client, msg)
	case "mining.authorize":
		return s.handleAuthorize(client, msg)
	case "mining.submit":
		return s.handleSubmit(client, msg)
	case "mining.get_transactions":
		return s.handleGetTransactions(client, msg)
	case "mining.extranonce.subscribe":
		return s.handleExtranonceSubscribe(client, msg)

	// Mobile-specific methods
	case "mining.set_device_info":
		return s.handleSetDeviceInfo(client, msg)
	case "mining.report_thermal":
		return s.handleReportThermal(client, msg)
	case "mining.get_mobile_config":
		return s.handleGetMobileConfig(client, msg)

	default:
		return errors.New("unknown method")
	}
}

// handleSubscribe handles mining.subscribe requests.
func (s *StratumServer) handleSubscribe(client *StratumClient, msg *StratumMessage) error {
	// Parse params [userAgent, sessionID]
	var params []interface{}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	if len(params) > 0 {
		if ua, ok := params[0].(string); ok {
			client.userAgent = ua
		}
	}

	// Generate subscription ID and extranonce
	subID := fmt.Sprintf("%x", client.ID)
	extranonce1 := fmt.Sprintf("%08x", client.ID)

	// Mark as subscribed
	client.subscribed = true

	// Send response
	result := []interface{}{
		[][]string{
			{"mining.set_difficulty", subID},
			{"mining.notify", subID},
		},
		extranonce1,
		4, // extranonce2 size
	}

	return s.sendResult(client, msg.ID, result)
}

// handleAuthorize handles mining.authorize requests.
func (s *StratumServer) handleAuthorize(client *StratumClient, msg *StratumMessage) error {
	// Parse params [username, password]
	var params []string
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	if len(params) < 1 {
		return errors.New("missing username")
	}

	client.workerName = params[0]
	client.authorized = true

	// Send initial difficulty
	s.setDifficulty(client, client.difficulty)

	// Send current job
	if job := s.jobManager.GetCurrentJob(); job != nil {
		s.sendJob(client, job)
	}

	return s.sendResult(client, msg.ID, true)
}

// handleSubmit handles share submissions from mobile miners.
func (s *StratumServer) handleSubmit(client *StratumClient, msg *StratumMessage) error {
	// Parse params [worker, jobID, extranonce2, ntime, nonce, thermalProof]
	var params []string
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	if len(params) < 6 {
		return errors.New("invalid submit parameters")
	}

	share := &Share{
		ClientID:     client.ID,
		WorkerName:   params[0],
		JobID:        params[1],
		Extranonce2:  params[2],
		Ntime:        params[3],
		Nonce:        params[4],
		ThermalProof: params[5],
		Difficulty:   client.difficulty,
		SubmittedAt:  time.Now(),
	}

	// Validate share
	result, err := s.shareValidator.ValidateShare(share, client.currentJob)
	if err != nil {
		client.rejectedShares++
		return s.sendResult(client, msg.ID, false)
	}

	// Update client stats
	client.submittedShares++
	client.acceptedShares++
	client.lastShareTime = time.Now()

	// Adjust difficulty if needed
	s.adjustDifficulty(client)

	// Check if share meets network difficulty
	if result.MeetsNetworkDifficulty {
		// Submit block to network
		s.submitBlock(result.Block)
	}

	return s.sendResult(client, msg.ID, true)
}

// Mobile-specific method handlers

// handleSetDeviceInfo handles device information from mobile miners.
func (s *StratumServer) handleSetDeviceInfo(client *StratumClient, msg *StratumMessage) error {
	var info struct {
		DeviceType   string  `json:"device_type"`
		SocModel     string  `json:"soc_model"`
		ThermalLimit float64 `json:"thermal_limit"`
		NPUCapable   bool    `json:"npu_capable"`
		MaxCores     int     `json:"max_cores"`
		RAMSize      int     `json:"ram_size_mb"`
	}

	if err := json.Unmarshal(msg.Params, &info); err != nil {
		return err
	}

	// Update client info
	client.deviceType = info.DeviceType
	client.socModel = info.SocModel
	client.thermalLimit = info.ThermalLimit
	client.npuCapable = info.NPUCapable

	// Adjust work parameters based on device
	s.optimizeForDevice(client)

	return s.sendResult(client, msg.ID, true)
}

// handleReportThermal handles thermal status reports from mobile miners.
func (s *StratumServer) handleReportThermal(client *StratumClient, msg *StratumMessage) error {
	var report struct {
		Temperature float64 `json:"temperature"`
		PowerUsage  float64 `json:"power_usage"`
		HashRate    float64 `json:"hash_rate"`
		Throttled   bool    `json:"throttled"`
	}

	if err := json.Unmarshal(msg.Params, &report); err != nil {
		return err
	}

	// Update client metrics
	client.temperature = report.Temperature
	client.powerUsage = report.PowerUsage
	client.hashRate = report.HashRate

	// Adjust difficulty if thermal throttling
	if report.Throttled {
		newDiff := client.difficulty * 0.8
		s.setDifficulty(client, newDiff)
	}

	return s.sendResult(client, msg.ID, true)
}

// handleGetMobileConfig returns optimized mining configuration.
func (s *StratumServer) handleGetMobileConfig(client *StratumClient, msg *StratumMessage) error {
	config := s.getMobileConfig(client)
	return s.sendResult(client, msg.ID, config)
}

// Helper methods

// sendJob sends a new mining job to a client.
func (s *StratumServer) sendJob(client *StratumClient, job *MiningJob) {
	params := []interface{}{
		job.ID,
		job.PreviousHash,
		"",         // coinbase1 (handled by pool)
		"",         // coinbase2
		[]string{}, // merkle branches
		fmt.Sprintf("%08x", job.Height),
		job.Target,
		true, // clean jobs
		// Mobile-specific parameters
		map[string]interface{}{
			"thermal_target": job.ThermalTarget,
			"npu_work":       job.NPUWork,
			"work_size":      job.WorkSize,
		},
	}

	client.currentJob = job
	s.sendNotification(client, "mining.notify", params)
}

// setDifficulty updates client difficulty.
func (s *StratumServer) setDifficulty(client *StratumClient, difficulty float64) {
	client.difficulty = difficulty
	client.targetDiff = difficulty

	s.sendNotification(client, "mining.set_difficulty", []float64{difficulty})
}

// adjustDifficulty adjusts difficulty based on share submission rate.
func (s *StratumServer) adjustDifficulty(client *StratumClient) {
	// Mobile-aware difficulty adjustment
	// Target: 1 share per 30 seconds for mobile devices

	if client.lastShareTime.IsZero() {
		return
	}

	timeSinceLastShare := time.Since(client.lastShareTime).Seconds()

	if timeSinceLastShare < 20 {
		// Too fast, increase difficulty
		client.targetDiff = client.difficulty * 1.2
	} else if timeSinceLastShare > 40 {
		// Too slow, decrease difficulty
		client.targetDiff = client.difficulty * 0.8
	}

	// Apply smoothing
	newDiff := client.difficulty*0.7 + client.targetDiff*0.3

	// Enforce mobile-friendly bounds
	if newDiff < s.cfg.MinMobileDifficulty {
		newDiff = s.cfg.MinMobileDifficulty
	} else if newDiff > s.cfg.MaxMobileDifficulty {
		newDiff = s.cfg.MaxMobileDifficulty
	}

	if newDiff != client.difficulty {
		s.setDifficulty(client, newDiff)
	}
}

// optimizeForDevice adjusts mining parameters for device capabilities.
func (s *StratumServer) optimizeForDevice(client *StratumClient) {
	var workSize WorkSizeConfig

	switch client.socModel {
	case "Snapdragon 8 Gen 3", "A17 Pro", "Tensor G3":
		// Flagship devices
		workSize = WorkSizeConfig{
			SearchSpace:   0x100000,        // 1M nonces
			NPUIterations: 100,             // Frequent NPU use
			CacheSize:     3 * 1024 * 1024, // 3MB
		}
		client.difficulty = s.cfg.InitialDifficulty

	case "Snapdragon 7 Gen 3", "A16", "Tensor G2":
		// Mid-range devices
		workSize = WorkSizeConfig{
			SearchSpace:   0x80000,         // 512K nonces
			NPUIterations: 150,             // Less frequent NPU
			CacheSize:     2 * 1024 * 1024, // 2MB
		}
		client.difficulty = s.cfg.InitialDifficulty * 0.7

	default:
		// Budget devices
		workSize = WorkSizeConfig{
			SearchSpace:   0x40000,     // 256K nonces
			NPUIterations: 200,         // Minimal NPU use
			CacheSize:     1024 * 1024, // 1MB
		}
		client.difficulty = s.cfg.InitialDifficulty * 0.5
	}

	// Store optimized config
	if client.currentJob != nil {
		client.currentJob.WorkSize = workSize
	}
}

// getMobileConfig returns device-specific mining configuration.
func (s *StratumServer) getMobileConfig(client *StratumClient) map[string]interface{} {
	return map[string]interface{}{
		"mining_intensity": s.getRecommendedIntensity(client),
		"thermal_limits": map[string]float64{
			"throttle_start": 45.0,
			"throttle_stop":  50.0,
			"optimal":        40.0,
		},
		"npu_config": map[string]interface{}{
			"enabled":    client.npuCapable,
			"iterations": client.currentJob.WorkSize.NPUIterations,
		},
		"power_management": map[string]interface{}{
			"charge_only":     true,
			"min_battery":     80,
			"screen_off_only": false,
		},
	}
}

// getRecommendedIntensity returns recommended mining intensity.
func (s *StratumServer) getRecommendedIntensity(client *StratumClient) string {
	switch client.socModel {
	case "Snapdragon 8 Gen 3", "A17 Pro":
		return "high"
	case "Snapdragon 7 Gen 3", "A16":
		return "medium"
	default:
		return "low"
	}
}

// Communication methods

// sendResult sends a result response to client.
func (s *StratumServer) sendResult(client *StratumClient, id interface{}, result interface{}) error {
	msg := StratumMessage{
		ID:     id,
		Result: result,
		Error:  nil,
	}
	return s.sendMessage(client, &msg)
}

// sendError sends an error response to client.
func (s *StratumServer) sendError(client *StratumClient, id interface{}, code int, message string) error {
	msg := StratumMessage{
		ID:     id,
		Result: nil,
		Error: &StratumError{
			Code:    code,
			Message: message,
		},
	}
	return s.sendMessage(client, &msg)
}

// sendNotification sends a notification to client.
func (s *StratumServer) sendNotification(client *StratumClient, method string, params interface{}) error {
	msg := StratumMessage{
		ID:     nil,
		Method: method,
		Params: mustMarshalJSON(params),
	}
	return s.sendMessage(client, &msg)
}

// sendMessage sends a message to client.
func (s *StratumServer) sendMessage(client *StratumClient, msg *StratumMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	if _, err := client.writer.Write(data); err != nil {
		return err
	}

	return client.writer.Flush()
}

// removeClient removes a disconnected client.
func (s *StratumServer) removeClient(client *StratumClient) {
	client.conn.Close()

	s.clientsMu.Lock()
	delete(s.clients, client.ID)
	s.clientsMu.Unlock()
}

// submitBlock submits a found block to the network.
func (s *StratumServer) submitBlock(block *wire.MsgBlock) {
	// This would connect to the Shell node to submit the block
	// Implementation depends on node RPC interface
}

// handleGetTransactions handles mining.get_transactions requests (returns empty for now).
func (s *StratumServer) handleGetTransactions(client *StratumClient, msg *StratumMessage) error {
	// Mobile miners typically don't need transaction data
	// Return empty array
	return s.sendResult(client, msg.ID, []string{})
}

// handleExtranonceSubscribe handles mining.extranonce.subscribe requests.
func (s *StratumServer) handleExtranonceSubscribe(client *StratumClient, msg *StratumMessage) error {
	// This allows clients to be notified when extranonce changes
	// For now, we don't support this feature
	return s.sendResult(client, msg.ID, false)
}

// mustMarshalJSON marshals value to JSON or panics.
func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(data)
}
