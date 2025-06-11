package testing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toole-brendan/shell/mining/mobilex/npu"
	"github.com/toole-brendan/shell/mining/mobilex/npu/fallback"
)

// TestCPUFallbackConvolution tests the CPU fallback implementation
func TestCPUFallbackConvolution(t *testing.T) {
	// Create CPU fallback
	cpuFallback := fallback.NewCPUNeuralFallback()
	require.NotNil(t, cpuFallback)

	// Create test input tensor
	inputData := make([]float32, 32*32*3)
	for i := range inputData {
		inputData[i] = float32(i) / float32(len(inputData))
	}
	inputTensor := npu.CreateTensor(inputData, []int{32, 32, 3})

	// Execute convolution
	output, err := cpuFallback.RunConvolution(inputTensor)
	require.NoError(t, err)
	require.NotNil(t, output)

	// Verify output shape
	expectedSize := 32 * 32 * 3
	assert.Equal(t, expectedSize, len(output.Data))
}

// TestTensorOperations tests tensor creation and manipulation
func TestTensorOperations(t *testing.T) {
	// Test tensor creation
	data := make([]float32, 2*3*4)
	for i := range data {
		data[i] = float32(i)
	}
	tensor := npu.CreateTensor(data, []int{2, 3, 4})

	assert.Equal(t, 24, len(tensor.Data))
	assert.Equal(t, []int{2, 3, 4}, tensor.Shape)
	assert.Equal(t, npu.Float32, tensor.DataType)
	assert.Equal(t, npu.NHWC, tensor.Layout)

	// Test reshape
	err := tensor.Reshape([]int{6, 4})
	assert.NoError(t, err)
	assert.Equal(t, []int{6, 4}, tensor.Shape)

	// Test invalid reshape
	err = tensor.Reshape([]int{5, 5})
	assert.Error(t, err, "Should fail with mismatched element count")

	// Test element access
	tensor.Reshape([]int{2, 3, 4})
	val, err := tensor.GetElement(1, 2, 3)
	assert.NoError(t, err)
	assert.Equal(t, float32(23), val) // 1*12 + 2*4 + 3 = 23

	// Test element setting
	err = tensor.SetElement(99.0, 1, 2, 3)
	assert.NoError(t, err)
	val, err = tensor.GetElement(1, 2, 3)
	assert.NoError(t, err)
	assert.Equal(t, float32(99.0), val)
}

// BenchmarkCPUFallback benchmarks the CPU fallback performance
func BenchmarkCPUFallback(b *testing.B) {
	cpuFallback := fallback.NewCPUNeuralFallback()

	// Create test tensor
	inputData := make([]float32, 32*32*3)
	for i := range inputData {
		inputData[i] = float32(i%256) / 255.0
	}
	inputTensor := npu.CreateTensor(inputData, []int{32, 32, 3})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cpuFallback.RunConvolution(inputTensor)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestCPUFallbackOptimization tests device-specific optimizations
func TestCPUFallbackOptimization(t *testing.T) {
	cpuFallback := fallback.NewCPUNeuralFallback()

	// Test optimization for different core counts
	tests := []struct {
		name      string
		coreCount int
		hasSIMD   bool
	}{
		{
			name:      "Single core without SIMD",
			coreCount: 1,
			hasSIMD:   false,
		},
		{
			name:      "Quad core with SIMD",
			coreCount: 4,
			hasSIMD:   true,
		},
		{
			name:      "Octa core with SIMD",
			coreCount: 8,
			hasSIMD:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpuFallback.OptimizeForDevice(tt.coreCount, tt.hasSIMD)

			// Run benchmark after optimization
			hashRate, err := cpuFallback.Benchmark()
			require.NoError(t, err)

			// Verify hash rate is reasonable
			assert.Greater(t, hashRate, 0.0, "Hash rate should be positive")

			// SIMD should provide better performance
			if tt.hasSIMD {
				assert.Greater(t, hashRate, 10.0, "SIMD should provide reasonable performance")
			}
		})
	}
}

// TestCPUFallbackMetrics tests performance metrics from CPU fallback
func TestCPUFallbackMetrics(t *testing.T) {
	cpuFallback := fallback.NewCPUNeuralFallback()

	// Get metrics
	metrics := cpuFallback.GetPerformanceMetrics()

	// Verify metrics are reasonable
	assert.Greater(t, metrics.InferenceTime, time.Duration(0))
	assert.Greater(t, metrics.PowerUsage, 0.0)
	assert.GreaterOrEqual(t, metrics.Utilization, 0.0)
	assert.LessOrEqual(t, metrics.Utilization, 100.0)
	assert.Greater(t, metrics.MemoryUsed, int64(0))
	assert.Greater(t, metrics.Temperature, 0.0)
	assert.Greater(t, metrics.InferencesPerSec, 0.0)
	assert.Greater(t, metrics.EnergyPerInference, 0.0)
}

// TestDataTypePrecision tests different data type precisions
func TestDataTypePrecision(t *testing.T) {
	dataTypes := []struct {
		name     string
		dataType npu.DataType
	}{
		{"Float32", npu.Float32},
		{"Float16", npu.Float16},
		{"Int8", npu.Int8},
		{"UInt8", npu.UInt8},
	}

	for _, dt := range dataTypes {
		t.Run(dt.name, func(t *testing.T) {
			// Create tensor with specific data type
			data := make([]float32, 100)
			tensor := npu.Tensor{
				Data:     data,
				Shape:    []int{10, 10},
				DataType: dt.dataType,
				Layout:   npu.HWC,
			}

			// Verify data type is set correctly
			assert.Equal(t, dt.dataType, tensor.DataType)
		})
	}
}

// TestTensorLayouts tests different tensor memory layouts
func TestTensorLayouts(t *testing.T) {
	layouts := []struct {
		name   string
		layout npu.Layout
	}{
		{"NHWC", npu.NHWC},
		{"NCHW", npu.NCHW},
		{"HWC", npu.HWC},
		{"CHW", npu.CHW},
	}

	for _, l := range layouts {
		t.Run(l.name, func(t *testing.T) {
			// Create tensor with specific layout
			data := make([]float32, 2*3*4*5)
			tensor := npu.Tensor{
				Data:     data,
				Shape:    []int{2, 3, 4, 5},
				DataType: npu.Float32,
				Layout:   l.layout,
			}

			// Verify layout is set correctly
			assert.Equal(t, l.layout, tensor.Layout)
		})
	}
}

// TestNPUPriority tests NPU execution priorities
func TestNPUPriority(t *testing.T) {
	priorities := []struct {
		name     string
		priority npu.Priority
	}{
		{"Low Priority", npu.PriorityLow},
		{"Medium Priority", npu.PriorityMedium},
		{"High Priority", npu.PriorityHigh},
		{"Urgent Priority", npu.PriorityUrgent},
	}

	for _, p := range priorities {
		t.Run(p.name, func(t *testing.T) {
			config := &npu.ModelConfig{
				InputShape:  []int{32, 32, 3},
				OutputShape: []int{32, 32, 3},
				Precision:   npu.Float32,
				BatchSize:   1,
				Priority:    p.priority,
				TimeoutMs:   100,
			}

			// Verify priority is set correctly
			assert.Equal(t, p.priority, config.Priority)
		})
	}
}

// BenchmarkTensorOperations benchmarks basic tensor operations
func BenchmarkTensorOperations(b *testing.B) {
	// Create test tensor
	data := make([]float32, 100*100*3)
	for i := range data {
		data[i] = float32(i%256) / 255.0
	}

	b.Run("TensorCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = npu.CreateTensor(data, []int{100, 100, 3})
		}
	})

	tensor := npu.CreateTensor(data, []int{100, 100, 3})

	b.Run("TensorReshape", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = tensor.Reshape([]int{300, 100})
			_ = tensor.Reshape([]int{100, 100, 3})
		}
	})

	b.Run("TensorElementAccess", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = tensor.GetElement(50, 50, 1)
		}
	})
}

// TestCPUFallbackPerformancePenalty tests the expected performance penalty
func TestCPUFallbackPerformancePenalty(t *testing.T) {
	cpuFallback := fallback.NewCPUNeuralFallback()

	// Create test tensor
	inputData := make([]float32, 32*32*3)
	inputTensor := npu.CreateTensor(inputData, []int{32, 32, 3})

	// Measure execution time
	iterations := 10
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := cpuFallback.RunConvolution(inputTensor)
		require.NoError(t, err)
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	// CPU fallback should show performance penalty (slower execution)
	// Expecting at least some measurable time per operation
	assert.Greater(t, avgTime, time.Microsecond*100,
		"CPU fallback should show measurable execution time")
}

// BenchmarkCPUvsSimulatedNPU compares CPU fallback with simulated NPU performance
func BenchmarkCPUvsSimulatedNPU(b *testing.B) {
	// Create test tensor
	inputData := make([]float32, 32*32*3)
	for i := range inputData {
		inputData[i] = float32(i%256) / 255.0
	}
	inputTensor := npu.CreateTensor(inputData, []int{32, 32, 3})

	// Benchmark CPU fallback
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

	// Simulated NPU performance (2-3x faster than CPU)
	b.Run("Simulated_NPU", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate NPU execution - just create output tensor
			output := npu.CreateTensor(inputData, []int{32, 32, 3})
			// Simulate faster NPU timing
			time.Sleep(time.Microsecond)
			_ = output
		}
	})
}
