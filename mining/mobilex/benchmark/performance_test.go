package benchmark

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/toole-brendan/shell/mining/mobilex"
	"github.com/toole-brendan/shell/mining/mobilex/npu"
	"github.com/toole-brendan/shell/mining/mobilex/npu/fallback"
	"github.com/toole-brendan/shell/wire"
)

// TestDevice represents a test device configuration
type TestDevice struct {
	Name     string
	SoC      string
	NPU      string
	Cores    int
	BigCores int
}

// BenchmarkMobileXPerformance benchmarks MobileX on different device configurations
func BenchmarkMobileXPerformance(b *testing.B) {
	devices := []TestDevice{
		{Name: "iPhone 15 Pro", SoC: "A17 Pro", NPU: "CoreML", Cores: 6, BigCores: 2},
		{Name: "Galaxy S24", SoC: "Snapdragon 8 Gen 3", NPU: "NNAPI", Cores: 8, BigCores: 3},
		{Name: "Pixel 8", SoC: "Tensor G3", NPU: "NNAPI", Cores: 9, BigCores: 1},
		{Name: "Budget Phone", SoC: "Generic ARM", NPU: "None", Cores: 8, BigCores: 2},
	}

	for _, device := range devices {
		b.Run(device.Name, func(b *testing.B) {
			benchmarkDevice(b, device)
		})
	}
}

func benchmarkDevice(b *testing.B, device TestDevice) {
	// Create device-specific configuration
	cfg := &mobilex.Config{
		RandomXMemory:        256 * 1024 * 1024, // Light mode for testing
		BigCores:             device.BigCores,
		LittleCores:          device.Cores - device.BigCores,
		NPUEnabled:           device.NPU != "None",
		ThermalProofRequired: true,
	}

	// Create miner
	miner, err := mobilex.NewMobileXMiner(cfg)
	if err != nil {
		b.Fatalf("Failed to create miner: %v", err)
	}
	defer miner.Close()

	// Start mining
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = miner.Start(ctx)
	if err != nil {
		b.Fatalf("Failed to start miner: %v", err)
	}
	defer miner.Stop()

	// Reset timer after setup
	b.ResetTimer()

	// Measure hash rate
	initialHashRate := miner.GetHashRate()

	// Run for benchmark duration
	time.Sleep(time.Second * time.Duration(b.N))

	endHashRate := miner.GetHashRate()

	// Calculate average hash rate
	avgHashRate := (initialHashRate + endHashRate) / 2

	b.ReportMetric(avgHashRate, "H/s")
	b.ReportMetric(float64(device.Cores), "cores")
	b.ReportMetric(float64(device.BigCores), "big_cores")
}

// BenchmarkNPUvsGPU compares NPU performance against CPU fallback
func BenchmarkNPUvsCPU(b *testing.B) {
	// Create test tensor
	inputData := make([]float32, 32*32*3)
	for i := range inputData {
		inputData[i] = float32(i%256) / 255.0
	}
	inputTensor := npu.CreateTensor(inputData, []int{32, 32, 3})

	b.Run("CPU_Fallback", func(b *testing.B) {
		cpuFallback := fallback.NewCPUNeuralFallback()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := cpuFallback.RunConvolution(inputTensor)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Simulated NPU performance (would use real NPU in production)
	b.Run("Simulated_NPU", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate NPU being 2-3x faster
			time.Sleep(time.Microsecond * 200)
		}
	})
}

// BenchmarkThermalCompliance benchmarks thermal proof generation overhead
func BenchmarkThermalCompliance(b *testing.B) {
	tv := mobilex.NewThermalVerification(2000, 5.0)

	// Create test header
	header := &wire.BlockHeader{
		Version:   1,
		Timestamp: time.Now(),
		Bits:      0x1d00ffff,
		Nonce:     12345,
	}

	headerBytes := make([]byte, 80)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		thermalProof := tv.GenerateThermalProof(headerBytes)
		header.ThermalProof = thermalProof

		// Validate the proof
		err := tv.ValidateThermalProof(header)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryAccess benchmarks ARM-optimized memory patterns
func BenchmarkMemoryAccess(b *testing.B) {
	patterns := []struct {
		name    string
		size    int
		pattern string
	}{
		{"L1_Cache", 32 * 1024, "sequential"},
		{"L2_Cache", 256 * 1024, "sequential"},
		{"L3_Cache", 2 * 1024 * 1024, "sequential"},
		{"Random_L2", 256 * 1024, "random"},
		{"Random_L3", 2 * 1024 * 1024, "random"},
	}

	for _, p := range patterns {
		b.Run(p.name, func(b *testing.B) {
			benchmarkMemoryPattern(b, p.size, p.pattern)
		})
	}
}

func benchmarkMemoryPattern(b *testing.B, size int, pattern string) {
	// Allocate memory
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Create access indices
	indices := make([]int, 1000)
	if pattern == "sequential" {
		for i := range indices {
			indices[i] = (i * 64) % size // Cache line stride
		}
	} else {
		// Random pattern
		for i := range indices {
			indices[i] = (i * 1009) % size // Prime number for pseudo-random
		}
	}

	b.ResetTimer()
	b.SetBytes(int64(len(indices) * 8)) // 8 bytes per access

	for i := 0; i < b.N; i++ {
		sum := byte(0)
		for _, idx := range indices {
			sum += data[idx]
		}
		// Prevent optimization
		if sum == 0 {
			b.Fatal("Unexpected sum")
		}
	}
}

// BenchmarkHeterogeneousScheduling benchmarks big.LITTLE core coordination
func BenchmarkHeterogeneousScheduling(b *testing.B) {
	configs := []struct {
		name        string
		bigCores    int
		littleCores int
	}{
		{"Balanced_4+4", 4, 4},
		{"Performance_6+2", 6, 2},
		{"Efficiency_2+6", 2, 6},
		{"HighEnd_3+5", 3, 5},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			scheduler := mobilex.NewHeterogeneousScheduler(cfg.bigCores, cfg.littleCores)
			scheduler.Start()
			defer scheduler.Stop()

			testData := make([]byte, 1024)
			for i := range testData {
				testData[i] = byte(i % 256)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				scheduler.DistributeMining(testData)
			}
		})
	}
}

// BenchmarkFullMiningLoop benchmarks the complete mining loop
func BenchmarkFullMiningLoop(b *testing.B) {
	// Create configuration for benchmarking
	cfg := &mobilex.Config{
		RandomXMemory:        256 * 1024 * 1024, // Light mode
		BigCores:             2,
		LittleCores:          runtime.NumCPU() - 2,
		NPUEnabled:           false, // CPU only for consistent benchmarking
		ThermalProofRequired: true,
	}

	miner, err := mobilex.NewMobileXMiner(cfg)
	if err != nil {
		b.Fatalf("Failed to create miner: %v", err)
	}
	defer miner.Close()

	// Create a test block
	msgBlock := &wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:   1,
			Timestamp: time.Now(),
			Bits:      0x207fffff, // Easy difficulty for benchmarking
			Nonce:     0,
		},
	}

	b.ResetTimer()

	hashesPerSecond := make([]float64, b.N)

	for i := 0; i < b.N; i++ {
		startTime := time.Now()
		startNonce := msgBlock.Header.Nonce

		// Mine for a fixed duration
		ticker := time.NewTicker(100 * time.Millisecond)
		quit := make(chan struct{})

		go func() {
			time.Sleep(time.Second)
			close(quit)
		}()

		found, _ := miner.SolveBlock(msgBlock, 1, ticker, quit)
		ticker.Stop()

		if found {
			duration := time.Since(startTime)
			noncesDone := msgBlock.Header.Nonce - startNonce
			hashesPerSecond[i] = float64(noncesDone) / duration.Seconds()
		}
	}

	// Calculate average hash rate
	var totalHashRate float64
	validSamples := 0
	for _, rate := range hashesPerSecond {
		if rate > 0 {
			totalHashRate += rate
			validSamples++
		}
	}

	if validSamples > 0 {
		avgHashRate := totalHashRate / float64(validSamples)
		b.ReportMetric(avgHashRate, "H/s")
	}
}

// BenchmarkPowerEfficiency estimates power efficiency
func BenchmarkPowerEfficiency(b *testing.B) {
	powerProfiles := []struct {
		name       string
		intensity  string
		cores      int
		powerWatts float64
	}{
		{"Light_2W", "light", 2, 2.0},
		{"Medium_5W", "medium", 4, 5.0},
		{"Full_8W", "full", 8, 8.0},
	}

	for _, profile := range powerProfiles {
		b.Run(profile.name, func(b *testing.B) {
			cfg := &mobilex.Config{
				RandomXMemory:        256 * 1024 * 1024,
				BigCores:             profile.cores / 4,
				LittleCores:          profile.cores - profile.cores/4,
				NPUEnabled:           false,
				ThermalProofRequired: true,
			}

			miner, err := mobilex.NewMobileXMiner(cfg)
			if err != nil {
				b.Fatalf("Failed to create miner: %v", err)
			}
			defer miner.Close()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err = miner.Start(ctx)
			if err != nil {
				b.Fatalf("Failed to start miner: %v", err)
			}
			defer miner.Stop()

			// Mine for a duration
			time.Sleep(time.Second * 5)

			hashRate := miner.GetHashRate()
			efficiency := hashRate / profile.powerWatts

			b.ReportMetric(hashRate, "H/s")
			b.ReportMetric(profile.powerWatts, "watts")
			b.ReportMetric(efficiency, "H/s/W")
		})
	}
}

// GeneratePerformanceReport generates a comprehensive performance report
func GeneratePerformanceReport() {
	fmt.Println("MobileX Performance Report")
	fmt.Println("=========================")
	fmt.Println()

	// Device capabilities
	fmt.Printf("CPU Architecture: %s\n", runtime.GOARCH)
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Println()

	// Run benchmarks and collect results
	results := testing.Benchmark(BenchmarkFullMiningLoop)
	fmt.Printf("Full Mining Loop: %.2f H/s\n", results.Extra["H/s"])

	// Memory performance
	fmt.Println("\nMemory Performance:")
	fmt.Println("Cache Level | Access Time (ns) | Bandwidth (GB/s)")
	fmt.Println("------------|------------------|----------------")
	// Would run actual memory benchmarks here

	// Thermal characteristics
	fmt.Println("\nThermal Characteristics:")
	fmt.Println("Intensity | Temperature | Hash Rate | Efficiency")
	fmt.Println("----------|-------------|-----------|------------")
	// Would measure actual thermal performance here

	fmt.Println("\nNote: Run on actual mobile hardware for accurate results")
}
