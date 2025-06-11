// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

//go:build !android
// +build !android

package adapters

import (
	"errors"

	"github.com/toole-brendan/shell/mining/mobilex/npu"
)

// AndroidNNAPIAdapter stub for non-Android platforms
type AndroidNNAPIAdapter struct{}

// NewAndroidNNAPIAdapter creates a stub adapter on non-Android platforms
func NewAndroidNNAPIAdapter() npu.NPUAdapter {
	return &AndroidNNAPIAdapter{}
}

// IsAvailable checks if NPU is available
func (a *AndroidNNAPIAdapter) IsAvailable() bool {
	return false
}

// Initialize prepares the NPU
func (a *AndroidNNAPIAdapter) Initialize(modelPath string) error {
	return errors.New("Android NNAPI not available on this platform")
}

// RunConvolution executes convolution on the NPU
func (a *AndroidNNAPIAdapter) RunConvolution(input npu.Tensor) (npu.Tensor, error) {
	return npu.Tensor{}, errors.New("Android NNAPI not available on this platform")
}

// GetPerformanceMetrics returns NPU metrics
func (a *AndroidNNAPIAdapter) GetPerformanceMetrics() npu.NPUMetrics {
	return npu.NPUMetrics{}
}

// GetHardwareInfo returns NPU hardware information
func (a *AndroidNNAPIAdapter) GetHardwareInfo() npu.HardwareInfo {
	return npu.HardwareInfo{
		Vendor: "Android NNAPI (stub)",
		Model:  "Not Available",
	}
}

// Shutdown releases NPU resources
func (a *AndroidNNAPIAdapter) Shutdown() error {
	return nil
}
