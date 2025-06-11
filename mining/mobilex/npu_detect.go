// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mobilex

import (
	"runtime"
	"strings"

	"github.com/toole-brendan/shell/mining/mobilex/npu"
	"github.com/toole-brendan/shell/mining/mobilex/npu/adapters"
)

// DetectNPUAdapter detects and returns the appropriate NPU adapter for the current platform.
// This is exported for use in demos and testing.
func DetectNPUAdapter() npu.NPUAdapter {
	// Detect platform and return appropriate adapter
	switch runtime.GOOS {
	case "android":
		return detectAndroidNPU()
	case "darwin":
		return detectAppleNPU()
	case "ios":
		return detectAppleNPU()
	default:
		// No NPU support on other platforms
		return nil
	}
}

// detectAndroidNPU detects and initializes Android NPU support.
func detectAndroidNPU() npu.NPUAdapter {
	adapter := adapters.NewAndroidNNAPIAdapter()

	// Check if the adapter is actually available
	if adapter.IsAvailable() {
		// Try to initialize with default model
		if err := adapter.Initialize(""); err == nil {
			return adapter
		}
	}

	// NPU not available or initialization failed
	return nil
}

// detectAppleNPU detects and initializes Apple Neural Engine support.
func detectAppleNPU() npu.NPUAdapter {
	adapter := adapters.NewIOSCoreMLAdapter()

	// Check if Core ML is available
	if adapter.IsAvailable() {
		// Try to initialize with default model
		if err := adapter.Initialize(""); err == nil {
			return adapter
		}
	}

	// Neural Engine not available
	return nil
}

// GetNPUInfo returns information about available NPU hardware.
// Returns nil if no NPU is detected.
func GetNPUInfo() *npu.HardwareInfo {
	adapter := DetectNPUAdapter()
	if adapter == nil {
		return nil
	}

	info := adapter.GetHardwareInfo()

	// Clean up the adapter
	_ = adapter.Shutdown()

	return &info
}

// detectNPUAdapter is a package-private version for internal use.
// This maintains compatibility with existing code.
func detectNPUAdapter() npu.NPUAdapter {
	return DetectNPUAdapter()
}

// Platform detection helpers

// IsAndroid returns true if running on Android.
func IsAndroid() bool {
	return runtime.GOOS == "android"
}

// IsDarwin returns true if running on macOS or iOS.
func IsDarwin() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "ios"
}

// GetSoCInfo attempts to detect the System-on-Chip information.
func GetSoCInfo() string {
	// This is a simplified version. In production, would read from:
	// - /proc/cpuinfo on Android
	// - sysctlbyname on iOS/macOS

	switch runtime.GOOS {
	case "android":
		return detectAndroidSoC()
	case "darwin", "ios":
		return detectAppleSoC()
	default:
		return "Unknown"
	}
}

// detectAndroidSoC attempts to identify the Android SoC.
func detectAndroidSoC() string {
	// In real implementation, would parse /proc/cpuinfo
	// For now, return a generic identifier

	// Common patterns to look for:
	// - "Qualcomm Technologies, Inc SDM845" -> Snapdragon 845
	// - "Qualcomm Technologies, Inc SM8550" -> Snapdragon 8 Gen 2
	// - "MT6889Z/CZA" -> MediaTek Dimensity 1000

	return "Android ARM64 SoC"
}

// detectAppleSoC attempts to identify the Apple SoC.
func detectAppleSoC() string {
	// In real implementation, would use sysctlbyname
	// to get hw.targettype

	// Examples:
	// - "J413" -> M1
	// - "J314s" -> M1 Pro
	// - "J316s" -> M1 Max
	// - "J413s" -> M2
	// - "D83AP" -> A14 (iPhone 12)
	// - "D84AP" -> A15 (iPhone 13)

	return "Apple Silicon"
}

// NPUCapabilities describes the capabilities of detected NPU hardware.
type NPUCapabilities struct {
	Available       bool
	Vendor          string
	Model           string
	ComputeUnits    int
	SupportedOps    []string
	EstimatedTOPS   float64 // Trillions of operations per second
	PowerEfficiency string  // e.g., "High", "Medium", "Low"
}

// GetNPUCapabilities returns detailed NPU capabilities.
func GetNPUCapabilities() NPUCapabilities {
	caps := NPUCapabilities{
		Available: false,
	}

	adapter := DetectNPUAdapter()
	if adapter == nil {
		return caps
	}

	defer adapter.Shutdown()

	if !adapter.IsAvailable() {
		return caps
	}

	info := adapter.GetHardwareInfo()
	caps.Available = true
	caps.Vendor = info.Vendor
	caps.Model = info.Model
	caps.ComputeUnits = info.ComputeUnits
	caps.SupportedOps = info.SupportedOps

	// Estimate TOPS based on known hardware
	caps.EstimatedTOPS = estimateTOPS(info)
	caps.PowerEfficiency = estimatePowerEfficiency(info)

	return caps
}

// estimateTOPS estimates the TOPS based on hardware info.
func estimateTOPS(info npu.HardwareInfo) float64 {
	// Rough estimates based on known hardware
	if strings.Contains(info.Model, "Neural Engine") {
		// Apple Neural Engine estimates
		if info.ComputeUnits >= 16 {
			return 15.8 // A15/A16/A17 Neural Engine
		}
		return 11.0 // A14 Neural Engine
	}

	if strings.Contains(info.Vendor, "Qualcomm") {
		// Qualcomm Hexagon estimates
		return 5.0 // Conservative estimate
	}

	if strings.Contains(info.Vendor, "MediaTek") {
		// MediaTek APU estimates
		return 4.0 // Conservative estimate
	}

	// Unknown hardware
	return 1.0
}

// estimatePowerEfficiency estimates power efficiency rating.
func estimatePowerEfficiency(info npu.HardwareInfo) string {
	tops := estimateTOPS(info)

	if tops >= 10.0 {
		return "High"
	} else if tops >= 5.0 {
		return "Medium"
	}
	return "Low"
}
