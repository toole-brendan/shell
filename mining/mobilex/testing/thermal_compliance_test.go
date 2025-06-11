package testing

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toole-brendan/shell/mining/mobilex"
	"github.com/toole-brendan/shell/wire"
)

// TestThermalProofGeneration tests the generation of thermal proofs
func TestThermalProofGeneration(t *testing.T) {
	tests := []struct {
		name          string
		baseFreq      uint64
		tolerance     float64
		headerData    []byte
		expectNonZero bool
	}{
		{
			name:          "Normal mining operation",
			baseFreq:      2000, // 2GHz
			tolerance:     5.0,  // 5% tolerance
			headerData:    make([]byte, 80),
			expectNonZero: true,
		},
		{
			name:          "High frequency operation",
			baseFreq:      3000, // 3GHz
			tolerance:     10.0, // 10% tolerance
			headerData:    make([]byte, 80),
			expectNonZero: true,
		},
		{
			name:          "Low frequency operation",
			baseFreq:      1000, // 1GHz
			tolerance:     3.0,  // 3% tolerance
			headerData:    make([]byte, 80),
			expectNonZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create thermal verification instance
			tv := mobilex.NewThermalVerification(tt.baseFreq, tt.tolerance)

			// Fill header data with random bytes
			rand.Read(tt.headerData)

			// Generate thermal proof
			proof := tv.GenerateThermalProof(tt.headerData)

			// Verify proof is non-zero
			if tt.expectNonZero {
				assert.NotEqual(t, uint64(0), proof, "Thermal proof should not be zero")
			}
		})
	}
}

// TestThermalProofValidation tests validation of thermal proofs
func TestThermalProofValidation(t *testing.T) {
	tv := mobilex.NewThermalVerification(2000, 5.0) // 2GHz, 5% tolerance

	tests := []struct {
		name        string
		header      *wire.BlockHeader
		expectError bool
		description string
	}{
		{
			name: "Valid thermal proof",
			header: &wire.BlockHeader{
				Version:      1,
				ThermalProof: 0, // Will be set by generating actual proof
				Timestamp:    time.Now(),
				Bits:         0x1d00ffff,
				Nonce:        12345,
			},
			expectError: false,
			description: "Normal operating temperature",
		},
		{
			name: "Invalid thermal proof - too low",
			header: &wire.BlockHeader{
				Version:      1,
				ThermalProof: 1, // Artificially low value
				Timestamp:    time.Now(),
				Bits:         0x1d00ffff,
				Nonce:        12346,
			},
			expectError: true,
			description: "Thermal proof indicates overclocking",
		},
		{
			name: "Invalid thermal proof - too high",
			header: &wire.BlockHeader{
				Version:      1,
				ThermalProof: ^uint64(0), // Max uint64 value
				Timestamp:    time.Now(),
				Bits:         0x1d00ffff,
				Nonce:        12347,
			},
			expectError: true,
			description: "Thermal proof indicates severe throttling",
		},
	}

	// Generate a valid proof for the first test case
	if len(tests) > 0 && tests[0].header != nil {
		headerBytes := serializeHeaderForTest(tests[0].header)
		tests[0].header.ThermalProof = tv.GenerateThermalProof(headerBytes)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate thermal proof
			err := tv.ValidateThermalProof(tt.header)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestThermalStatistics tests thermal statistics collection
func TestThermalStatistics(t *testing.T) {
	tv := mobilex.NewThermalVerification(2000, 5.0)

	// Generate multiple thermal proofs to build history
	for i := 0; i < 100; i++ {
		headerData := make([]byte, 80)
		rand.Read(headerData)

		// Update temperature to simulate variations
		temp := 38.0 + float64(i%10)*0.5
		tv.UpdateTemperature(temp)

		tv.GenerateThermalProof(headerData)
	}

	// Get statistics
	stats := tv.GetThermalStatistics()

	// Verify statistics are reasonable
	assert.Greater(t, stats.AverageTemperature, 35.0, "Average temperature too low")
	assert.Less(t, stats.AverageTemperature, 45.0, "Average temperature too high")
	assert.Greater(t, stats.SampleCount, 0, "Should have samples")
	assert.Greater(t, stats.StdDevTemperature, 0.0, "Should have temperature variation")
}

// TestDeviceProfiles tests device-specific configuration
func TestDeviceProfiles(t *testing.T) {
	devices := []struct {
		name          string
		deviceName    string
		expectedSoC   string
		hasNPU        bool
		expectedClass string
	}{
		{
			name:          "iPhone 15 Pro profile",
			deviceName:    "iPhone 15 Pro",
			expectedSoC:   "A17 Pro",
			hasNPU:        true,
			expectedClass: "flagship",
		},
		{
			name:          "Galaxy S24 profile",
			deviceName:    "Galaxy S24",
			expectedSoC:   "Snapdragon 8 Gen 3",
			hasNPU:        true,
			expectedClass: "flagship",
		},
		{
			name:          "Unknown device profile",
			deviceName:    "Unknown Device",
			expectedSoC:   "Unknown",
			hasNPU:        false,
			expectedClass: "budget",
		},
	}

	for _, device := range devices {
		t.Run(device.name, func(t *testing.T) {
			// Get device profile
			profile := mobilex.GetDeviceProfile(device.deviceName)

			// Verify profile properties
			assert.Equal(t, device.expectedSoC, profile.SoC)
			assert.Equal(t, device.hasNPU, profile.HasNPU)
			assert.Equal(t, device.expectedClass, profile.ThermalClass)

			// Test configuration optimization
			cfg := mobilex.DefaultConfig()
			cfg.OptimizeForDevice(profile)

			// Verify configuration was adjusted
			if device.expectedClass == "budget" {
				assert.Equal(t, uint64(256*1024*1024), cfg.RandomXMemory,
					"Budget device should use light mode")
			}
		})
	}
}

// TestMiningIntensityLevels tests different mining intensity configurations
func TestMiningIntensityLevels(t *testing.T) {
	cfg := mobilex.DefaultConfig()

	tests := []struct {
		name      string
		intensity mobilex.MiningIntensity
		maxCores  int
		maxPower  float64
		maxTemp   float64
	}{
		{
			name:      "Light intensity",
			intensity: cfg.IntensityLight,
			maxCores:  2,
			maxPower:  2.0,
			maxTemp:   42.0,
		},
		{
			name:      "Medium intensity",
			intensity: cfg.IntensityMedium,
			maxCores:  4,
			maxPower:  4.0,
			maxTemp:   45.0,
		},
		{
			name:      "Full intensity",
			intensity: cfg.IntensityFull,
			maxCores:  8,
			maxPower:  8.0,
			maxTemp:   48.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.maxCores, tt.intensity.CoreCount)
			assert.Equal(t, tt.maxPower, tt.intensity.PowerLimit)
			assert.Equal(t, tt.maxTemp, tt.intensity.ThermalLimit)
		})
	}
}

// TestThermalProofSerialization tests serialization/deserialization
func TestThermalProofSerialization(t *testing.T) {
	originalHeader := &wire.BlockHeader{
		Version:      1,
		ThermalProof: 1234567890,
		Timestamp:    time.Now(),
		Bits:         0x1d00ffff,
		Nonce:        12345,
	}

	// Serialize using wire package functions
	var buf bytes.Buffer
	err := originalHeader.Serialize(&buf)
	require.NoError(t, err)
	require.Equal(t, wire.MaxBlockHeaderPayload, buf.Len(), "Header size should be 88 bytes with thermal proof")

	// Deserialize
	deserializedHeader := &wire.BlockHeader{}
	err = deserializedHeader.Deserialize(&buf)
	require.NoError(t, err)

	// Verify thermal proof survived serialization
	assert.Equal(t, originalHeader.ThermalProof, deserializedHeader.ThermalProof)
}

// TestValidationParameters tests thermal validation parameters
func TestValidationParameters(t *testing.T) {
	params := mobilex.DefaultValidationParams()

	assert.Equal(t, 0.10, params.RandomValidationRate, "Should validate 10% of blocks")
	assert.Equal(t, 0.50, params.ValidationClockSpeed, "Should run at 50% clock speed")
	assert.Equal(t, 1000, params.StatisticalWindowSize, "Should analyze 1000 blocks")
	assert.Equal(t, 3.0, params.ThermalOutlierThreshold, "Should use 3 sigma threshold")
	assert.Equal(t, 30*time.Second, params.ValidationTimeout, "Should timeout after 30s")
}

// Helper function to serialize header for testing
func serializeHeaderForTest(header *wire.BlockHeader) []byte {
	// Create a copy without thermal proof to match the validation logic
	headerCopy := *header
	headerCopy.ThermalProof = 0

	// Simple serialization for testing
	buf := make([]byte, 80)
	// In real implementation, this would use proper wire protocol serialization
	return buf
}
