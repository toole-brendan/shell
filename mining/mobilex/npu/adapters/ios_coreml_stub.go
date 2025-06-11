// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

//go:build !ios && !darwin
// +build !ios,!darwin

package adapters

import (
	"errors"

	"github.com/toole-brendan/shell/mining/mobilex/npu"
)

// IOSCoreMLAdapter stub for non-iOS/macOS platforms
type IOSCoreMLAdapter struct{}

// NewIOSCoreMLAdapter creates a stub adapter on non-iOS/macOS platforms
func NewIOSCoreMLAdapter() npu.NPUAdapter {
	return &IOSCoreMLAdapter{}
}

// IsAvailable checks if NPU is available
func (a *IOSCoreMLAdapter) IsAvailable() bool {
	return false
}

// Initialize prepares the NPU
func (a *IOSCoreMLAdapter) Initialize(modelPath string) error {
	return errors.New("Core ML not available on this platform")
}

// RunConvolution executes convolution on the NPU
func (a *IOSCoreMLAdapter) RunConvolution(input npu.Tensor) (npu.Tensor, error) {
	return npu.Tensor{}, errors.New("Core ML not available on this platform")
}

// GetPerformanceMetrics returns NPU metrics
func (a *IOSCoreMLAdapter) GetPerformanceMetrics() npu.NPUMetrics {
	return npu.NPUMetrics{}
}

// GetHardwareInfo returns NPU hardware information
func (a *IOSCoreMLAdapter) GetHardwareInfo() npu.HardwareInfo {
	return npu.HardwareInfo{
		Vendor: "Apple Core ML (stub)",
		Model:  "Not Available",
	}
}

// Shutdown releases NPU resources
func (a *IOSCoreMLAdapter) Shutdown() error {
	return nil
}
