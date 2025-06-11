// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mobilex

import (
	"time"
)

// Config holds the configuration parameters for mobile mining.
type Config struct {
	// RandomX base configuration
	RandomXMemory     uint64 // Memory requirement (2GB for fast mode, 256MB for light)
	RandomXCacheSize  uint64 // Cache size for light mode
	RandomXIterations int    // Number of iterations per hash

	// Mobile-specific parameters
	ThermalProofRequired    bool    // Whether thermal proof is required
	ThermalTolerancePercent float64 // Thermal variance tolerance (default 5%)
	MaxOperatingTemp        float64 // Maximum operating temperature in Celsius
	OptimalOperatingTemp    float64 // Optimal operating temperature in Celsius

	// NPU configuration
	NPUEnabled         bool    // Whether to use NPU acceleration
	NPUInterval        int     // Run NPU operations every N iterations (100-200)
	NPUModelPath       string  // Path to neural network model weights
	NPUFallbackPenalty float64 // Performance penalty for CPU fallback (50-60%)

	// ARM64 optimizations
	UseNEON        bool // Use NEON vector instructions
	UseSVE         bool // Use Scalable Vector Extension (if available)
	UseDotProduct  bool // Use SDOT/UDOT instructions
	CacheLineSize  int  // ARM cache line size (typically 64 bytes)
	WorkingSetSize int  // Working set size in MB (1-3 MB for L2/L3 cache)

	// Heterogeneous core settings
	BigCores     int   // Number of performance cores to use
	LittleCores  int   // Number of efficiency cores to use
	SyncInterval int   // Synchronization interval between cores
	CoreAffinity []int // CPU core affinity mask

	// Power management
	MinBatteryLevel      int     // Minimum battery level to start mining (%)
	RequireCharging      bool    // Only mine when device is charging
	ThermalThrottleStart float64 // Temperature to start throttling
	ThermalThrottleStop  float64 // Temperature to stop mining

	// Mining intensity levels
	IntensityLight  MiningIntensity
	IntensityMedium MiningIntensity
	IntensityFull   MiningIntensity
}

// MiningIntensity represents different mining intensity levels.
type MiningIntensity struct {
	Name           string
	CoreCount      int     // Number of cores to use
	MaxHashRate    float64 // Maximum hash rate (H/s)
	PowerLimit     float64 // Power limit in watts
	ThermalLimit   float64 // Thermal limit in Celsius
	NPUUtilization float64 // NPU utilization percentage
}

// DefaultConfig returns the default mobile mining configuration.
func DefaultConfig() *Config {
	return &Config{
		// RandomX base settings
		RandomXMemory:     2 * 1024 * 1024 * 1024, // 2GB
		RandomXCacheSize:  256 * 1024 * 1024,      // 256MB
		RandomXIterations: 2048,

		// Thermal settings
		ThermalProofRequired:    true,
		ThermalTolerancePercent: 5.0,
		MaxOperatingTemp:        50.0, // 50°C max
		OptimalOperatingTemp:    40.0, // 40°C optimal

		// NPU settings
		NPUEnabled:         true,
		NPUInterval:        150,  // Every 150 iterations
		NPUFallbackPenalty: 0.55, // 55% penalty without NPU

		// ARM64 optimizations
		UseNEON:        true,
		UseSVE:         false, // Auto-detect at runtime
		UseDotProduct:  true,
		CacheLineSize:  64,
		WorkingSetSize: 2, // 2MB working set

		// Heterogeneous cores
		BigCores:     4,
		LittleCores:  4,
		SyncInterval: 75,

		// Power management
		MinBatteryLevel:      80,
		RequireCharging:      true,
		ThermalThrottleStart: 45.0,
		ThermalThrottleStop:  48.0,

		// Mining intensity presets
		IntensityLight: MiningIntensity{
			Name:           "light",
			CoreCount:      2,
			MaxHashRate:    30.0,
			PowerLimit:     2.0,
			ThermalLimit:   42.0,
			NPUUtilization: 0.25,
		},
		IntensityMedium: MiningIntensity{
			Name:           "medium",
			CoreCount:      4,
			MaxHashRate:    60.0,
			PowerLimit:     4.0,
			ThermalLimit:   45.0,
			NPUUtilization: 0.50,
		},
		IntensityFull: MiningIntensity{
			Name:           "full",
			CoreCount:      8,
			MaxHashRate:    120.0,
			PowerLimit:     8.0,
			ThermalLimit:   48.0,
			NPUUtilization: 1.0,
		},
	}
}

// LightModeConfig returns configuration optimized for older/budget phones.
func LightModeConfig() *Config {
	cfg := DefaultConfig()
	cfg.RandomXMemory = 256 * 1024 * 1024 // 256MB light mode
	cfg.WorkingSetSize = 1                // 1MB working set
	cfg.BigCores = 2
	cfg.LittleCores = 2
	return cfg
}

// DeviceProfile represents a specific device's mining capabilities.
type DeviceProfile struct {
	Name         string
	SoC          string // System on Chip name
	MaxHashRate  float64
	CoreCount    int
	HasNPU       bool
	NPUType      string // "coreml", "nnapi", "snpe", etc.
	ThermalClass string // "flagship", "midrange", "budget"
}

// GetDeviceProfile returns optimized settings for known device types.
func GetDeviceProfile(deviceName string) *DeviceProfile {
	profiles := map[string]*DeviceProfile{
		"iPhone 15 Pro": {
			Name:         "iPhone 15 Pro",
			SoC:          "A17 Pro",
			MaxHashRate:  150.0,
			CoreCount:    6,
			HasNPU:       true,
			NPUType:      "coreml",
			ThermalClass: "flagship",
		},
		"Galaxy S24": {
			Name:         "Galaxy S24",
			SoC:          "Snapdragon 8 Gen 3",
			MaxHashRate:  120.0,
			CoreCount:    8,
			HasNPU:       true,
			NPUType:      "nnapi",
			ThermalClass: "flagship",
		},
		"Pixel 8": {
			Name:         "Pixel 8",
			SoC:          "Tensor G3",
			MaxHashRate:  100.0,
			CoreCount:    8,
			HasNPU:       true,
			NPUType:      "nnapi",
			ThermalClass: "flagship",
		},
	}

	if profile, ok := profiles[deviceName]; ok {
		return profile
	}

	// Default profile for unknown devices
	return &DeviceProfile{
		Name:         "Unknown",
		SoC:          "Unknown",
		MaxHashRate:  50.0,
		CoreCount:    4,
		HasNPU:       false,
		ThermalClass: "budget",
	}
}

// OptimizeForDevice adjusts configuration based on device profile.
func (c *Config) OptimizeForDevice(profile *DeviceProfile) {
	switch profile.ThermalClass {
	case "flagship":
		c.MaxOperatingTemp = 45.0
		c.OptimalOperatingTemp = 38.0
		c.BigCores = min(profile.CoreCount/2, 4)
		c.LittleCores = min(profile.CoreCount/2, 4)

	case "midrange":
		c.MaxOperatingTemp = 43.0
		c.OptimalOperatingTemp = 40.0
		c.BigCores = min(profile.CoreCount/2, 2)
		c.LittleCores = min(profile.CoreCount/2, 4)
		c.RandomXMemory = 1 * 1024 * 1024 * 1024 // 1GB

	case "budget":
		c.MaxOperatingTemp = 42.0
		c.OptimalOperatingTemp = 40.0
		c.BigCores = 1
		c.LittleCores = min(profile.CoreCount-1, 3)
		c.RandomXMemory = 256 * 1024 * 1024 // 256MB light mode
		c.WorkingSetSize = 1
	}

	c.NPUEnabled = profile.HasNPU
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ValidationParams holds parameters for thermal proof validation.
type ValidationParams struct {
	RandomValidationRate    float64       // Percentage of blocks to randomly validate
	ValidationClockSpeed    float64       // Clock speed multiplier for validation (0.5 = 50%)
	StatisticalWindowSize   int           // Number of blocks for statistical analysis
	ThermalOutlierThreshold float64       // Z-score threshold for thermal outliers
	ValidationTimeout       time.Duration // Maximum time for validation
}

// DefaultValidationParams returns default thermal validation parameters.
func DefaultValidationParams() *ValidationParams {
	return &ValidationParams{
		RandomValidationRate:    0.10, // Validate 10% of blocks
		ValidationClockSpeed:    0.50, // Run at 50% clock speed
		StatisticalWindowSize:   1000, // Analyze last 1000 blocks
		ThermalOutlierThreshold: 3.0,  // 3 standard deviations
		ValidationTimeout:       30 * time.Second,
	}
}
