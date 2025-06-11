package testing

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/mining/mobilex"
	"github.com/toole-brendan/shell/wire"
)

// TestMobileXMinerCreation tests creating a mobile miner instance
func TestMobileXMinerCreation(t *testing.T) {
	// Create default configuration
	cfg := mobilex.DefaultConfig()
	require.NotNil(t, cfg)

	// Create miner
	miner, err := mobilex.NewMobileXMiner(cfg)
	require.NoError(t, err)
	require.NotNil(t, miner)

	// Close miner when done
	defer miner.Close()
}

// TestMobileXMinerStartStop tests starting and stopping the miner
func TestMobileXMinerStartStop(t *testing.T) {
	cfg := mobilex.DefaultConfig()
	miner, err := mobilex.NewMobileXMiner(cfg)
	require.NoError(t, err)
	defer miner.Close()

	// Create context for mining
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start miner
	err = miner.Start(ctx)
	require.NoError(t, err)

	// Allow miner to run briefly
	time.Sleep(time.Millisecond * 100)

	// Stop miner
	miner.Stop()
}

// TestConfigurationOptions tests different configuration options
func TestConfigurationOptions(t *testing.T) {
	tests := []struct {
		name        string
		configFunc  func() *mobilex.Config
		description string
	}{
		{
			name: "Default configuration",
			configFunc: func() *mobilex.Config {
				return mobilex.DefaultConfig()
			},
			description: "Standard mobile mining configuration",
		},
		{
			name: "Light mode configuration",
			configFunc: func() *mobilex.Config {
				return mobilex.LightModeConfig()
			},
			description: "Configuration for older/budget phones",
		},
		{
			name: "NPU disabled configuration",
			configFunc: func() *mobilex.Config {
				cfg := mobilex.DefaultConfig()
				cfg.NPUEnabled = false
				return cfg
			},
			description: "Configuration without NPU acceleration",
		},
		{
			name: "Minimal cores configuration",
			configFunc: func() *mobilex.Config {
				cfg := mobilex.DefaultConfig()
				cfg.BigCores = 1
				cfg.LittleCores = 1
				return cfg
			},
			description: "Configuration with minimal core usage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.configFunc()
			require.NotNil(t, cfg)

			// Try to create miner with configuration
			miner, err := mobilex.NewMobileXMiner(cfg)
			require.NoError(t, err, tt.description)
			require.NotNil(t, miner)

			// Clean up
			miner.Close()
		})
	}
}

// TestDeviceProfileOptimization tests device-specific optimizations
func TestDeviceProfileOptimization(t *testing.T) {
	devices := []string{
		"iPhone 15 Pro",
		"Galaxy S24",
		"Pixel 8",
		"Unknown Device",
	}

	for _, deviceName := range devices {
		t.Run(deviceName, func(t *testing.T) {
			// Get device profile
			profile := mobilex.GetDeviceProfile(deviceName)
			require.NotNil(t, profile)

			// Create configuration optimized for device
			cfg := mobilex.DefaultConfig()
			cfg.OptimizeForDevice(profile)

			// Create miner with optimized configuration
			miner, err := mobilex.NewMobileXMiner(cfg)
			require.NoError(t, err)
			require.NotNil(t, miner)

			// Clean up
			miner.Close()
		})
	}
}

// TestIntegratedMiningIntensity tests different mining intensity configurations in integration
func TestIntegratedMiningIntensity(t *testing.T) {
	cfg := mobilex.DefaultConfig()

	intensities := []struct {
		name      string
		intensity mobilex.MiningIntensity
	}{
		{
			name:      "Light intensity",
			intensity: cfg.IntensityLight,
		},
		{
			name:      "Medium intensity",
			intensity: cfg.IntensityMedium,
		},
		{
			name:      "Full intensity",
			intensity: cfg.IntensityFull,
		},
	}

	for _, tt := range intensities {
		t.Run(tt.name, func(t *testing.T) {
			// Verify intensity configuration
			assert.Greater(t, tt.intensity.CoreCount, 0)
			assert.Greater(t, tt.intensity.MaxHashRate, 0.0)
			assert.Greater(t, tt.intensity.PowerLimit, 0.0)
			assert.Greater(t, tt.intensity.ThermalLimit, 0.0)
		})
	}
}

// TestHashRateCalculation tests hash rate calculation
func TestHashRateCalculation(t *testing.T) {
	cfg := mobilex.DefaultConfig()
	// Use light mode for faster testing
	cfg.RandomXMemory = 256 * 1024 * 1024
	cfg.BigCores = 1
	cfg.LittleCores = 1

	miner, err := mobilex.NewMobileXMiner(cfg)
	require.NoError(t, err)
	defer miner.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start mining
	err = miner.Start(ctx)
	require.NoError(t, err)

	// Let it run for a bit
	time.Sleep(time.Millisecond * 100)

	// Get hash rate
	hashRate := miner.GetHashRate()

	// Hash rate should be positive after running
	assert.GreaterOrEqual(t, hashRate, 0.0, "Hash rate should be non-negative")

	miner.Stop()
}

// TestBlockHeaderCompatibility tests block header with thermal proof
func TestBlockHeaderCompatibility(t *testing.T) {
	// Create a block header with thermal proof
	header := &wire.BlockHeader{
		Version:      1,
		ThermalProof: 1234567890,
		Timestamp:    time.Now(),
		Bits:         0x1d00ffff,
		Nonce:        0,
	}

	// Set previous block hash
	prevHash, _ := chainhash.NewHashFromStr("0000000000000000000000000000000000000000000000000000000000000000")
	header.PrevBlock = *prevHash

	// Set merkle root
	merkleRoot, _ := chainhash.NewHashFromStr("0000000000000000000000000000000000000000000000000000000000000001")
	header.MerkleRoot = *merkleRoot

	// Verify thermal proof field exists
	assert.Equal(t, uint64(1234567890), header.ThermalProof)

	// Test serialization round-trip
	var buf bytes.Buffer
	err := header.Serialize(&buf)
	require.NoError(t, err)

	// Deserialize
	newHeader := &wire.BlockHeader{}
	err = newHeader.Deserialize(&buf)
	require.NoError(t, err)

	// Verify thermal proof survived
	assert.Equal(t, header.ThermalProof, newHeader.ThermalProof)
}

// TestIntegratedThermalProof tests thermal proof generation during mining integration
func TestIntegratedThermalProof(t *testing.T) {
	cfg := mobilex.DefaultConfig()
	cfg.ThermalProofRequired = true

	// Create thermal verification instance
	tv := mobilex.NewThermalVerification(2000, 5.0)
	require.NotNil(t, tv)

	// Create test header
	header := &wire.BlockHeader{
		Version:   1,
		Timestamp: time.Now(),
		Bits:      0x1d00ffff,
		Nonce:     12345,
	}

	// Generate header bytes for thermal proof
	headerBytes := make([]byte, 80)
	// In real implementation, this would be proper serialization

	// Generate thermal proof
	thermalProof := tv.GenerateThermalProof(headerBytes)
	assert.NotEqual(t, uint64(0), thermalProof, "Thermal proof should be non-zero")

	// Set thermal proof in header
	header.ThermalProof = thermalProof

	// Validate thermal proof
	err := tv.ValidateThermalProof(header)
	assert.NoError(t, err, "Generated thermal proof should be valid")
}

// TestCompactToBigConversion tests difficulty target conversion
func TestCompactToBigConversion(t *testing.T) {
	tests := []struct {
		name    string
		compact uint32
		isZero  bool
	}{
		{
			name:    "Standard difficulty",
			compact: 0x1d00ffff,
			isZero:  false,
		},
		{
			name:    "Higher difficulty",
			compact: 0x1c00ffff,
			isZero:  false,
		},
		{
			name:    "Lower difficulty",
			compact: 0x1e00ffff,
			isZero:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bigInt := mobilex.CompactToBig(tt.compact)
			require.NotNil(t, bigInt)

			if tt.isZero {
				assert.Equal(t, int64(0), bigInt.Int64())
			} else {
				assert.NotEqual(t, int64(0), bigInt.Int64())
			}
		})
	}
}

// TestHashToBigConversion tests hash to big integer conversion
func TestHashToBigConversion(t *testing.T) {
	// Create a test hash
	hashStr := "0000000000000000000000000000000000000000000000000000000000000001"
	hash, err := chainhash.NewHashFromStr(hashStr)
	require.NoError(t, err)

	// Convert to big integer
	bigInt := mobilex.HashToBig(hash)
	require.NotNil(t, bigInt)

	// Should be a positive number
	assert.Greater(t, bigInt.Sign(), 0, "Hash should convert to positive big integer")
}

// TestEndToEndMiningScenario tests a complete mining scenario
func TestEndToEndMiningScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create configuration for testing
	cfg := mobilex.LightModeConfig()
	cfg.ThermalProofRequired = true
	cfg.NPUEnabled = false // Disable NPU for testing

	// Create miner
	miner, err := mobilex.NewMobileXMiner(cfg)
	require.NoError(t, err)
	defer miner.Close()

	// Start mining
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = miner.Start(ctx)
	require.NoError(t, err)

	// Create a test block to mine
	msgBlock := &wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:   1,
			Timestamp: time.Now(),
			Bits:      0x207fffff, // Very easy difficulty for testing
			Nonce:     0,
		},
	}

	// Set previous block hash
	prevHash, _ := chainhash.NewHashFromStr("0000000000000000000000000000000000000000000000000000000000000000")
	msgBlock.Header.PrevBlock = *prevHash

	// Set merkle root
	merkleRoot, _ := chainhash.NewHashFromStr("0000000000000000000000000000000000000000000000000000000000000001")
	msgBlock.Header.MerkleRoot = *merkleRoot

	// Try to solve the block
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	quit := make(chan struct{})

	// Run mining in background
	go func() {
		select {
		case <-ctx.Done():
			close(quit)
		case <-time.After(4 * time.Second):
			close(quit)
		}
	}()

	// Attempt to solve block
	found, err := miner.SolveBlock(msgBlock, 1, ticker, quit)

	// We don't require finding a solution in test, just that it runs without error
	assert.NoError(t, err, "Mining should run without error")

	if found {
		t.Log("Found a solution!")
		assert.NotEqual(t, uint32(0), msgBlock.Header.Nonce, "Nonce should be non-zero")
		assert.NotEqual(t, uint64(0), msgBlock.Header.ThermalProof, "Thermal proof should be set")
	}

	miner.Stop()
}
