// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// mobilex-demo demonstrates mobile-optimized mining with dual-algorithm support
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/toole-brendan/shell/chaincfg"
	"github.com/toole-brendan/shell/mining"
	"github.com/toole-brendan/shell/mining/mobilex"
	"github.com/toole-brendan/shell/mining/randomx"
	"github.com/toole-brendan/shell/wire"
)

var (
	// Command line flags
	durationFlag  = flag.Duration("duration", 60*time.Second, "Mining duration")
	intensityFlag = flag.Int("intensity", 2, "Mining intensity (1=light, 2=medium, 3=full)")
	algorithmFlag = flag.String("algorithm", "dual", "Mining algorithm (randomx, mobilex, dual)")
	deviceFlag    = flag.String("device", "flagship", "Device class (budget, midrange, flagship)")
	verboseFlag   = flag.Bool("verbose", false, "Verbose logging")
	npuFlag       = flag.Bool("npu", true, "Enable NPU if available")
	thermalFlag   = flag.Float64("thermal-limit", 45.0, "Thermal limit in Celsius")
)

func main() {
	flag.Parse()

	fmt.Println("Shell Reserve - Mobile Mining Demo")
	fmt.Println("==================================")

	// Create demo configuration
	config := createDemoConfig()

	// Display configuration
	displayConfig(config)

	// Initialize mining components
	miner, err := initializeMiner(config)
	if err != nil {
		log.Fatalf("Failed to initialize miner: %v", err)
	}
	defer miner.Close()

	// Create demo block for mining
	demoBlock := createDemoBlock()

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *durationFlag)
	defer cancel()

	// Setup interrupt handling
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Start mining demonstration
	fmt.Println("\nStarting mobile mining demonstration...")
	fmt.Println("Press Ctrl+C to stop early")
	fmt.Println()

	// Start the mining process
	results := runMiningDemo(ctx, miner, demoBlock, interrupt)

	// Display results
	displayResults(results)
}

// DemoConfig holds the demonstration configuration
type DemoConfig struct {
	Duration     time.Duration
	Intensity    int
	Algorithm    string
	DeviceClass  string
	NPUEnabled   bool
	ThermalLimit float64
	Verbose      bool
}

// MiningResults holds the mining demonstration results
type MiningResults struct {
	Duration         time.Duration
	TotalHashes      uint64
	RandomXHashes    uint64
	MobileXHashes    uint64
	HashRate         float64
	AvgTemperature   float64
	MaxTemperature   float64
	NPUUtilization   float64
	ThermalThrottles int
	BlocksFound      int
	Success          bool
}

func createDemoConfig() *DemoConfig {
	return &DemoConfig{
		Duration:     *durationFlag,
		Intensity:    *intensityFlag,
		Algorithm:    *algorithmFlag,
		DeviceClass:  *deviceFlag,
		NPUEnabled:   *npuFlag,
		ThermalLimit: *thermalFlag,
		Verbose:      *verboseFlag,
	}
}

func displayConfig(config *DemoConfig) {
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Duration:        %v\n", config.Duration)
	fmt.Printf("  Intensity:       %d (%s)\n", config.Intensity, getIntensityName(config.Intensity))
	fmt.Printf("  Algorithm:       %s\n", config.Algorithm)
	fmt.Printf("  Device Class:    %s\n", config.DeviceClass)
	fmt.Printf("  NPU Enabled:     %v\n", config.NPUEnabled)
	fmt.Printf("  Thermal Limit:   %.1f°C\n", config.ThermalLimit)
	fmt.Printf("  Verbose:         %v\n", config.Verbose)
	fmt.Println()
}

func getIntensityName(intensity int) string {
	switch intensity {
	case 1:
		return "Light"
	case 2:
		return "Medium"
	case 3:
		return "Full"
	default:
		return "Unknown"
	}
}

func initializeMiner(config *DemoConfig) (*DualAlgorithmMiner, error) {
	// Create network parameters
	params := &chaincfg.MainNetParams

	// Configure MobileX settings based on device class
	mobileXConfig := createMobileXConfig(config)

	// Create mobile miner
	mobileMiner, err := mobilex.NewMobileXMiner(mobileXConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create mobile miner: %w", err)
	}

	// Create RandomX miner with mobile integration
	var algorithm randomx.MiningAlgorithm
	switch config.Algorithm {
	case "randomx":
		algorithm = randomx.AlgorithmRandomX
	case "mobilex":
		algorithm = randomx.AlgorithmMobileX
	case "dual":
		algorithm = randomx.AlgorithmDual
	default:
		return nil, fmt.Errorf("unknown algorithm: %s", config.Algorithm)
	}

	// Create dual-algorithm miner
	randomXMiner := randomx.NewRandomXMinerWithMobile(
		getMemorySize(config.DeviceClass),
		&mobileXAdapter{mobileMiner}, // Wrap with adapter
		algorithm,
	)

	// Create mining policy
	policy := mining.NewMiningPolicy(params)

	return &DualAlgorithmMiner{
		randomXMiner: randomXMiner,
		mobileMiner:  mobileMiner,
		policy:       policy,
		config:       config,
	}, nil
}

// mobileXAdapter adapts MobileXMiner to the RandomX MobileMiner interface
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
	ctx := context.Background() // Use background context for interface compatibility
	return m.miner.Start(ctx)
}

func (m *mobileXAdapter) Stop() {
	m.miner.Stop()
}

func (m *mobileXAdapter) Close() {
	m.miner.Close()
}

func createMobileXConfig(config *DemoConfig) *mobilex.Config {
	return &mobilex.Config{
		// Basic configuration
		NPUEnabled:  config.NPUEnabled,
		UseNEON:     true, // Enable ARM64 optimizations
		NPUInterval: 150,  // NPU operations every 150 iterations

		// Device-specific settings
		BigCores:      getBigCores(config.DeviceClass),
		LittleCores:   getLittleCores(config.DeviceClass),
		RandomXMemory: uint64(getMemorySize(config.DeviceClass) * 1024 * 1024),

		// Thermal management
		ThermalThrottleStart:    float64(config.ThermalLimit - 5.0),
		ThermalThrottleStop:     float64(config.ThermalLimit),
		OptimalOperatingTemp:    35.0,
		ThermalTolerancePercent: 5,

		// NPU settings
		NPUModelPath: "/tmp/mobilex_model.bin", // Demo model path
	}
}

func getBigCores(deviceClass string) int {
	switch deviceClass {
	case "budget":
		return 2
	case "midrange":
		return 4
	case "flagship":
		return 4
	default:
		return 2
	}
}

func getLittleCores(deviceClass string) int {
	switch deviceClass {
	case "budget":
		return 4
	case "midrange":
		return 4
	case "flagship":
		return 4
	default:
		return 4
	}
}

func getMemorySize(deviceClass string) int64 {
	switch deviceClass {
	case "budget":
		return 256 // 256MB light mode
	case "midrange":
		return 512 // 512MB light mode
	case "flagship":
		return 2048 // 2GB fast mode
	default:
		return 256
	}
}

func createDemoBlock() *wire.MsgBlock {
	// Create a demo block for mining
	block := &wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:      1,
			PrevBlock:    [32]byte{}, // Zero hash for demo
			MerkleRoot:   [32]byte{}, // Zero hash for demo
			Timestamp:    time.Now(),
			Bits:         0x1e0ffff0, // Easy difficulty for demo
			Nonce:        0,
			ThermalProof: 0, // Will be set by MobileX if used
		},
		Transactions: []*wire.MsgTx{
			// Demo coinbase transaction
			{
				Version: 1,
				TxIn: []*wire.TxIn{{
					PreviousOutPoint: wire.OutPoint{Index: 0xffffffff},
					SignatureScript:  []byte("Shell Mobile Mining Demo"),
				}},
				TxOut: []*wire.TxOut{{
					Value:    5000000000,               // 50 XSL reward
					PkScript: []byte{0x76, 0xa9, 0x14}, // Demo script
				}},
			},
		},
	}

	return block
}

func runMiningDemo(ctx context.Context, miner *DualAlgorithmMiner, block *wire.MsgBlock, interrupt chan os.Signal) *MiningResults {
	results := &MiningResults{
		Duration: *durationFlag,
	}

	// Start mining
	if err := miner.Start(ctx); err != nil {
		log.Printf("Failed to start miner: %v", err)
		return results
	}

	// Statistics tracking
	startTime := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Printf("%-10s %-15s %-15s %-15s %-10s %-10s\n",
		"Time", "Total H/s", "RandomX H/s", "MobileX H/s", "Temp°C", "NPU%")
	fmt.Println(string(make([]byte, 80, 80)))

	quit := make(chan struct{})

	// Mining loop
	go func() {
		defer close(quit)

		miningTicker := time.NewTicker(100 * time.Millisecond)
		defer miningTicker.Stop()

		found, err := miner.SolveBlock(block, 1, miningTicker, quit)
		if err != nil {
			log.Printf("Mining error: %v", err)
		}
		if found {
			results.BlocksFound++
			fmt.Println("\n🎉 Block found!")
		}
	}()

	// Statistics loop
	for {
		select {
		case <-ctx.Done():
			close(quit)
			results.Duration = time.Since(startTime)
			results.Success = true
			return results

		case <-interrupt:
			fmt.Println("\nInterrupted by user")
			close(quit)
			results.Duration = time.Since(startTime)
			results.Success = false
			return results

		case <-ticker.C:
			elapsed := time.Since(startTime)

			// Get mining statistics
			randomXRate, mobileXRate, totalRate := miner.GetCombinedHashRate()
			temp := miner.GetTemperature()
			npuUtil := miner.GetNPUUtilization()

			// Update results
			results.TotalHashes += uint64(totalRate * 5) // Approximate hashes in 5 seconds
			results.RandomXHashes += uint64(randomXRate * 5)
			results.MobileXHashes += uint64(mobileXRate * 5)
			results.HashRate = totalRate
			results.AvgTemperature = (results.AvgTemperature + temp) / 2
			if temp > results.MaxTemperature {
				results.MaxTemperature = temp
			}
			results.NPUUtilization = npuUtil

			// Display current stats
			fmt.Printf("%-10s %-15.1f %-15.1f %-15.1f %-10.1f %-10.1f\n",
				formatDuration(elapsed), totalRate, randomXRate, mobileXRate, temp, npuUtil)

		case <-quit:
			results.Duration = time.Since(startTime)
			results.Success = true
			return results
		}
	}
}

func displayResults(results *MiningResults) {
	fmt.Println("\n" + string(make([]byte, 50, 50)))
	fmt.Println("Mining Demo Results")
	fmt.Println(string(make([]byte, 50, 50)))

	fmt.Printf("Status:           %s\n", func() string {
		if results.Success {
			return "✅ Completed"
		}
		return "❌ Interrupted"
	}())

	fmt.Printf("Duration:         %v\n", results.Duration)
	fmt.Printf("Total Hashes:     %d\n", results.TotalHashes)
	fmt.Printf("  RandomX:        %d (%.1f%%)\n", results.RandomXHashes,
		float64(results.RandomXHashes)/float64(results.TotalHashes)*100)
	fmt.Printf("  MobileX:        %d (%.1f%%)\n", results.MobileXHashes,
		float64(results.MobileXHashes)/float64(results.TotalHashes)*100)
	fmt.Printf("Average Hash Rate: %.1f H/s\n", results.HashRate)
	fmt.Printf("Temperature:      %.1f°C (avg), %.1f°C (max)\n",
		results.AvgTemperature, results.MaxTemperature)
	fmt.Printf("NPU Utilization:  %.1f%%\n", results.NPUUtilization)
	fmt.Printf("Thermal Throttles: %d\n", results.ThermalThrottles)
	fmt.Printf("Blocks Found:     %d\n", results.BlocksFound)

	if results.BlocksFound > 0 {
		fmt.Println("\n🎉 Congratulations! You found blocks during the demo!")
	}

	fmt.Println("\nDemo completed successfully!")
	fmt.Println("This demonstrates Shell Reserve's mobile mining capabilities.")
}

func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// DualAlgorithmMiner combines RandomX and MobileX miners
type DualAlgorithmMiner struct {
	randomXMiner *randomx.RandomXMiner
	mobileMiner  *mobilex.MobileXMiner
	policy       *mining.MiningPolicy
	config       *DemoConfig
}

func (dam *DualAlgorithmMiner) Start(ctx context.Context) error {
	// Start the RandomX miner (which includes mobile integration)
	// We'll use a dummy config for the demo
	config := &randomx.Config{
		NumWorkers: uint32(dam.config.Intensity),
	}

	dam.randomXMiner.Start(config)
	return nil
}

func (dam *DualAlgorithmMiner) SolveBlock(block *wire.MsgBlock, height int32, ticker *time.Ticker, quit chan struct{}) (bool, error) {
	// This would normally integrate with the actual mining logic
	// For demo purposes, we'll simulate mining

	select {
	case <-quit:
		return false, nil
	case <-ticker.C:
		// Simulate mining work
		// In real implementation, this would call the actual mining functions
		return false, nil
	}
}

func (dam *DualAlgorithmMiner) GetCombinedHashRate() (randomX, mobileX, total float64) {
	// Simulate hash rates based on configuration
	switch dam.config.DeviceClass {
	case "budget":
		randomX = 15.0 * float64(dam.config.Intensity)
		mobileX = 25.0 * float64(dam.config.Intensity)
	case "midrange":
		randomX = 30.0 * float64(dam.config.Intensity)
		mobileX = 60.0 * float64(dam.config.Intensity)
	case "flagship":
		randomX = 50.0 * float64(dam.config.Intensity)
		mobileX = 120.0 * float64(dam.config.Intensity)
	}

	// Apply NPU bonus for MobileX
	if dam.config.NPUEnabled {
		mobileX *= 1.3 // 30% NPU bonus
	}

	total = randomX + mobileX
	return
}

func (dam *DualAlgorithmMiner) GetTemperature() float64 {
	// Simulate temperature based on intensity and time
	baseTemp := 30.0
	intensityHeat := float64(dam.config.Intensity) * 5.0

	// Add some variation
	variation := float64(time.Now().Unix()%10) * 0.5

	return baseTemp + intensityHeat + variation
}

func (dam *DualAlgorithmMiner) GetNPUUtilization() float64 {
	if !dam.config.NPUEnabled {
		return 0.0
	}

	// Simulate NPU utilization
	baseUtil := 60.0
	intensityBonus := float64(dam.config.Intensity) * 10.0
	variation := float64(time.Now().Unix()%20) * 1.0

	util := baseUtil + intensityBonus + variation
	if util > 100.0 {
		util = 100.0
	}

	return util
}

func (dam *DualAlgorithmMiner) Close() {
	if dam.randomXMiner != nil {
		dam.randomXMiner.Stop()
	}
	if dam.mobileMiner != nil {
		dam.mobileMiner.Close()
	}
}
