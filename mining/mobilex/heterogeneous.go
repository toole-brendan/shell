// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mobilex

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// CoreType represents the type of CPU core.
type CoreType int

const (
	BigCore CoreType = iota
	LittleCore
)

// TaskType represents different types of mining tasks.
type TaskType int

const (
	VectorOps TaskType = iota
	MemoryAccess
	MainHash
	NPUCoordination
)

// CPUCore represents a single CPU core.
type CPUCore struct {
	ID       int
	Type     CoreType
	Active   bool
	Load     float64
	Affinity uint64
}

// MiningTask represents a unit of mining work.
type MiningTask struct {
	Type     TaskType
	Data     []byte
	CoreType CoreType
	Priority int
}

// WorkSplitter distributes tasks across cores.
type WorkSplitter struct {
	taskQueue   chan MiningTask
	resultQueue chan []byte
	workers     []*Worker
}

// Worker represents a single mining worker thread.
type Worker struct {
	id       int
	core     *CPUCore
	taskChan chan MiningTask
	quit     chan struct{}
}

// HeterogeneousScheduler manages work distribution across heterogeneous cores.
type HeterogeneousScheduler struct {
	bigCores     []*CPUCore
	littleCores  []*CPUCore
	workSplitter *WorkSplitter
	syncInterval int

	activeCores int32 // atomic
	intensity   int32 // atomic: 0=stopped, 1=low, 2=medium, 3=high
	running     int32 // atomic

	mutex            sync.RWMutex
	metricsCollector *schedulerMetrics
}

// schedulerMetrics tracks scheduler performance.
type schedulerMetrics struct {
	tasksScheduled        uint64
	tasksCompleted        uint64
	bigCoreUtilization    float64
	littleCoreUtilization float64
}

// NewHeterogeneousScheduler creates a new heterogeneous core scheduler.
func NewHeterogeneousScheduler(bigCoreCount, littleCoreCount int) *HeterogeneousScheduler {
	scheduler := &HeterogeneousScheduler{
		bigCores:         make([]*CPUCore, bigCoreCount),
		littleCores:      make([]*CPUCore, littleCoreCount),
		syncInterval:     75,
		intensity:        2, // Start at medium intensity
		metricsCollector: &schedulerMetrics{},
	}

	// Initialize big cores
	for i := 0; i < bigCoreCount; i++ {
		scheduler.bigCores[i] = &CPUCore{
			ID:   i,
			Type: BigCore,
		}
	}

	// Initialize little cores
	for i := 0; i < littleCoreCount; i++ {
		scheduler.littleCores[i] = &CPUCore{
			ID:   bigCoreCount + i,
			Type: LittleCore,
		}
	}

	// Initialize work splitter
	scheduler.workSplitter = &WorkSplitter{
		taskQueue:   make(chan MiningTask, 100),
		resultQueue: make(chan []byte, 100),
		workers:     make([]*Worker, 0, bigCoreCount+littleCoreCount),
	}

	return scheduler
}

// Start begins the heterogeneous scheduling.
func (hs *HeterogeneousScheduler) Start() {
	if !atomic.CompareAndSwapInt32(&hs.running, 0, 1) {
		return
	}

	// Start workers based on intensity
	hs.adjustWorkers()

	// Start task distribution
	go hs.taskDistributionLoop()

	// Start synchronization
	go hs.synchronizationLoop()
}

// Stop stops the heterogeneous scheduling.
func (hs *HeterogeneousScheduler) Stop() {
	atomic.StoreInt32(&hs.running, 0)

	// Stop all workers
	for _, worker := range hs.workSplitter.workers {
		close(worker.quit)
	}
}

// DistributeMining distributes mining work across heterogeneous cores.
func (hs *HeterogeneousScheduler) DistributeMining(data []byte) {
	// Performance cores: Main hash computation, vector operations
	bigCoreTasks := []MiningTask{
		{Type: VectorOps, CoreType: BigCore, Priority: 1, Data: data},
		{Type: MainHash, CoreType: BigCore, Priority: 1, Data: data},
	}

	// Efficiency cores: Memory scheduling, NPU coordination
	littleCoreTasks := []MiningTask{
		{Type: MemoryAccess, CoreType: LittleCore, Priority: 2, Data: data},
		{Type: NPUCoordination, CoreType: LittleCore, Priority: 2, Data: data},
	}

	// Execute tasks
	hs.workSplitter.Execute(bigCoreTasks, littleCoreTasks)
}

// Execute runs tasks on appropriate cores.
func (ws *WorkSplitter) Execute(bigTasks, littleTasks []MiningTask) {
	// Queue big core tasks
	for _, task := range bigTasks {
		select {
		case ws.taskQueue <- task:
			// Task queued
		default:
			// Queue full, drop task
		}
	}

	// Queue little core tasks
	for _, task := range littleTasks {
		select {
		case ws.taskQueue <- task:
			// Task queued
		default:
			// Queue full, drop task
		}
	}
}

// ReduceIntensity reduces mining intensity.
func (hs *HeterogeneousScheduler) ReduceIntensity() {
	current := atomic.LoadInt32(&hs.intensity)
	if current > 0 {
		atomic.StoreInt32(&hs.intensity, current-1)
		hs.adjustWorkers()
	}
}

// IncreaseIntensity increases mining intensity.
func (hs *HeterogeneousScheduler) IncreaseIntensity() {
	current := atomic.LoadInt32(&hs.intensity)
	if current < 3 {
		atomic.StoreInt32(&hs.intensity, current+1)
		hs.adjustWorkers()
	}
}

// ActiveCores returns the number of active cores.
func (hs *HeterogeneousScheduler) ActiveCores() int {
	return int(atomic.LoadInt32(&hs.activeCores))
}

// adjustWorkers adjusts the number of active workers based on intensity.
func (hs *HeterogeneousScheduler) adjustWorkers() {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	intensity := atomic.LoadInt32(&hs.intensity)

	var targetBigCores, targetLittleCores int
	switch intensity {
	case 0: // Stopped
		targetBigCores = 0
		targetLittleCores = 0
	case 1: // Low
		targetBigCores = 1
		targetLittleCores = 1
	case 2: // Medium
		targetBigCores = len(hs.bigCores) / 2
		targetLittleCores = len(hs.littleCores) / 2
	case 3: // High
		targetBigCores = len(hs.bigCores)
		targetLittleCores = len(hs.littleCores)
	}

	// Update active cores
	activeCores := 0

	// Activate/deactivate big cores
	for i := 0; i < len(hs.bigCores); i++ {
		if i < targetBigCores {
			hs.bigCores[i].Active = true
			activeCores++
		} else {
			hs.bigCores[i].Active = false
		}
	}

	// Activate/deactivate little cores
	for i := 0; i < len(hs.littleCores); i++ {
		if i < targetLittleCores {
			hs.littleCores[i].Active = true
			activeCores++
		} else {
			hs.littleCores[i].Active = false
		}
	}

	atomic.StoreInt32(&hs.activeCores, int32(activeCores))
}

// taskDistributionLoop continuously distributes tasks to workers.
func (hs *HeterogeneousScheduler) taskDistributionLoop() {
	for atomic.LoadInt32(&hs.running) == 1 {
		select {
		case task := <-hs.workSplitter.taskQueue:
			// Find appropriate worker for task
			worker := hs.findWorkerForTask(task)
			if worker != nil {
				select {
				case worker.taskChan <- task:
					atomic.AddUint64(&hs.metricsCollector.tasksScheduled, 1)
				default:
					// Worker busy, requeue
					hs.workSplitter.taskQueue <- task
				}
			}
		default:
			// No tasks, yield CPU
			runtime.Gosched()
		}
	}
}

// findWorkerForTask finds an appropriate worker for the given task.
func (hs *HeterogeneousScheduler) findWorkerForTask(task MiningTask) *Worker {
	hs.mutex.RLock()
	defer hs.mutex.RUnlock()

	// Look for matching core type
	for _, worker := range hs.workSplitter.workers {
		if worker.core.Type == task.CoreType && worker.core.Active {
			return worker
		}
	}

	// Fallback to any active worker
	for _, worker := range hs.workSplitter.workers {
		if worker.core.Active {
			return worker
		}
	}

	return nil
}

// synchronizationLoop handles periodic synchronization between cores.
func (hs *HeterogeneousScheduler) synchronizationLoop() {
	ticker := time.NewTicker(time.Duration(hs.syncInterval) * time.Millisecond)
	defer ticker.Stop()

	for atomic.LoadInt32(&hs.running) == 1 {
		select {
		case <-ticker.C:
			hs.synchronizeCores()
		}
	}
}

// synchronizeCores synchronizes work between big and little cores.
func (hs *HeterogeneousScheduler) synchronizeCores() {
	hs.mutex.RLock()
	defer hs.mutex.RUnlock()

	// Calculate utilization
	var bigCoreLoad, littleCoreLoad float64
	bigCoreCount, littleCoreCount := 0, 0

	for _, core := range hs.bigCores {
		if core.Active {
			bigCoreLoad += core.Load
			bigCoreCount++
		}
	}

	for _, core := range hs.littleCores {
		if core.Active {
			littleCoreLoad += core.Load
			littleCoreCount++
		}
	}

	// Update metrics
	if bigCoreCount > 0 {
		hs.metricsCollector.bigCoreUtilization = bigCoreLoad / float64(bigCoreCount)
	}
	if littleCoreCount > 0 {
		hs.metricsCollector.littleCoreUtilization = littleCoreLoad / float64(littleCoreCount)
	}
}

// GetMetrics returns current scheduler metrics.
func (hs *HeterogeneousScheduler) GetMetrics() schedulerMetrics {
	return schedulerMetrics{
		tasksScheduled:        atomic.LoadUint64(&hs.metricsCollector.tasksScheduled),
		tasksCompleted:        atomic.LoadUint64(&hs.metricsCollector.tasksCompleted),
		bigCoreUtilization:    hs.metricsCollector.bigCoreUtilization,
		littleCoreUtilization: hs.metricsCollector.littleCoreUtilization,
	}
}
