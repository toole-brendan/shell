// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mobilex

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// MiningMetrics contains real-time mining performance metrics.
type MiningMetrics struct {
	HashRate        float64       // Hashes per second
	HashesCompleted uint64        // Total hashes completed
	Temperature     float64       // Current temperature in Celsius
	PowerUsage      float64       // Current power consumption in watts
	NPUUtilization  float64       // NPU utilization percentage
	Duration        time.Duration // Time since mining started
}

// MetricsCollector collects and aggregates mining metrics.
type MetricsCollector struct {
	metrics      []MiningMetrics
	metricsMutex sync.RWMutex
	maxMetrics   int

	errors      map[string]uint64
	errorsMutex sync.RWMutex

	startTime time.Time
	running   int32 // atomic

	// Aggregated metrics
	totalHashes     uint64 // atomic
	avgHashRate     float64
	avgTemperature  float64
	avgPowerUsage   float64
	peakHashRate    float64
	peakTemperature float64
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics:    make([]MiningMetrics, 0, 1000),
		maxMetrics: 1000,
		errors:     make(map[string]uint64),
		startTime:  time.Now(),
	}
}

// Start begins metrics collection.
func (mc *MetricsCollector) Start(ctx context.Context) {
	if !atomic.CompareAndSwapInt32(&mc.running, 0, 1) {
		return
	}

	// Start aggregation loop
	go mc.aggregationLoop(ctx)
}

// Stop stops metrics collection.
func (mc *MetricsCollector) Stop() {
	atomic.StoreInt32(&mc.running, 0)
}

// Record records a new set of metrics.
func (mc *MetricsCollector) Record(metrics MiningMetrics) {
	mc.metricsMutex.Lock()
	defer mc.metricsMutex.Unlock()

	// Add to metrics history
	mc.metrics = append(mc.metrics, metrics)

	// Maintain maximum size
	if len(mc.metrics) > mc.maxMetrics {
		mc.metrics = mc.metrics[len(mc.metrics)-mc.maxMetrics:]
	}

	// Update atomic counters
	atomic.StoreUint64(&mc.totalHashes, metrics.HashesCompleted)

	// Update peak values
	if metrics.HashRate > mc.peakHashRate {
		mc.peakHashRate = metrics.HashRate
	}
	if metrics.Temperature > mc.peakTemperature {
		mc.peakTemperature = metrics.Temperature
	}
}

// RecordError records an error occurrence.
func (mc *MetricsCollector) RecordError(errorType string, err error) {
	mc.errorsMutex.Lock()
	defer mc.errorsMutex.Unlock()

	mc.errors[errorType]++
}

// GetCurrentMetrics returns the most recent metrics.
func (mc *MetricsCollector) GetCurrentMetrics() MiningMetrics {
	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()

	if len(mc.metrics) == 0 {
		return MiningMetrics{}
	}

	return mc.metrics[len(mc.metrics)-1]
}

// GetAverageMetrics returns averaged metrics over the collection period.
func (mc *MetricsCollector) GetAverageMetrics() MiningMetrics {
	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()

	if len(mc.metrics) == 0 {
		return MiningMetrics{}
	}

	var totalHashRate, totalTemp, totalPower, totalNPU float64
	for _, m := range mc.metrics {
		totalHashRate += m.HashRate
		totalTemp += m.Temperature
		totalPower += m.PowerUsage
		totalNPU += m.NPUUtilization
	}

	count := float64(len(mc.metrics))
	return MiningMetrics{
		HashRate:        totalHashRate / count,
		HashesCompleted: atomic.LoadUint64(&mc.totalHashes),
		Temperature:     totalTemp / count,
		PowerUsage:      totalPower / count,
		NPUUtilization:  totalNPU / count,
		Duration:        time.Since(mc.startTime),
	}
}

// GetMetricsSummary returns a comprehensive metrics summary.
func (mc *MetricsCollector) GetMetricsSummary() MetricsSummary {
	mc.metricsMutex.RLock()
	avgMetrics := mc.GetAverageMetrics()
	mc.metricsMutex.RUnlock()

	mc.errorsMutex.RLock()
	errorCopy := make(map[string]uint64)
	for k, v := range mc.errors {
		errorCopy[k] = v
	}
	mc.errorsMutex.RUnlock()

	return MetricsSummary{
		StartTime:       mc.startTime,
		Duration:        time.Since(mc.startTime),
		TotalHashes:     atomic.LoadUint64(&mc.totalHashes),
		AverageHashRate: avgMetrics.HashRate,
		PeakHashRate:    mc.peakHashRate,
		AverageTemp:     avgMetrics.Temperature,
		PeakTemp:        mc.peakTemperature,
		AveragePower:    avgMetrics.PowerUsage,
		AverageNPU:      avgMetrics.NPUUtilization,
		Errors:          errorCopy,
		SampleCount:     len(mc.metrics),
	}
}

// MetricsSummary contains a comprehensive summary of mining metrics.
type MetricsSummary struct {
	StartTime       time.Time
	Duration        time.Duration
	TotalHashes     uint64
	AverageHashRate float64
	PeakHashRate    float64
	AverageTemp     float64
	PeakTemp        float64
	AveragePower    float64
	AverageNPU      float64
	Errors          map[string]uint64
	SampleCount     int
}

// aggregationLoop continuously aggregates metrics.
func (mc *MetricsCollector) aggregationLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for atomic.LoadInt32(&mc.running) == 1 {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.updateAggregates()
		}
	}
}

// updateAggregates updates the aggregated metrics.
func (mc *MetricsCollector) updateAggregates() {
	mc.metricsMutex.Lock()
	defer mc.metricsMutex.Unlock()

	if len(mc.metrics) == 0 {
		return
	}

	// Calculate moving averages
	windowSize := 100
	if len(mc.metrics) < windowSize {
		windowSize = len(mc.metrics)
	}

	recentMetrics := mc.metrics[len(mc.metrics)-windowSize:]

	var totalHashRate, totalTemp, totalPower float64
	for _, m := range recentMetrics {
		totalHashRate += m.HashRate
		totalTemp += m.Temperature
		totalPower += m.PowerUsage
	}

	count := float64(len(recentMetrics))
	mc.avgHashRate = totalHashRate / count
	mc.avgTemperature = totalTemp / count
	mc.avgPowerUsage = totalPower / count
}

// GetHashRateHistory returns historical hash rate data.
func (mc *MetricsCollector) GetHashRateHistory(duration time.Duration) []HashRatePoint {
	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()

	cutoff := time.Now().Add(-duration)
	points := make([]HashRatePoint, 0)

	for _, m := range mc.metrics {
		timestamp := mc.startTime.Add(m.Duration)
		if timestamp.After(cutoff) {
			points = append(points, HashRatePoint{
				Timestamp: timestamp,
				HashRate:  m.HashRate,
			})
		}
	}

	return points
}

// HashRatePoint represents a hash rate measurement at a specific time.
type HashRatePoint struct {
	Timestamp time.Time
	HashRate  float64
}

// GetEfficiency calculates mining efficiency (hashes per watt).
func (mc *MetricsCollector) GetEfficiency() float64 {
	avg := mc.GetAverageMetrics()
	if avg.PowerUsage <= 0 {
		return 0
	}
	return avg.HashRate / avg.PowerUsage
}

// GetThermalEfficiency calculates thermal efficiency score.
func (mc *MetricsCollector) GetThermalEfficiency() float64 {
	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()

	if len(mc.metrics) == 0 {
		return 0
	}

	// Calculate percentage of time within optimal temperature range
	optimalCount := 0
	for _, m := range mc.metrics {
		if m.Temperature >= 35 && m.Temperature <= 45 {
			optimalCount++
		}
	}

	return float64(optimalCount) / float64(len(mc.metrics)) * 100
}

// Reset clears all collected metrics.
func (mc *MetricsCollector) Reset() {
	mc.metricsMutex.Lock()
	defer mc.metricsMutex.Unlock()

	mc.metrics = mc.metrics[:0]
	mc.totalHashes = 0
	mc.avgHashRate = 0
	mc.avgTemperature = 0
	mc.avgPowerUsage = 0
	mc.peakHashRate = 0
	mc.peakTemperature = 0
	mc.startTime = time.Now()

	mc.errorsMutex.Lock()
	mc.errors = make(map[string]uint64)
	mc.errorsMutex.Unlock()
}
