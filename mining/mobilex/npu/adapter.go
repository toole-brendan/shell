// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package npu

import (
	"errors"
	"time"
)

// Common NPU errors.
var (
	ErrNPUNotAvailable    = errors.New("NPU not available on this device")
	ErrNPUInitFailed      = errors.New("NPU initialization failed")
	ErrNPUExecutionFailed = errors.New("NPU execution failed")
	ErrNPUModelInvalid    = errors.New("NPU model is invalid")
	ErrNPUTimeout         = errors.New("NPU operation timed out")
)

// NPUAdapter is the interface for platform-specific NPU implementations.
type NPUAdapter interface {
	// IsAvailable checks if NPU hardware is available and accessible.
	IsAvailable() bool

	// Initialize prepares the NPU for use with the given model.
	Initialize(modelPath string) error

	// RunConvolution executes a convolution operation on the NPU.
	RunConvolution(input Tensor) (Tensor, error)

	// GetPerformanceMetrics returns current NPU performance metrics.
	GetPerformanceMetrics() NPUMetrics

	// GetHardwareInfo returns information about the NPU hardware.
	GetHardwareInfo() HardwareInfo

	// Shutdown releases NPU resources.
	Shutdown() error
}

// Tensor represents a multi-dimensional array for NPU operations.
type Tensor struct {
	Data     []float32 // Flattened tensor data
	Shape    []int     // Dimensions (e.g., [32, 32, 3] for 32x32x3)
	DataType DataType  // Data type of tensor elements
	Layout   Layout    // Memory layout (NHWC, NCHW, etc.)
}

// DataType represents the data type of tensor elements.
type DataType int

const (
	Float32 DataType = iota
	Float16
	Int8
	UInt8
)

// Layout represents the memory layout of the tensor.
type Layout int

const (
	NHWC Layout = iota // Batch, Height, Width, Channel (TensorFlow default)
	NCHW               // Batch, Channel, Height, Width (PyTorch default)
	HWC                // Height, Width, Channel (no batch dimension)
	CHW                // Channel, Height, Width (no batch dimension)
)

// NPUMetrics contains performance metrics from NPU operations.
type NPUMetrics struct {
	InferenceTime      time.Duration // Time for last inference
	PowerUsage         float64       // Watts consumed
	Utilization        float64       // NPU utilization percentage (0-100)
	MemoryUsed         int64         // Bytes of NPU memory used
	Temperature        float64       // NPU temperature in Celsius
	InferencesPerSec   float64       // Throughput
	EnergyPerInference float64       // Joules per inference
}

// HardwareInfo contains information about the NPU hardware.
type HardwareInfo struct {
	Vendor          string   // NPU vendor (Qualcomm, Apple, MediaTek, etc.)
	Model           string   // NPU model name
	ComputeUnits    int      // Number of compute units
	MaxFrequency    int      // Maximum frequency in MHz
	MemoryBandwidth float64  // GB/s
	SupportedOps    []string // List of supported operations
	Precision       []string // Supported precisions (fp32, fp16, int8)
}

// ModelConfig contains configuration for the neural network model.
type ModelConfig struct {
	ModelPath   string   // Path to model file
	InputShape  []int    // Expected input shape
	OutputShape []int    // Expected output shape
	Precision   DataType // Computation precision
	BatchSize   int      // Batch size for inference
	CacheModel  bool     // Whether to cache compiled model
	Priority    Priority // Execution priority
	TimeoutMs   int      // Timeout in milliseconds
}

// Priority represents execution priority for NPU operations.
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityUrgent
)

// NPUManager manages NPU operations across different platforms.
type NPUManager struct {
	adapter       NPUAdapter
	config        *ModelConfig
	metrics       []NPUMetrics
	metricsBuffer int
	fallbackFunc  func(Tensor) (Tensor, error)
}

// NewNPUManager creates a new NPU manager with the given adapter.
func NewNPUManager(adapter NPUAdapter, config *ModelConfig) *NPUManager {
	return &NPUManager{
		adapter:       adapter,
		config:        config,
		metricsBuffer: 100,
		metrics:       make([]NPUMetrics, 0, 100),
	}
}

// SetFallback sets a CPU fallback function for when NPU is unavailable.
func (m *NPUManager) SetFallback(fallback func(Tensor) (Tensor, error)) {
	m.fallbackFunc = fallback
}

// ExecuteConvolution runs a convolution operation, falling back to CPU if needed.
func (m *NPUManager) ExecuteConvolution(input Tensor) (Tensor, error) {
	// Check if NPU is available
	if !m.adapter.IsAvailable() {
		if m.fallbackFunc != nil {
			return m.fallbackFunc(input)
		}
		return Tensor{}, ErrNPUNotAvailable
	}

	// Record start time
	startTime := time.Now()

	// Execute on NPU
	output, err := m.adapter.RunConvolution(input)
	if err != nil {
		// Try fallback on error
		if m.fallbackFunc != nil {
			return m.fallbackFunc(input)
		}
		return Tensor{}, err
	}

	// Record metrics
	metrics := m.adapter.GetPerformanceMetrics()
	metrics.InferenceTime = time.Since(startTime)
	m.recordMetrics(metrics)

	return output, nil
}

// recordMetrics stores performance metrics for analysis.
func (m *NPUManager) recordMetrics(metrics NPUMetrics) {
	m.metrics = append(m.metrics, metrics)
	if len(m.metrics) > m.metricsBuffer {
		m.metrics = m.metrics[1:]
	}
}

// GetAverageMetrics returns average performance metrics.
func (m *NPUManager) GetAverageMetrics() NPUMetrics {
	if len(m.metrics) == 0 {
		return NPUMetrics{}
	}

	var avg NPUMetrics
	for _, metric := range m.metrics {
		avg.InferenceTime += metric.InferenceTime
		avg.PowerUsage += metric.PowerUsage
		avg.Utilization += metric.Utilization
		avg.MemoryUsed += metric.MemoryUsed
		avg.Temperature += metric.Temperature
		avg.InferencesPerSec += metric.InferencesPerSec
		avg.EnergyPerInference += metric.EnergyPerInference
	}

	count := float64(len(m.metrics))
	avg.InferenceTime /= time.Duration(count)
	avg.PowerUsage /= count
	avg.Utilization /= count
	avg.MemoryUsed /= int64(count)
	avg.Temperature /= count
	avg.InferencesPerSec /= count
	avg.EnergyPerInference /= count

	return avg
}

// CreateTensor creates a new tensor with the given parameters.
func CreateTensor(data []float32, shape []int) Tensor {
	return Tensor{
		Data:     data,
		Shape:    shape,
		DataType: Float32,
		Layout:   NHWC,
	}
}

// Reshape reshapes a tensor to new dimensions.
func (t *Tensor) Reshape(newShape []int) error {
	// Verify total elements match
	oldSize := 1
	for _, dim := range t.Shape {
		oldSize *= dim
	}

	newSize := 1
	for _, dim := range newShape {
		newSize *= dim
	}

	if oldSize != newSize {
		return errors.New("reshape: total elements must remain the same")
	}

	t.Shape = newShape
	return nil
}

// GetElement returns the element at the given indices.
func (t *Tensor) GetElement(indices ...int) (float32, error) {
	if len(indices) != len(t.Shape) {
		return 0, errors.New("indices dimension mismatch")
	}

	// Calculate flat index
	flatIndex := 0
	stride := 1
	for i := len(indices) - 1; i >= 0; i-- {
		if indices[i] >= t.Shape[i] || indices[i] < 0 {
			return 0, errors.New("index out of bounds")
		}
		flatIndex += indices[i] * stride
		stride *= t.Shape[i]
	}

	return t.Data[flatIndex], nil
}

// SetElement sets the element at the given indices.
func (t *Tensor) SetElement(value float32, indices ...int) error {
	if len(indices) != len(t.Shape) {
		return errors.New("indices dimension mismatch")
	}

	// Calculate flat index
	flatIndex := 0
	stride := 1
	for i := len(indices) - 1; i >= 0; i-- {
		if indices[i] >= t.Shape[i] || indices[i] < 0 {
			return errors.New("index out of bounds")
		}
		flatIndex += indices[i] * stride
		stride *= t.Shape[i]
	}

	t.Data[flatIndex] = value
	return nil
}
