// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mobilex

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/mining/mobilex/npu"
	"github.com/toole-brendan/shell/mining/mobilex/npu/fallback"
	"github.com/toole-brendan/shell/mining/randomx"
	"github.com/toole-brendan/shell/wire"
)

// MobileXMiner implements mobile-optimized mining with MobileX algorithm.
type MobileXMiner struct {
	cfg           *Config
	arm64         *ARM64Optimizer
	thermal       *ThermalVerification
	npu           *npu.NPUManager
	heterogeneous *HeterogeneousScheduler

	// RandomX components (integrated from existing implementation)
	cache   *randomx.Cache
	dataset *randomx.Dataset
	vm      *randomx.VM

	// Mining state
	mining          int32 // atomic flag
	hashesCompleted uint64
	bestHash        chainhash.Hash
	bestHashMutex   sync.RWMutex

	// Metrics
	startTime        time.Time
	metricsCollector *MetricsCollector
}

// NewMobileXMiner creates a new mobile-optimized miner.
func NewMobileXMiner(cfg *Config) (*MobileXMiner, error) {
	// Initialize ARM64 optimizations
	arm64 := NewARM64Optimizer()

	// Initialize thermal verification
	thermal := NewThermalVerification(2000, cfg.ThermalTolerancePercent) // 2GHz base freq

	// Initialize NPU manager
	npuAdapter := detectNPUAdapter() // Platform-specific detection
	npuConfig := &npu.ModelConfig{
		ModelPath:   cfg.NPUModelPath,
		InputShape:  []int{32, 32, 3},
		OutputShape: []int{32, 32, 3},
		Precision:   npu.Float32,
		BatchSize:   1,
		Priority:    npu.PriorityHigh,
		TimeoutMs:   100,
	}
	npuManager := npu.NewNPUManager(npuAdapter, npuConfig)

	// Set CPU fallback
	cpuFallback := fallback.NewCPUNeuralFallback()
	npuManager.SetFallback(cpuFallback.RunConvolution)

	// Initialize heterogeneous scheduler
	heterogeneous := NewHeterogeneousScheduler(cfg.BigCores, cfg.LittleCores)

	// Initialize RandomX components
	seed := make([]byte, 32) // Will be properly set during mining
	cache, err := randomx.NewCache(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create RandomX cache: %w", err)
	}

	// For mobile, we'll use light mode (cache only) by default
	// Full dataset mode can be enabled for devices with enough memory
	var dataset *randomx.Dataset
	var vm *randomx.VM

	if cfg.RandomXMemory >= 2*1024*1024*1024 {
		// Fast mode with full dataset
		dataset, err = randomx.NewDataset(cache)
		if err != nil {
			cache.Close()
			return nil, fmt.Errorf("failed to create RandomX dataset: %w", err)
		}
		vm, err = randomx.NewVM(cache, dataset)
	} else {
		// Light mode with cache only
		vm, err = randomx.NewVM(cache, nil)
	}

	if err != nil {
		if dataset != nil {
			dataset.Close()
		}
		cache.Close()
		return nil, fmt.Errorf("failed to create RandomX VM: %w", err)
	}

	miner := &MobileXMiner{
		cfg:              cfg,
		arm64:            arm64,
		thermal:          thermal,
		npu:              npuManager,
		heterogeneous:    heterogeneous,
		cache:            cache,
		dataset:          dataset,
		vm:               vm,
		metricsCollector: NewMetricsCollector(),
	}

	return miner, nil
}

// Start begins the mining process.
func (m *MobileXMiner) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&m.mining, 0, 1) {
		return errors.New("miner is already running")
	}

	m.startTime = time.Now()
	m.hashesCompleted = 0

	// Start heterogeneous core scheduling
	m.heterogeneous.Start()

	// Start thermal monitoring
	go m.thermalMonitoringLoop(ctx)

	// Start metrics collection
	go m.metricsCollector.Start(ctx)

	return nil
}

// Stop stops the mining process.
func (m *MobileXMiner) Stop() {
	atomic.StoreInt32(&m.mining, 0)
	m.heterogeneous.Stop()
	m.metricsCollector.Stop()
}

// Close releases all resources.
func (m *MobileXMiner) Close() {
	m.Stop()

	if m.vm != nil {
		m.vm.Close()
	}
	if m.dataset != nil {
		m.dataset.Close()
	}
	if m.cache != nil {
		m.cache.Close()
	}
}

// SolveBlock attempts to find a valid solution for the given block.
func (m *MobileXMiner) SolveBlock(msgBlock *wire.MsgBlock, blockHeight int32,
	ticker *time.Ticker, quit chan struct{}) (bool, error) {

	// Validate we're mining
	if atomic.LoadInt32(&m.mining) != 1 {
		return false, errors.New("miner not started")
	}

	// Get target difficulty
	targetDifficulty := CompactToBig(msgBlock.Header.Bits)

	// Main mining loop
	for {
		select {
		case <-quit:
			return false, nil
		case <-ticker.C:
			// Update metrics periodically
			m.updateMetrics()
		default:
			// Attempt to solve block
			found, err := m.mineIteration(&msgBlock.Header, targetDifficulty, blockHeight)
			if err != nil {
				return false, err
			}
			if found {
				return true, nil
			}
		}
	}
}

// mineIteration performs one mining iteration.
func (m *MobileXMiner) mineIteration(header *wire.BlockHeader, target *big.Int, blockHeight int32) (bool, error) {
	// Increment nonce
	header.Nonce++

	// Check if we need to run NPU operations
	if m.shouldRunNPU() {
		if err := m.runNPUStep(); err != nil {
			// Log error but continue mining
			m.metricsCollector.RecordError("npu_error", err)
		}
	}

	// Compute MobileX hash
	hash := m.computeMobileXHash(header)

	// Increment hash counter
	atomic.AddUint64(&m.hashesCompleted, 1)

	// Check if hash meets target
	hashBig := HashToBig(&hash)
	if hashBig.Cmp(target) <= 0 {
		// Found a solution! Generate thermal proof
		headerBytes := serializeBlockHeader(header)
		header.ThermalProof = m.thermal.GenerateThermalProof(headerBytes)

		// Update best hash
		m.updateBestHash(hash)

		return true, nil
	}

	return false, nil
}

// computeMobileXHash computes the MobileX hash for a block header.
func (m *MobileXMiner) computeMobileXHash(header *wire.BlockHeader) chainhash.Hash {
	// Serialize header
	headerBytes := serializeBlockHeader(header)

	// Apply ARM64 optimizations if available
	if m.cfg.UseNEON && m.arm64.HasNEON() {
		// Pre-process with NEON vector operations
		headerBytes = m.arm64.VectorHash(headerBytes)
	}

	// Run through RandomX VM
	vmOutput := m.vm.CalcHash(headerBytes)

	// Apply additional mobile-specific mixing
	// This ensures mobile hardware advantages
	mixed := m.applyMobileMixing(vmOutput)

	// Convert to chainhash.Hash
	var hash chainhash.Hash
	copy(hash[:], mixed)

	return hash
}

// applyMobileMixing applies mobile-specific mixing to the RandomX output.
func (m *MobileXMiner) applyMobileMixing(randomxHash []byte) []byte {
	// Convert to uint32s for ARM-specific operations
	uint32Data := bytesToUint32s(randomxHash)

	// Apply ARM-specific hash operations
	mixed := m.arm64.ARMSpecificHash(uint32Data)

	// Mix with heterogeneous core scheduling state
	coreState := m.heterogeneous.GetCoreState()
	for i := range mixed {
		mixed[i] ^= coreState
	}

	// Final hash
	finalBytes := uint32sToBytes(mixed)
	finalHash := sha256.Sum256(finalBytes)

	return finalHash[:]
}

// shouldRunNPU determines if NPU operations should run this iteration.
func (m *MobileXMiner) shouldRunNPU() bool {
	// Run NPU every N iterations based on configuration
	hashCount := atomic.LoadUint64(&m.hashesCompleted)
	return m.cfg.NPUEnabled && (hashCount%uint64(m.cfg.NPUInterval) == 0)
}

// runNPUStep executes NPU operations and feeds results back into mining.
func (m *MobileXMiner) runNPUStep() error {
	// Get current RandomX VM state (cache state for mixing)
	// In a real implementation, we would extract internal VM state
	// For now, use hash counter as pseudo-state
	vmState := make([]byte, 2048)
	binary.LittleEndian.PutUint64(vmState, atomic.LoadUint64(&m.hashesCompleted))

	// Hash the state to create more entropy
	stateHash := sha256.Sum256(vmState[:8])
	copy(vmState, stateHash[:])

	// Convert to tensor (32x32x3)
	tensor := stateToTensor(vmState)

	// Run through NPU
	output, err := m.npu.ExecuteConvolution(tensor)
	if err != nil {
		return fmt.Errorf("NPU execution failed: %w", err)
	}

	// Mix NPU results back into mining state
	// This affects future hash computations
	npuResult := tensorToState(output)
	m.mixNPUResults(npuResult)

	return nil
}

// mixNPUResults mixes NPU computation results into the mining process.
func (m *MobileXMiner) mixNPUResults(npuResult []byte) {
	// In a real implementation, this would modify RandomX VM state
	// For now, we'll use it to influence nonce selection
	if len(npuResult) >= 4 {
		// Use NPU result to skip certain nonce ranges
		// This simulates NPU influence on mining
		skip := binary.LittleEndian.Uint32(npuResult[:4]) % 1000
		atomic.AddUint64(&m.hashesCompleted, uint64(skip))
	}
}

// thermalMonitoringLoop continuously monitors thermal state.
func (m *MobileXMiner) thermalMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get current temperature (placeholder - would read from sensors)
			temp := m.readTemperature()
			m.thermal.UpdateTemperature(temp)

			// Check thermal limits
			if temp > m.cfg.ThermalThrottleStop {
				// Stop mining if too hot
				atomic.StoreInt32(&m.mining, 0)
			} else if temp > m.cfg.ThermalThrottleStart {
				// Reduce intensity
				m.heterogeneous.ReduceIntensity()
			} else if temp < m.cfg.OptimalOperatingTemp {
				// Can increase intensity
				m.heterogeneous.IncreaseIntensity()
			}
		}
	}
}

// readTemperature reads the current device temperature.
func (m *MobileXMiner) readTemperature() float64 {
	// In real implementation, this would read from:
	// - /sys/class/thermal/thermal_zone*/temp on Android
	// - IOKit on iOS
	// For now, return a simulated value
	return 40.0 + float64(atomic.LoadUint64(&m.hashesCompleted)%10)*0.5
}

// updateMetrics updates mining metrics.
func (m *MobileXMiner) updateMetrics() {
	elapsed := time.Since(m.startTime).Seconds()
	hashRate := float64(atomic.LoadUint64(&m.hashesCompleted)) / elapsed

	metrics := MiningMetrics{
		HashRate:        hashRate,
		HashesCompleted: atomic.LoadUint64(&m.hashesCompleted),
		Temperature:     m.thermal.getCurrentTemperature(),
		PowerUsage:      m.estimatePowerUsage(),
		NPUUtilization:  m.npu.GetAverageMetrics().Utilization,
		Duration:        time.Since(m.startTime),
	}

	m.metricsCollector.Record(metrics)
}

// estimatePowerUsage estimates current power consumption.
func (m *MobileXMiner) estimatePowerUsage() float64 {
	// Base CPU power
	cpuPower := float64(m.heterogeneous.ActiveCores()) * 0.5

	// NPU power
	npuPower := m.npu.GetAverageMetrics().PowerUsage

	// Total estimate
	return cpuPower + npuPower
}

// updateBestHash updates the best hash found.
func (m *MobileXMiner) updateBestHash(hash chainhash.Hash) {
	m.bestHashMutex.Lock()
	defer m.bestHashMutex.Unlock()
	m.bestHash = hash
}

// GetHashRate returns the current hash rate.
func (m *MobileXMiner) GetHashRate() float64 {
	elapsed := time.Since(m.startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(atomic.LoadUint64(&m.hashesCompleted)) / elapsed
}

// Helper functions

// serializeBlockHeader serializes a block header to bytes.
func serializeBlockHeader(header *wire.BlockHeader) []byte {
	// This is a simplified version - real implementation would use wire protocol
	var buf [88]byte
	binary.LittleEndian.PutUint32(buf[0:4], uint32(header.Version))
	copy(buf[4:36], header.PrevBlock[:])
	copy(buf[36:68], header.MerkleRoot[:])
	binary.LittleEndian.PutUint32(buf[68:72], uint32(header.Timestamp.Unix()))
	binary.LittleEndian.PutUint32(buf[72:76], header.Bits)
	binary.LittleEndian.PutUint32(buf[76:80], header.Nonce)
	binary.LittleEndian.PutUint64(buf[80:88], header.ThermalProof)
	return buf[:]
}

// stateToTensor converts VM state to a tensor for NPU processing.
func stateToTensor(state []byte) npu.Tensor {
	// Convert first 3072 bytes (32*32*3) to tensor
	data := make([]float32, 32*32*3)
	for i := 0; i < len(data) && i < len(state); i++ {
		data[i] = float32(state[i]) / 255.0
	}

	return npu.CreateTensor(data, []int{32, 32, 3})
}

// tensorToState converts tensor output back to VM state.
func tensorToState(tensor npu.Tensor) []byte {
	state := make([]byte, 2048)
	for i := 0; i < len(tensor.Data) && i < len(state); i++ {
		state[i] = byte(tensor.Data[i] * 255.0)
	}
	return state
}

// bytesToUint32s converts bytes to uint32 slice.
func bytesToUint32s(b []byte) []uint32 {
	result := make([]uint32, len(b)/4)
	for i := range result {
		result[i] = binary.LittleEndian.Uint32(b[i*4:])
	}
	return result
}

// uint32sToBytes converts uint32 slice to bytes.
func uint32sToBytes(u []uint32) []byte {
	result := make([]byte, len(u)*4)
	for i, v := range u {
		binary.LittleEndian.PutUint32(result[i*4:], v)
	}
	return result
}

// CompactToBig converts a compact representation to a big integer.
func CompactToBig(compact uint32) *big.Int {
	// This is the same as Bitcoin's compact target representation
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

// HashToBig converts a hash to a big integer.
func HashToBig(hash *chainhash.Hash) *big.Int {
	// Reverse the bytes because big integers are big endian
	var buf [32]byte
	for i := 0; i < 32; i++ {
		buf[i] = hash[31-i]
	}
	return new(big.Int).SetBytes(buf[:])
}
