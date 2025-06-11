// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fallback

import (
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/toole-brendan/shell/mining/mobilex/npu"
)

// CPUNeuralFallback provides CPU-based neural computation fallback.
type CPUNeuralFallback struct {
	numThreads         int
	performancePenalty float64 // Expected 50-60% penalty vs NPU
	convWeights        []float32
	convBias           []float32
	kernelSize         int
	stride             int
	padding            int
}

// NewCPUNeuralFallback creates a new CPU fallback implementation.
func NewCPUNeuralFallback() *CPUNeuralFallback {
	return &CPUNeuralFallback{
		numThreads:         runtime.NumCPU(),
		performancePenalty: 0.55, // 55% penalty
		kernelSize:         3,    // 3x3 kernel
		stride:             1,
		padding:            1,
		convWeights:        initializeDepthwiseWeights(),
		convBias:           initializeBias(),
	}
}

// RunConvolution performs depthwise separable convolution on CPU.
func (cf *CPUNeuralFallback) RunConvolution(input npu.Tensor) (npu.Tensor, error) {
	// Simulate performance penalty with artificial delay
	time.Sleep(time.Duration(cf.performancePenalty * float64(time.Millisecond)))

	// Validate input shape (expecting 32x32x3)
	if len(input.Shape) != 3 || input.Shape[0] != 32 || input.Shape[1] != 32 || input.Shape[2] != 3 {
		// Reshape if needed
		input.Reshape([]int{32, 32, 3})
	}

	// Perform depthwise convolution
	depthwiseOutput := cf.depthwiseConvolution(input)

	// Perform pointwise convolution (1x1)
	output := cf.pointwiseConvolution(depthwiseOutput)

	// Apply ReLU activation
	cf.applyReLU(&output)

	return output, nil
}

// depthwiseConvolution performs depthwise convolution (each channel independently).
func (cf *CPUNeuralFallback) depthwiseConvolution(input npu.Tensor) npu.Tensor {
	height, width, channels := input.Shape[0], input.Shape[1], input.Shape[2]

	// Calculate output dimensions
	outHeight := (height+2*cf.padding-cf.kernelSize)/cf.stride + 1
	outWidth := (width+2*cf.padding-cf.kernelSize)/cf.stride + 1

	// Create output tensor
	outputData := make([]float32, outHeight*outWidth*channels)
	output := npu.Tensor{
		Data:     outputData,
		Shape:    []int{outHeight, outWidth, channels},
		DataType: npu.Float32,
		Layout:   npu.HWC,
	}

	// Parallel processing across channels
	var wg sync.WaitGroup
	channelsPerThread := channels / cf.numThreads
	if channelsPerThread < 1 {
		channelsPerThread = 1
	}

	for c := 0; c < channels; c += channelsPerThread {
		wg.Add(1)
		go func(startC int) {
			defer wg.Done()
			endC := startC + channelsPerThread
			if endC > channels {
				endC = channels
			}

			for ch := startC; ch < endC; ch++ {
				cf.processChannel(input, &output, ch)
			}
		}(c)
	}

	wg.Wait()
	return output
}

// processChannel processes a single channel for depthwise convolution.
func (cf *CPUNeuralFallback) processChannel(input npu.Tensor, output *npu.Tensor, channel int) {
	height, width := input.Shape[0], input.Shape[1]
	outHeight, outWidth := output.Shape[0], output.Shape[1]

	// Get weights for this channel
	weightOffset := channel * cf.kernelSize * cf.kernelSize

	for y := 0; y < outHeight; y++ {
		for x := 0; x < outWidth; x++ {
			sum := float32(0.0)

			// Apply kernel
			for ky := 0; ky < cf.kernelSize; ky++ {
				for kx := 0; kx < cf.kernelSize; kx++ {
					// Calculate input coordinates
					inY := y*cf.stride - cf.padding + ky
					inX := x*cf.stride - cf.padding + kx

					// Check bounds
					if inY >= 0 && inY < height && inX >= 0 && inX < width {
						inIdx := (inY*width+inX)*input.Shape[2] + channel
						wIdx := weightOffset + ky*cf.kernelSize + kx
						sum += input.Data[inIdx] * cf.convWeights[wIdx]
					}
				}
			}

			// Add bias and store result
			outIdx := (y*outWidth+x)*output.Shape[2] + channel
			output.Data[outIdx] = sum + cf.convBias[channel]
		}
	}
}

// pointwiseConvolution performs 1x1 convolution to mix channels.
func (cf *CPUNeuralFallback) pointwiseConvolution(input npu.Tensor) npu.Tensor {
	height, width, channels := input.Shape[0], input.Shape[1], input.Shape[2]

	// For simplicity, keep same number of channels
	outputData := make([]float32, height*width*channels)
	output := npu.Tensor{
		Data:     outputData,
		Shape:    []int{height, width, channels},
		DataType: npu.Float32,
		Layout:   npu.HWC,
	}

	// Simple channel mixing
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			for c := 0; c < channels; c++ {
				sum := float32(0.0)

				// Mix all input channels
				for ic := 0; ic < channels; ic++ {
					inIdx := (y*width+x)*channels + ic
					// Simple mixing weights
					weight := float32(1.0)
					if ic == c {
						weight = 2.0
					}
					sum += input.Data[inIdx] * weight / float32(channels)
				}

				outIdx := (y*width+x)*channels + c
				output.Data[outIdx] = sum
			}
		}
	}

	return output
}

// applyReLU applies ReLU activation in-place.
func (cf *CPUNeuralFallback) applyReLU(tensor *npu.Tensor) {
	for i := range tensor.Data {
		if tensor.Data[i] < 0 {
			tensor.Data[i] = 0
		}
	}
}

// initializeDepthwiseWeights initializes random weights for depthwise convolution.
func initializeDepthwiseWeights() []float32 {
	// 3x3 kernel for each of 3 channels
	weights := make([]float32, 3*3*3)

	// Initialize with small random values
	for i := range weights {
		// Simple deterministic initialization for reproducibility
		weights[i] = float32(math.Sin(float64(i))) * 0.1
	}

	return weights
}

// initializeBias initializes bias values.
func initializeBias() []float32 {
	// One bias per channel
	bias := make([]float32, 3)
	for i := range bias {
		bias[i] = 0.01
	}
	return bias
}

// GetPerformanceMetrics returns CPU fallback performance metrics.
func (cf *CPUNeuralFallback) GetPerformanceMetrics() npu.NPUMetrics {
	return npu.NPUMetrics{
		InferenceTime:      time.Duration(cf.performancePenalty) * time.Millisecond,
		PowerUsage:         5.0, // Estimated 5W for CPU computation
		Utilization:        float64(cf.numThreads) / float64(runtime.NumCPU()) * 100,
		MemoryUsed:         32 * 32 * 3 * 4 * 2, // Input + output tensors
		Temperature:        45.0,                // Estimated temperature
		InferencesPerSec:   1000.0 / cf.performancePenalty,
		EnergyPerInference: 5.0 / (1000.0 / cf.performancePenalty),
	}
}

// OptimizeForDevice adjusts CPU fallback based on device capabilities.
func (cf *CPUNeuralFallback) OptimizeForDevice(coreCount int, hasSIMD bool) {
	cf.numThreads = coreCount

	// Adjust performance penalty based on CPU features
	if hasSIMD {
		cf.performancePenalty = 0.50 // Better performance with SIMD
	} else {
		cf.performancePenalty = 0.60 // Worse without SIMD
	}
}

// Benchmark runs a simple benchmark of the CPU fallback.
func (cf *CPUNeuralFallback) Benchmark() (float64, error) {
	// Create test input
	testData := make([]float32, 32*32*3)
	for i := range testData {
		testData[i] = float32(i%256) / 255.0
	}

	testInput := npu.Tensor{
		Data:     testData,
		Shape:    []int{32, 32, 3},
		DataType: npu.Float32,
		Layout:   npu.HWC,
	}

	// Run multiple iterations
	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := cf.RunConvolution(testInput)
		if err != nil {
			return 0, err
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed.Seconds() / float64(iterations)
	hashRate := 1.0 / avgTime

	return hashRate, nil
}
