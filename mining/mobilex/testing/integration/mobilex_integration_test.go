// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/mining"
	"github.com/toole-brendan/shell/mining/mobilex"
	"github.com/toole-brendan/shell/mining/randomx"
	"github.com/toole-brendan/shell/wire"
)

// TestMobileXIntegration tests the full mobile mining integration
func TestMobileXIntegration(t *testing.T) {
	t.Run("MobileX_Configuration", testMobileXConfiguration)
	t.Run("MobileX_BlockValidation", testMobileXBlockValidation)
	t.Run("DualAlgorithm_Mining", testDualAlgorithmMining)
	t.Run("Mining_Policy", testMiningPolicy)
	t.Run("Thermal_Validation", testThermalValidation)
}

func testMobileXConfiguration(t *testing.T) {
	// Test MobileX configuration creation and validation
	config := &mobilex.Config{
		NPUEnabled:              true,
		UseNEON:                 true,
		NPUInterval:             150,
		BigCores:                4,
		LittleCores:             4,
		RandomXMemory:           2 * 1024 * 1024 * 1024, // 2GB
		ThermalThrottleStart:    40.0,
		ThermalThrottleStop:     45.0,
		OptimalOperatingTemp:    35.0,
		ThermalTolerancePercent: 5,
		NPUModelPath:            "/tmp/test_model.bin",
	}

	// Create MobileX miner
	miner, err := mobilex.NewMobileXMiner(config)
	if err != nil {
		t.Fatalf("Failed to create MobileX miner: %v", err)
	}
	defer miner.Close()

	// Verify miner was created successfully
	if miner == nil {
		t.Fatal("MobileX miner is nil")
	}

	t.Logf("✅ MobileX miner created successfully with config: %+v", config)
}

func testMobileXBlockValidation(t *testing.T) {
	// Create test block with thermal proof
	block := createTestBlock(t, true) // with thermal proof

	// Create mining policy
	params := &chaincfg.MainNetParams
	params.MobileXEnabled = true
	policy := mining.NewMiningPolicy(params)

	// Test algorithm detection
	algorithm := policy.DetectAlgorithm(&block.Header)
	if algorithm != mining.AlgorithmMobileX {
		t.Errorf("Expected MobileX algorithm, got %v", algorithm)
	}

	// Test block validation
	err := policy.ValidateBlockAlgorithm(&block.Header, 1000)
	if err != nil {
		t.Errorf("MobileX block validation failed: %v", err)
	}

	t.Logf("✅ MobileX block validation passed")
}

func testDualAlgorithmMining(t *testing.T) {
	// Create MobileX miner
	mobileConfig := &mobilex.Config{
		NPUEnabled:              false, // Disable NPU for test
		UseNEON:                 false, // Disable NEON for test
		NPUInterval:             150,
		BigCores:                2,
		LittleCores:             2,
		RandomXMemory:           256 * 1024 * 1024, // 256MB light mode
		ThermalThrottleStart:    40.0,
		ThermalThrottleStop:     45.0,
		OptimalOperatingTemp:    35.0,
		ThermalTolerancePercent: 5,
		NPUModelPath:            "",
	}

	mobileMiner, err := mobilex.NewMobileXMiner(mobileConfig)
	if err != nil {
		t.Fatalf("Failed to create mobile miner: %v", err)
	}
	defer mobileMiner.Close()

	// Create adapter for interface compatibility
	adapter := &mobileXAdapter{mobileMiner}

	// Create dual-algorithm RandomX miner
	randomXMiner := randomx.NewRandomXMinerWithMobile(
		256, // 256MB memory
		adapter,
		randomx.AlgorithmDual,
	)

	// Test algorithm detection
	if randomXMiner.GetAlgorithm() != randomx.AlgorithmDual {
		t.Errorf("Expected dual algorithm, got %v", randomXMiner.GetAlgorithm())
	}

	// Test MobileX status
	if !randomXMiner.IsMobileXEnabled() {
		t.Error("MobileX should be enabled for dual mining")
	}

	// Test hash rate retrieval
	randomXRate, mobileXRate, totalRate := randomXMiner.GetCombinedHashRate()
	t.Logf("Hash rates - RandomX: %.2f, MobileX: %.2f, Total: %.2f",
		randomXRate, mobileXRate, totalRate)

	t.Logf("✅ Dual-algorithm mining setup successful")
}

func testMiningPolicy(t *testing.T) {
	// Test policy with MobileX disabled
	params := &chaincfg.MainNetParams
	params.MobileXEnabled = false

	policy := mining.NewMiningPolicy(params)

	// Should support only RandomX
	algorithms := policy.GetSupportedAlgorithms()
	if len(algorithms) != 1 || algorithms[0] != mining.AlgorithmRandomX {
		t.Errorf("Expected only RandomX, got %v", algorithms)
	}

	// Test ratio calculation
	randomXPercent, mobileXPercent := policy.GetAlgorithmRatio()
	if randomXPercent != 100.0 || mobileXPercent != 0.0 {
		t.Errorf("Expected 100/0 ratio, got %.1f/%.1f", randomXPercent, mobileXPercent)
	}

	// Enable MobileX
	policy.UpdateMobileXStatus(true)

	// Should now support dual mining
	if !policy.IsDualMiningEnabled() {
		t.Error("Dual mining should be enabled")
	}

	// Test dual ratio
	randomXPercent, mobileXPercent = policy.GetAlgorithmRatio()
	if randomXPercent != 50.0 || mobileXPercent != 50.0 {
		t.Errorf("Expected 50/50 ratio, got %.1f/%.1f", randomXPercent, mobileXPercent)
	}

	t.Logf("✅ Mining policy tests passed")
}

func testThermalValidation(t *testing.T) {
	// Create blocks with and without thermal proof
	validBlock := createTestBlock(t, true)    // with thermal proof
	invalidBlock := createTestBlock(t, false) // without thermal proof

	// Create policy with MobileX enabled
	params := &chaincfg.MainNetParams
	params.MobileXEnabled = true
	policy := mining.NewMiningPolicy(params)

	// Valid MobileX block should pass
	err := policy.ValidateBlockAlgorithm(&validBlock.Header, 1000)
	if err != nil {
		t.Errorf("Valid MobileX block should pass validation: %v", err)
	}

	// Block without thermal proof should be detected as RandomX
	algorithm := policy.DetectAlgorithm(&invalidBlock.Header)
	if algorithm != mining.AlgorithmRandomX {
		t.Errorf("Block without thermal proof should be RandomX, got %v", algorithm)
	}

	// Test thermal proof validation specifically
	validBlock.Header.ThermalProof = 0 // Remove thermal proof
	err = policy.ValidateBlockAlgorithm(&validBlock.Header, 1000)
	if err == nil {
		t.Error("Block without thermal proof should fail MobileX validation")
	}

	t.Logf("✅ Thermal validation tests passed")
}

// Helper functions

func createTestBlock(t *testing.T, withThermalProof bool) *wire.MsgBlock {
	block := &wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:      1,
			PrevBlock:    [32]byte{},
			MerkleRoot:   [32]byte{},
			Timestamp:    time.Now(),
			Bits:         0x1e0ffff0, // Easy difficulty
			Nonce:        12345,
			ThermalProof: 0,
		},
		Transactions: []*wire.MsgTx{
			{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: wire.OutPoint{Index: 0xffffffff},
					SignatureScript:  []byte("Test coinbase"),
				}},
				TxOut: []*wire.TxOut{{
					Value:    5000000000,
					PkScript: []byte{0x76, 0xa9, 0x14},
				}},
			},
		},
	}

	if withThermalProof {
		// Generate a valid thermal proof
		block.Header.ThermalProof = generateTestThermalProof(&block.Header)
	}

	return block
}

func generateTestThermalProof(header *wire.BlockHeader) uint64 {
	// Generate a simple test thermal proof
	// In real implementation, this would use proper thermal verification
	proof := uint64(header.Nonce)
	proof ^= uint64(header.Timestamp.Unix())
	proof &= 0xFFFFFFFFFFFF // Limit to reasonable range

	// Ensure it's non-zero for MobileX detection
	if proof == 0 {
		proof = 0x123456789ABC
	}

	return proof
}

// mobileXAdapter adapts MobileXMiner to MobileMiner interface for testing
type mobileXAdapter struct {
	miner *mobilex.MobileXMiner
}

func (m *mobileXAdapter) SolveBlock(msgBlock *wire.MsgBlock, blockHeight int32, ticker *time.Ticker, quit chan struct{}) (bool, error) {
	return m.miner.SolveBlock(msgBlock, blockHeight, ticker, quit)
}

func (m *mobileXAdapter) GetHashRate() float64 {
	return m.miner.GetHashRate()
}

func (m *mobileXAdapter) Start() error {
	ctx := context.Background()
	return m.miner.Start(ctx)
}

func (m *mobileXAdapter) Stop() {
	m.miner.Stop()
}

func (m *mobileXAdapter) Close() {
	m.miner.Close()
}

// BenchmarkMobileXIntegration benchmarks the mobile mining integration
func BenchmarkMobileXIntegration(b *testing.B) {
	config := &mobilex.Config{
		NPUEnabled:              false, // Disable NPU for consistent benchmarking
		UseNEON:                 false, // Disable NEON for consistent benchmarking
		NPUInterval:             150,
		BigCores:                2,
		LittleCores:             2,
		RandomXMemory:           256 * 1024 * 1024,
		ThermalThrottleStart:    40.0,
		ThermalThrottleStop:     45.0,
		OptimalOperatingTemp:    35.0,
		ThermalTolerancePercent: 5,
	}

	miner, err := mobilex.NewMobileXMiner(config)
	if err != nil {
		b.Fatalf("Failed to create miner: %v", err)
	}
	defer miner.Close()

	block := createTestBlock(nil, true)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Benchmark thermal proof generation
		block.Header.Nonce = uint32(i)
		block.Header.ThermalProof = generateTestThermalProof(&block.Header)

		// Benchmark policy validation
		params := &chaincfg.MainNetParams
		params.MobileXEnabled = true
		policy := mining.NewMiningPolicy(params)

		err := policy.ValidateBlockAlgorithm(&block.Header, 1000)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

// TestMobileXParameterValidation tests network parameter validation
func TestMobileXParameterValidation(t *testing.T) {
	params := &chaincfg.MainNetParams

	// Test default MobileX parameters
	if params.MobileXEnabled {
		t.Error("MobileX should be disabled by default")
	}

	if params.MobileXSeedRotation != 2048 {
		t.Errorf("Expected MobileX seed rotation 2048, got %d", params.MobileXSeedRotation)
	}

	if params.MobileXMemoryLightMode != 256*1024*1024 {
		t.Errorf("Expected 256MB light mode, got %d", params.MobileXMemoryLightMode)
	}

	if params.MobileXMemoryFastMode != 2*1024*1024*1024 {
		t.Errorf("Expected 2GB fast mode, got %d", params.MobileXMemoryFastMode)
	}

	if params.MobileXNPUInterval != 150 {
		t.Errorf("Expected NPU interval 150, got %d", params.MobileXNPUInterval)
	}

	if params.MobileXThermalTolerance != 5 {
		t.Errorf("Expected thermal tolerance 5%%, got %d%%", params.MobileXThermalTolerance)
	}

	t.Logf("✅ MobileX parameter validation passed")
}
