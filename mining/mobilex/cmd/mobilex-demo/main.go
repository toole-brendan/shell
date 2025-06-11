// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// mobilex-demo is a command-line demonstration of MobileX mining on ARM64 devices.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/toole-brendan/shell/chaincfg/chainhash"
	"github.com/toole-brendan/shell/mining/mobilex"
	"github.com/toole-brendan/shell/wire"
)

func main() {
	// Command-line flags
	var (
		intensity  = flag.Int("intensity", 2, "Mining intensity (0=stop, 1=low, 2=medium, 3=high)")
		npuEnabled = flag.Bool("npu", true, "Enable NPU acceleration if available")
		duration   = flag.Duration("duration", 0, "Mining duration (0=unlimited)")
		showInfo   = flag.Bool("info", false, "Show system information and exit")
	)
	flag.Parse()

	// Show system information if requested
	if *showInfo {
		showSystemInfo()
		return
	}

	// Create mobile miner configuration
	cfg := &mobilex.Config{
		RandomXMemory:           256 * 1024 * 1024, // 256MB for light mode
		RandomXCacheSize:        256 * 1024 * 1024,
		NPUEnabled:              *npuEnabled,
		NPUInterval:             150, // Run NPU every 150 hashes
		NPUModelPath:            "",  // Use built-in model
		UseNEON:                 true,
		ThermalThrottleStart:    45.0, // Start throttling at 45°C
		ThermalThrottleStop:     50.0, // Stop mining at 50°C
		OptimalOperatingTemp:    40.0, // Optimal temp
		ThermalTolerancePercent: 5.0,  // 5% tolerance
		BigCores:                runtime.NumCPU() / 2,
		LittleCores:             runtime.NumCPU() / 2,
	}

	// Create miner
	miner, err := mobilex.NewMobileXMiner(cfg)
	if err != nil {
		log.Fatalf("Failed to create miner: %v", err)
	}
	defer miner.Close()

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		fmt.Println("\nStopping miner...")
		cancel()
	}()

	// Start mining
	fmt.Println("Starting MobileX mining demo...")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Intensity: %d\n", *intensity)
	fmt.Printf("  NPU Enabled: %v\n", *npuEnabled)
	fmt.Printf("  CPU Cores: %d (big: %d, little: %d)\n",
		runtime.NumCPU(), cfg.BigCores, cfg.LittleCores)
	fmt.Printf("  Memory Mode: Light (256MB cache)\n")
	fmt.Println()

	// Start the miner
	if err := miner.Start(ctx); err != nil {
		log.Fatalf("Failed to start miner: %v", err)
	}

	// Create a test block to mine
	testBlock := createTestBlock()

	// Mining ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Quit channel
	quit := make(chan struct{})

	// Start time
	startTime := time.Now()

	// Duration timer if specified
	var durationTimer <-chan time.Time
	if *duration > 0 {
		durationTimer = time.After(*duration)
	} else {
		// Create a channel that never sends
		durationTimer = make(chan time.Time)
	}

	// Mining loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(quit)
				return
			case <-durationTimer:
				fmt.Printf("\nMining duration reached (%v)\n", *duration)
				close(quit)
				return
			default:
				// Try to solve the block
				found, err := miner.SolveBlock(testBlock, 1000, ticker, quit)
				if err != nil {
					log.Printf("Mining error: %v", err)
					continue
				}
				if found {
					fmt.Printf("\n🎉 Block found! Nonce: %d, ThermalProof: %d\n",
						testBlock.Header.Nonce, testBlock.Header.ThermalProof)
					// In a real implementation, would submit the block
				}
			}
		}
	}()

	// Monitor mining progress
	progressTicker := time.NewTicker(5 * time.Second)
	defer progressTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-quit:
			return
		case <-progressTicker.C:
			hashRate := miner.GetHashRate()
			elapsed := time.Since(startTime)
			fmt.Printf("Hash rate: %.2f H/s | Duration: %s\n",
				hashRate, elapsed.Round(time.Second))
		}
	}
}

// createTestBlock creates a test block for mining.
func createTestBlock() *wire.MsgBlock {
	// Create a test block header
	prevHash, _ := chainhash.NewHashFromStr("0000000000000000000000000000000000000000000000000000000000000000")
	merkleRoot, _ := chainhash.NewHashFromStr("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b")

	block := &wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:      1,
			PrevBlock:    *prevHash,
			MerkleRoot:   *merkleRoot,
			Timestamp:    time.Now(),
			Bits:         0x1d00ffff, // Easy difficulty for demo
			Nonce:        0,
			ThermalProof: 0,
		},
		Transactions: []*wire.MsgTx{},
	}

	return block
}

// showSystemInfo displays system information relevant to mobile mining.
func showSystemInfo() {
	fmt.Println("MobileX Mining System Information")
	fmt.Println("=================================")
	fmt.Printf("OS: %s\n", runtime.GOOS)
	fmt.Printf("Architecture: %s\n", runtime.GOARCH)
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("Go Version: %s\n", runtime.Version())

	// Check ARM64 features
	if runtime.GOARCH == "arm64" {
		opt := mobilex.NewARM64Optimizer()
		fmt.Println("\nARM64 Features:")
		fmt.Printf("  NEON: %v\n", opt.HasNEON())
		fmt.Printf("  Working Set Size: %d KB\n", opt.GetOptimalWorkingSetSize()/1024)
		fmt.Printf("  SoC Type: %s\n", opt.DetectSoCType())
	}

	// Check NPU availability
	fmt.Println("\nNPU Support:")
	caps := mobilex.GetNPUCapabilities()
	if caps.Available {
		fmt.Printf("  Vendor: %s\n", caps.Vendor)
		fmt.Printf("  Model: %s\n", caps.Model)
		fmt.Printf("  Compute Units: %d\n", caps.ComputeUnits)
		fmt.Printf("  Estimated Performance: %.1f TOPS\n", caps.EstimatedTOPS)
		fmt.Printf("  Power Efficiency: %s\n", caps.PowerEfficiency)
	} else {
		fmt.Println("  No NPU detected (will use CPU fallback)")
	}
}
