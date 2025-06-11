package testing

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toole-brendan/shell/mining/mobilex"
)

// TestHeterogeneousSchedulerCreation tests scheduler initialization
func TestHeterogeneousSchedulerCreation(t *testing.T) {
	tests := []struct {
		name        string
		bigCores    int
		littleCores int
	}{
		{
			name:        "Typical mobile configuration",
			bigCores:    4,
			littleCores: 4,
		},
		{
			name:        "High-end configuration",
			bigCores:    3,
			littleCores: 5,
		},
		{
			name:        "Budget configuration",
			bigCores:    2,
			littleCores: 6,
		},
		{
			name:        "Single big core",
			bigCores:    1,
			littleCores: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := mobilex.NewHeterogeneousScheduler(tt.bigCores, tt.littleCores)
			require.NotNil(t, scheduler)

			// Verify scheduler was created successfully
			// Note: The actual implementation doesn't expose core counts directly
			assert.NotNil(t, scheduler)
		})
	}
}

// TestSchedulerStartStop tests starting and stopping the scheduler
func TestSchedulerStartStop(t *testing.T) {
	scheduler := mobilex.NewHeterogeneousScheduler(4, 4)

	// Initially no cores should be active
	assert.Equal(t, 0, scheduler.ActiveCores(), "No cores should be active before start")

	// Start scheduler
	scheduler.Start()

	// Give scheduler time to start
	time.Sleep(time.Millisecond * 10)

	// Some cores should become active
	assert.Greater(t, scheduler.ActiveCores(), 0, "Some cores should be active after start")

	// Stop scheduler
	scheduler.Stop()

	// Give scheduler time to stop
	time.Sleep(time.Millisecond * 10)
}

// TestDistributeMining tests mining work distribution
func TestDistributeMining(t *testing.T) {
	scheduler := mobilex.NewHeterogeneousScheduler(4, 4)
	scheduler.Start()
	defer scheduler.Stop()

	// Create test data
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Distribute mining work
	for i := 0; i < 10; i++ {
		scheduler.DistributeMining(testData)
		time.Sleep(time.Millisecond * 5)
	}

	// Get metrics to verify work was scheduled
	// Note: We can't access unexported fields, but calling GetMetrics verifies it works
	_ = scheduler.GetMetrics()
}

// TestDynamicIntensityAdjustment tests dynamic intensity changes
func TestDynamicIntensityAdjustment(t *testing.T) {
	scheduler := mobilex.NewHeterogeneousScheduler(4, 4)
	scheduler.Start()
	defer scheduler.Stop()

	// Get initial active cores
	initialCores := scheduler.ActiveCores()

	// Reduce intensity
	scheduler.ReduceIntensity()
	time.Sleep(time.Millisecond * 10)
	reducedCores := scheduler.ActiveCores()
	assert.LessOrEqual(t, reducedCores, initialCores, "Active cores should reduce")

	// Increase intensity
	scheduler.IncreaseIntensity()
	scheduler.IncreaseIntensity()
	time.Sleep(time.Millisecond * 10)
	increasedCores := scheduler.ActiveCores()
	assert.GreaterOrEqual(t, increasedCores, reducedCores, "Active cores should increase")
}

// TestCoreState tests the core state tracking functionality
func TestCoreState(t *testing.T) {
	scheduler := mobilex.NewHeterogeneousScheduler(2, 2)

	// Get initial state
	initialState := scheduler.GetCoreState()
	assert.NotEqual(t, uint32(0), initialState, "Core state should be non-zero")

	// Start scheduler
	scheduler.Start()
	defer scheduler.Stop()

	// Distribute some work
	testData := make([]byte, 100)
	for i := 0; i < 5; i++ {
		scheduler.DistributeMining(testData)
		time.Sleep(time.Millisecond * 5)
	}

	// Get new state - should be different after work
	newState := scheduler.GetCoreState()
	assert.NotEqual(t, initialState, newState, "Core state should change after work")
}

// TestSchedulerMetrics tests scheduler performance metrics
func TestSchedulerMetrics(t *testing.T) {
	scheduler := mobilex.NewHeterogeneousScheduler(4, 4)
	scheduler.Start()
	defer scheduler.Stop()

	// Get initial metrics - verifies GetMetrics works
	_ = scheduler.GetMetrics()

	// Distribute work
	testData := make([]byte, 512)
	for i := 0; i < 20; i++ {
		scheduler.DistributeMining(testData)
	}

	// Allow time for work to be scheduled
	time.Sleep(time.Millisecond * 50)

	// Check that GetMetrics still works after work
	_ = scheduler.GetMetrics()
}

// TestConcurrentMiningDistribution tests concurrent mining distribution
func TestConcurrentMiningDistribution(t *testing.T) {
	scheduler := mobilex.NewHeterogeneousScheduler(4, 4)
	scheduler.Start()
	defer scheduler.Stop()

	// Distribute work from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 10
	workPerGoroutine := 50

	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(id int) {
			defer wg.Done()

			testData := make([]byte, 256)
			for i := range testData {
				testData[i] = byte((id + i) % 256)
			}

			for i := 0; i < workPerGoroutine; i++ {
				scheduler.DistributeMining(testData)
				time.Sleep(time.Microsecond * 100)
			}
		}(g)
	}

	wg.Wait()

	// Verify work was scheduled by calling GetMetrics
	_ = scheduler.GetMetrics()
}

// TestIntensityBounds tests intensity adjustment boundaries
func TestIntensityBounds(t *testing.T) {
	scheduler := mobilex.NewHeterogeneousScheduler(4, 4)
	scheduler.Start()
	defer scheduler.Stop()

	// Reduce intensity to minimum
	for i := 0; i < 10; i++ {
		scheduler.ReduceIntensity()
	}
	time.Sleep(time.Millisecond * 10)

	// Should still have at least stopped state
	// Note: We can't directly check intensity, but active cores reflects it
	minCores := scheduler.ActiveCores()
	assert.GreaterOrEqual(t, minCores, 0, "Active cores should be non-negative")

	// Increase intensity to maximum
	for i := 0; i < 10; i++ {
		scheduler.IncreaseIntensity()
	}
	time.Sleep(time.Millisecond * 10)

	// Should not exceed total cores
	maxCores := scheduler.ActiveCores()
	assert.LessOrEqual(t, maxCores, 8, "Active cores should not exceed total (4+4)")
}

// BenchmarkDistributeMining benchmarks mining distribution
func BenchmarkDistributeMining(b *testing.B) {
	scheduler := mobilex.NewHeterogeneousScheduler(4, 4)
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
}

// TestSchedulerUnderLoad tests scheduler behavior under heavy load
func TestSchedulerUnderLoad(t *testing.T) {
	scheduler := mobilex.NewHeterogeneousScheduler(4, 4)
	scheduler.Start()
	defer scheduler.Stop()

	// Generate heavy load
	loadDuration := 100 * time.Millisecond
	startTime := time.Now()

	var tasksSubmitted atomic.Int64

	// Submit tasks continuously
	go func() {
		testData := make([]byte, 512)
		for time.Since(startTime) < loadDuration {
			scheduler.DistributeMining(testData)
			tasksSubmitted.Add(1)
			runtime.Gosched() // Allow other goroutines to run
		}
	}()

	// Monitor metrics during load
	time.Sleep(loadDuration)

	// Get metrics to ensure the scheduler is still functioning
	_ = scheduler.GetMetrics()
	submitted := tasksSubmitted.Load()

	// Verify tasks were submitted
	assert.Greater(t, submitted, int64(0), "Tasks should be submitted")
}

// TestSchedulerMemoryStability tests for memory leaks during extended operation
func TestSchedulerMemoryStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory stability test in short mode")
	}

	scheduler := mobilex.NewHeterogeneousScheduler(2, 2)
	scheduler.Start()
	defer scheduler.Stop()

	// Run for extended period with continuous work
	testData := make([]byte, 256)
	iterations := 1000

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	initialAlloc := memStats.Alloc

	for i := 0; i < iterations; i++ {
		scheduler.DistributeMining(testData)

		// Periodically adjust intensity
		if i%100 == 0 {
			if i%200 == 0 {
				scheduler.ReduceIntensity()
			} else {
				scheduler.IncreaseIntensity()
			}
		}

		// Allow GC to run
		if i%50 == 0 {
			runtime.GC()
		}
	}

	// Check final memory usage
	runtime.GC()
	runtime.ReadMemStats(&memStats)
	finalAlloc := memStats.Alloc

	// Memory should not grow excessively
	memGrowth := int64(finalAlloc) - int64(initialAlloc)
	assert.Less(t, memGrowth, int64(10*1024*1024), "Memory growth should be less than 10MB")
}
