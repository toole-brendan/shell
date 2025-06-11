// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mobilex

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/toole-brendan/shell/wire"
)

// Errors for thermal verification.
var (
	ErrThermalProofInvalid    = errors.New("thermal proof is invalid")
	ErrThermalLimitExceeded   = errors.New("thermal limit exceeded")
	ErrThermalProofTimeout    = errors.New("thermal proof validation timeout")
	ErrNoPMUAccess            = errors.New("no access to performance monitoring unit")
	ErrThermalDataUnavailable = errors.New("thermal data unavailable")
)

// ARMPMUCounters provides access to ARM Performance Monitoring Unit counters.
type ARMPMUCounters struct {
	// These would be implemented via CGO to access ARM PMU registers
	// For now, this is a placeholder interface
	cycleCounterSupported       bool
	instructionCounterSupported bool
}

// ReadCycleCount reads the current CPU cycle counter.
func (pmu *ARMPMUCounters) ReadCycleCount() uint64 {
	// In real implementation, this would use CGO to access ARM PMU registers
	// For example: PMCCNTR_EL0 (Performance Monitors Cycle Count Register)
	// return readPMCCNTR()
	return uint64(time.Now().UnixNano())
}

// ReadInstructionCount reads the current instruction counter.
func (pmu *ARMPMUCounters) ReadInstructionCount() uint64 {
	// In real implementation, this would access ARM PMU instruction counter
	return uint64(time.Now().UnixNano() / 2)
}

// ThermalProof represents a proof of thermal compliance during mining.
type ThermalProof struct {
	CycleCount     uint64   // Actual cycles used
	ExpectedCycles uint64   // Thermal-compliant cycle count
	Frequency      uint64   // Operating frequency in MHz
	Temperature    float64  // SoC temperature in Celsius
	Timestamp      int64    // Proof generation time
	WorkHash       [32]byte // Hash of the work being validated
}

// ThermalVerification manages thermal proof generation and validation.
type ThermalVerification struct {
	pmcCounters    *ARMPMUCounters
	baseFreq       uint64  // Expected CPU frequency in MHz
	tolerance      float64 // Allowed variance (e.g., 5%)
	currentTemp    float64 // Current temperature
	tempMutex      sync.RWMutex
	thermalHistory []ThermalProof
	historyMutex   sync.RWMutex
	maxHistorySize int
	validator      *ThermalValidator
}

// NewThermalVerification creates a new thermal verification system.
func NewThermalVerification(baseFreq uint64, tolerance float64) *ThermalVerification {
	return &ThermalVerification{
		pmcCounters:    &ARMPMUCounters{cycleCounterSupported: true},
		baseFreq:       baseFreq,
		tolerance:      tolerance,
		currentTemp:    40.0, // Default optimal temperature
		maxHistorySize: 1000,
		thermalHistory: make([]ThermalProof, 0, 1000),
		validator:      NewThermalValidator(),
	}
}

// GenerateThermalProof creates a thermal proof for the given header bytes.
func (tv *ThermalVerification) GenerateThermalProof(headerBytes []byte) uint64 {
	// Start cycle counting
	startCycles := tv.pmcCounters.ReadCycleCount()
	startTime := time.Now()

	// Run subset of work at half speed to measure thermal compliance
	testWorkload := headerBytes[:32] // Use first 32 bytes as test workload
	tv.runHalfSpeedHash(testWorkload)

	// Measure elapsed cycles and time
	endCycles := tv.pmcCounters.ReadCycleCount()
	elapsedTime := time.Since(startTime)
	cycleDelta := endCycles - startCycles

	// Calculate effective frequency
	effectiveFreq := uint64(float64(cycleDelta) / elapsedTime.Seconds() / 1e6)

	// Create thermal proof
	proof := ThermalProof{
		CycleCount:     cycleDelta,
		ExpectedCycles: tv.calculateExpectedCycles(len(testWorkload)),
		Frequency:      effectiveFreq,
		Temperature:    tv.getCurrentTemperature(),
		Timestamp:      time.Now().Unix(),
		WorkHash:       sha256.Sum256(headerBytes),
	}

	// Store in history for statistical analysis
	tv.addToHistory(proof)

	// Generate compact proof value
	return tv.encodeProof(proof)
}

// ValidateThermalProof validates a thermal proof from a block header.
func (tv *ThermalVerification) ValidateThermalProof(header *wire.BlockHeader) error {
	// Serialize header for hashing (excluding thermal proof itself)
	headerBytes := serializeHeaderForThermalValidation(header)

	// Re-compute thermal proof for verification
	expectedProof := tv.GenerateThermalProof(headerBytes)

	// Allow tolerance for legitimate thermal differences
	actualProof := header.ThermalProof
	toleranceRange := uint64(float64(expectedProof) * tv.tolerance / 100.0)

	minAcceptable := expectedProof - toleranceRange
	maxAcceptable := expectedProof + toleranceRange

	if actualProof < minAcceptable || actualProof > maxAcceptable {
		return fmt.Errorf("%w: proof %d outside acceptable range [%d, %d]",
			ErrThermalProofInvalid, actualProof, minAcceptable, maxAcceptable)
	}

	return nil
}

// runHalfSpeedHash runs a hash computation at reduced speed for thermal testing.
func (tv *ThermalVerification) runHalfSpeedHash(workload []byte) {
	// This simulates running at 50% clock speed
	// In real implementation, this would use frequency scaling
	hash := sha256.Sum256(workload)

	// Artificial delay to simulate half-speed operation
	time.Sleep(100 * time.Microsecond)

	// Do some work to ensure compiler doesn't optimize this away
	for i := 0; i < 100; i++ {
		hash = sha256.Sum256(hash[:])
	}
}

// calculateExpectedCycles calculates the expected cycle count for a workload.
func (tv *ThermalVerification) calculateExpectedCycles(workloadSize int) uint64 {
	// Base cycles for SHA256 operation
	baseCycles := uint64(workloadSize) * 100 // Rough estimate

	// Adjust for temperature
	temp := tv.getCurrentTemperature()
	thermalMultiplier := 1.0

	if temp > 45.0 {
		// Higher temperature = slower expected performance
		thermalMultiplier = 1.0 + (temp-45.0)*0.02
	} else if temp < 35.0 {
		// Lower temperature = faster expected performance
		thermalMultiplier = 1.0 - (35.0-temp)*0.01
	}

	return uint64(float64(baseCycles) * thermalMultiplier)
}

// encodeProof encodes a thermal proof into a compact uint64.
func (tv *ThermalVerification) encodeProof(proof ThermalProof) uint64 {
	// Combine various proof elements into a single uint64
	// This is a simplified encoding - real implementation would be more sophisticated
	data := make([]byte, 32)
	binary.LittleEndian.PutUint64(data[0:8], proof.CycleCount)
	binary.LittleEndian.PutUint64(data[8:16], proof.ExpectedCycles)
	binary.LittleEndian.PutUint64(data[16:24], proof.Frequency)
	binary.LittleEndian.PutUint64(data[24:32], uint64(proof.Temperature*100))

	hash := sha256.Sum256(data)
	return binary.LittleEndian.Uint64(hash[:8])
}

// getCurrentTemperature returns the current SoC temperature.
func (tv *ThermalVerification) getCurrentTemperature() float64 {
	tv.tempMutex.RLock()
	defer tv.tempMutex.RUnlock()
	return tv.currentTemp
}

// UpdateTemperature updates the current temperature reading.
func (tv *ThermalVerification) UpdateTemperature(temp float64) {
	tv.tempMutex.Lock()
	defer tv.tempMutex.Unlock()
	tv.currentTemp = temp
}

// addToHistory adds a thermal proof to the history for statistical analysis.
func (tv *ThermalVerification) addToHistory(proof ThermalProof) {
	tv.historyMutex.Lock()
	defer tv.historyMutex.Unlock()

	tv.thermalHistory = append(tv.thermalHistory, proof)

	// Maintain maximum history size
	if len(tv.thermalHistory) > tv.maxHistorySize {
		tv.thermalHistory = tv.thermalHistory[len(tv.thermalHistory)-tv.maxHistorySize:]
	}
}

// GetThermalStatistics returns statistical analysis of thermal history.
func (tv *ThermalVerification) GetThermalStatistics() ThermalStatistics {
	tv.historyMutex.RLock()
	defer tv.historyMutex.RUnlock()

	if len(tv.thermalHistory) == 0 {
		return ThermalStatistics{}
	}

	var totalTemp float64
	var totalFreq float64
	minTemp := math.MaxFloat64
	maxTemp := 0.0

	for _, proof := range tv.thermalHistory {
		totalTemp += proof.Temperature
		totalFreq += float64(proof.Frequency)

		if proof.Temperature < minTemp {
			minTemp = proof.Temperature
		}
		if proof.Temperature > maxTemp {
			maxTemp = proof.Temperature
		}
	}

	count := float64(len(tv.thermalHistory))
	avgTemp := totalTemp / count
	avgFreq := totalFreq / count

	// Calculate standard deviation
	var tempVariance float64
	for _, proof := range tv.thermalHistory {
		diff := proof.Temperature - avgTemp
		tempVariance += diff * diff
	}
	tempStdDev := math.Sqrt(tempVariance / count)

	return ThermalStatistics{
		AverageTemperature: avgTemp,
		MinTemperature:     minTemp,
		MaxTemperature:     maxTemp,
		StdDevTemperature:  tempStdDev,
		AverageFrequency:   avgFreq,
		SampleCount:        len(tv.thermalHistory),
	}
}

// ThermalStatistics contains statistical analysis of thermal history.
type ThermalStatistics struct {
	AverageTemperature float64
	MinTemperature     float64
	MaxTemperature     float64
	StdDevTemperature  float64
	AverageFrequency   float64
	SampleCount        int
}

// ThermalValidator performs validation of thermal proofs.
type ThermalValidator struct {
	validationCache sync.Map // Cache of recently validated proofs
}

// NewThermalValidator creates a new thermal validator.
func NewThermalValidator() *ThermalValidator {
	return &ThermalValidator{}
}

// ValidateWithRecomputation validates a thermal proof by recomputing at reduced speed.
func (tv *ThermalValidator) ValidateWithRecomputation(header *wire.BlockHeader, clockSpeed float64) error {
	// Check cache first
	cacheKey := header.BlockHash()
	if cached, ok := tv.validationCache.Load(cacheKey); ok {
		if err, ok := cached.(error); ok {
			return err
		}
		return nil
	}

	// Perform validation at reduced clock speed
	// In real implementation, this would actually reduce CPU frequency
	startTime := time.Now()

	// Simulate reduced speed validation
	headerBytes := serializeHeaderForThermalValidation(header)
	workload := headerBytes[:64] // Use larger subset for validation

	// Run validation workload
	for i := 0; i < 1000; i++ {
		hash := sha256.Sum256(workload)
		workload = hash[:]

		// Simulate clock speed reduction
		sleepDuration := time.Duration(float64(100) * (1.0 - clockSpeed) * float64(time.Microsecond))
		time.Sleep(sleepDuration)
	}

	elapsed := time.Since(startTime)

	// Check if elapsed time is within acceptable range
	expectedTime := time.Duration(float64(100*time.Millisecond) / clockSpeed)
	tolerance := expectedTime / 10 // 10% tolerance

	var validationErr error
	if elapsed < expectedTime-tolerance || elapsed > expectedTime+tolerance {
		validationErr = fmt.Errorf("%w: validation took %v, expected %v±%v",
			ErrThermalProofInvalid, elapsed, expectedTime, tolerance)
	}

	// Cache the result
	tv.validationCache.Store(cacheKey, validationErr)

	return validationErr
}

// serializeHeaderForThermalValidation serializes a block header for thermal validation,
// excluding the thermal proof itself to avoid circular dependency.
func serializeHeaderForThermalValidation(header *wire.BlockHeader) []byte {
	// Create a copy without thermal proof
	headerCopy := *header
	headerCopy.ThermalProof = 0

	// Serialize to bytes
	var buf [80]byte // Original header size without thermal proof
	binary.LittleEndian.PutUint32(buf[0:4], uint32(headerCopy.Version))
	copy(buf[4:36], headerCopy.PrevBlock[:])
	copy(buf[36:68], headerCopy.MerkleRoot[:])
	binary.LittleEndian.PutUint32(buf[68:72], uint32(headerCopy.Timestamp.Unix()))
	binary.LittleEndian.PutUint32(buf[72:76], headerCopy.Bits)
	binary.LittleEndian.PutUint32(buf[76:80], headerCopy.Nonce)

	return buf[:]
}

// DetectThermalCheating performs statistical analysis to detect systematic thermal cheating.
func DetectThermalCheating(proofs []ThermalProof, threshold float64) []int {
	if len(proofs) < 10 {
		return nil // Not enough data
	}

	// Calculate mean and standard deviation of temperatures
	var sum float64
	for _, proof := range proofs {
		sum += proof.Temperature
	}
	mean := sum / float64(len(proofs))

	var variance float64
	for _, proof := range proofs {
		diff := proof.Temperature - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(proofs)))

	// Find outliers (Z-score > threshold)
	var outliers []int
	for i, proof := range proofs {
		zScore := math.Abs(proof.Temperature-mean) / stdDev
		if zScore > threshold {
			outliers = append(outliers, i)
		}
	}

	return outliers
}
