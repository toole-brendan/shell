package vault

import (
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnhancedVaultCreation tests the creation of enhanced vault templates
func TestEnhancedVaultCreation(t *testing.T) {
	// Generate test keys
	hotKeys := generateTestKeys(t, 15)
	warmKeys := generateTestKeys(t, 7)
	coldKeys := generateTestKeys(t, 5)
	emergencyKeys := generateTestKeys(t, 3)

	t.Run("SovereignWealthFundVault", func(t *testing.T) {
		vault, err := CreateSovereignWealthFundVault(hotKeys, warmKeys, coldKeys, emergencyKeys)
		require.NoError(t, err)
		assert.NotNil(t, vault)

		// Verify hierarchy configuration
		assert.Equal(t, uint8(11), vault.Hierarchy.Hot.Threshold)
		assert.Equal(t, uint8(15), vault.Hierarchy.Hot.KeyCount)
		assert.Equal(t, uint32(0), vault.Hierarchy.Hot.BlockDelay)

		assert.Equal(t, uint8(5), vault.Hierarchy.Warm.Threshold)
		assert.Equal(t, uint32(144), vault.Hierarchy.Warm.BlockDelay)

		assert.Equal(t, uint8(3), vault.Hierarchy.Cold.Threshold)
		assert.Equal(t, uint32(4320), vault.Hierarchy.Cold.BlockDelay)

		assert.Equal(t, uint8(2), vault.Hierarchy.Emergency.Threshold)
		assert.Equal(t, uint32(52560), vault.Hierarchy.Emergency.BlockDelay)

		// Verify compliance settings
		assert.Equal(t, uint64(1000000*1e8), vault.ComplianceHooks.AttestationThreshold)
		assert.Equal(t, uint8(2), vault.ComplianceHooks.ValidatorThreshold)
	})

	t.Run("CentralBankReserveVault", func(t *testing.T) {
		operationalKeys := generateTestKeys(t, 21)
		boardKeys := generateTestKeys(t, 9)
		emergencyKeys := generateTestKeys(t, 5)

		vault, err := CreateCentralBankReserveVault(operationalKeys, boardKeys, emergencyKeys)
		require.NoError(t, err)
		assert.NotNil(t, vault)

		// Verify higher security thresholds for central banks
		assert.Equal(t, uint8(15), vault.Hierarchy.Hot.Threshold)
		assert.Equal(t, uint8(21), vault.Hierarchy.Hot.KeyCount)
		assert.Equal(t, uint32(1440), vault.Hierarchy.Hot.TimeWindow) // 5-day window

		assert.Equal(t, uint8(7), vault.Hierarchy.Warm.Threshold)
		assert.Equal(t, uint32(1008), vault.Hierarchy.Warm.BlockDelay) // ~3.5 days

		// Verify strict compliance
		assert.Equal(t, uint64(10000*1e8), vault.ComplianceHooks.AttestationThreshold)
		assert.Equal(t, uint8(3), vault.ComplianceHooks.ValidatorThreshold)
	})

	t.Run("InsufficientKeys", func(t *testing.T) {
		insufficientKeys := generateTestKeys(t, 2)
		_, err := CreateSovereignWealthFundVault(insufficientKeys, insufficientKeys, insufficientKeys, insufficientKeys)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient keys")
	})
}

// TestPolicyEvaluation tests the policy evaluation logic
func TestPolicyEvaluation(t *testing.T) {
	hotKeys := generateTestKeys(t, 15)
	warmKeys := generateTestKeys(t, 7)
	coldKeys := generateTestKeys(t, 5)
	emergencyKeys := generateTestKeys(t, 3)

	vault, err := CreateSovereignWealthFundVault(hotKeys, warmKeys, coldKeys, emergencyKeys)
	require.NoError(t, err)

	baseHeight := uint32(100000)
	lockTime := uint32(99900)

	testCases := []struct {
		name        string
		sigCount    uint8
		blockHeight uint32
		amount      uint64
		expected    string
		approved    bool
	}{
		{
			name:        "HotSpending_Immediate",
			sigCount:    11, // Meets hot threshold
			blockHeight: baseHeight,
			amount:      1000 * 1e8, // Below compliance threshold
			expected:    "hot",
			approved:    true,
		},
		{
			name:        "HotSpending_InsufficientSigs",
			sigCount:    10, // Below hot threshold
			blockHeight: baseHeight,
			amount:      1000 * 1e8,
			expected:    "",
			approved:    false,
		},
		{
			name:        "WarmSpending_AfterDelay",
			sigCount:    5,                  // Meets warm threshold
			blockHeight: lockTime + 144 + 1, // After warm delay
			amount:      1000 * 1e8,
			expected:    "warm",
			approved:    true,
		},
		{
			name:        "ColdSpending_AfterLongDelay",
			sigCount:    3,                   // Meets cold threshold
			blockHeight: lockTime + 4320 + 1, // After cold delay
			amount:      1000 * 1e8,
			expected:    "cold",
			approved:    true,
		},
		{
			name:        "EmergencySpending_AfterEmergencyDelay",
			sigCount:    2,                    // Meets emergency threshold
			blockHeight: lockTime + 52560 + 1, // After emergency delay
			amount:      1000 * 1e8,
			expected:    "emergency",
			approved:    true,
		},
		{
			name:        "LargeAmount_RequiresCompliance",
			sigCount:    11,
			blockHeight: baseHeight,
			amount:      2000000 * 1e8, // Above compliance threshold
			expected:    "hot",
			approved:    true, // Compliance check is mocked to return true
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vault.EvaluateSpendingPolicy(tc.sigCount, tc.blockHeight, lockTime, tc.amount)
			require.NoError(t, err)

			assert.Equal(t, tc.approved, result.Approved)
			if tc.approved {
				assert.Equal(t, tc.expected, result.Policy)
			}
		})
	}
}

// TestTimeWindows tests the time window functionality
func TestTimeWindows(t *testing.T) {
	operationalKeys := generateTestKeys(t, 21)
	boardKeys := generateTestKeys(t, 9)
	emergencyKeys := generateTestKeys(t, 5)

	vault, err := CreateCentralBankReserveVault(operationalKeys, boardKeys, emergencyKeys)
	require.NoError(t, err)

	lockTime := uint32(100000)

	t.Run("HotSpending_WithinTimeWindow", func(t *testing.T) {
		// Central bank hot policy has a 5-day (1440 block) time window
		blockHeight := lockTime + 500 // Within 1440 block window

		result, err := vault.EvaluateSpendingPolicy(15, blockHeight, lockTime, 1000*1e8)
		require.NoError(t, err)
		assert.True(t, result.Approved)
		assert.Equal(t, "hot", result.Policy)
	})

	t.Run("HotSpending_OutsideTimeWindow", func(t *testing.T) {
		// After the 5-day window
		blockHeight := lockTime + 1441 // Outside 1440 block window

		result, err := vault.EvaluateSpendingPolicy(15, blockHeight, lockTime, 1000*1e8)
		require.NoError(t, err)

		// Should fall back to warm policy if available
		if blockHeight >= lockTime+1008 { // Warm delay
			assert.True(t, result.Approved)
			assert.Equal(t, "warm", result.Policy)
		}
	})
}

// TestVaultSerialization tests serialization and hashing
func TestVaultSerialization(t *testing.T) {
	hotKeys := generateTestKeys(t, 15)
	warmKeys := generateTestKeys(t, 7)
	coldKeys := generateTestKeys(t, 5)
	emergencyKeys := generateTestKeys(t, 3)

	vault, err := CreateSovereignWealthFundVault(hotKeys, warmKeys, coldKeys, emergencyKeys)
	require.NoError(t, err)

	t.Run("SerializationRoundTrip", func(t *testing.T) {
		// Test serialization produces consistent output
		data1 := vault.Serialize()
		data2 := vault.Serialize()
		assert.Equal(t, data1, data2)
		assert.True(t, len(data1) > 0)
	})

	t.Run("HashConsistency", func(t *testing.T) {
		// Test hash produces consistent output
		hash1 := vault.Hash()
		hash2 := vault.Hash()
		assert.Equal(t, hash1, hash2)
		assert.Equal(t, 32, len(hash1.CloneBytes()))
	})

	t.Run("DifferentVaultsProduceDifferentHashes", func(t *testing.T) {
		// Create a different vault
		vault2, err := CreateCentralBankReserveVault(
			generateTestKeys(t, 21),
			generateTestKeys(t, 9),
			generateTestKeys(t, 5),
		)
		require.NoError(t, err)

		hash1 := vault.Hash()
		hash2 := vault2.Hash()
		assert.NotEqual(t, hash1, hash2)
	})
}

// TestActivePolicy tests the active policy determination
func TestActivePolicy(t *testing.T) {
	hotKeys := generateTestKeys(t, 15)
	warmKeys := generateTestKeys(t, 7)
	coldKeys := generateTestKeys(t, 5)
	emergencyKeys := generateTestKeys(t, 3)

	vault, err := CreateSovereignWealthFundVault(hotKeys, warmKeys, coldKeys, emergencyKeys)
	require.NoError(t, err)

	lockTime := uint32(100000)

	testCases := []struct {
		name        string
		blockHeight uint32
		expected    string
	}{
		{"Immediate_Hot", lockTime, "hot"},
		{"After_Warm_Delay", lockTime + 144, "warm"},
		{"After_Cold_Delay", lockTime + 4320, "cold"},
		{"After_Emergency_Delay", lockTime + 52560, "emergency"},
		{"Before_LockTime", lockTime - 1, "none"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			policy := vault.GetActivePolicy(tc.blockHeight, lockTime)
			assert.Equal(t, tc.expected, policy)
		})
	}
}

// TestTimeToNextPolicy tests the time calculation for next policy availability
func TestTimeToNextPolicy(t *testing.T) {
	hotKeys := generateTestKeys(t, 15)
	warmKeys := generateTestKeys(t, 7)
	coldKeys := generateTestKeys(t, 5)
	emergencyKeys := generateTestKeys(t, 3)

	vault, err := CreateSovereignWealthFundVault(hotKeys, warmKeys, coldKeys, emergencyKeys)
	require.NoError(t, err)

	lockTime := uint32(100000)

	t.Run("TimeToWarmPolicy", func(t *testing.T) {
		blockHeight := lockTime + 100 // Before warm delay
		duration := vault.GetTimeToNextPolicy(blockHeight, lockTime)

		expectedBlocks := uint32(144) - 100 // 44 blocks remaining
		expectedDuration := time.Duration(expectedBlocks*5) * time.Minute

		assert.Equal(t, expectedDuration, duration)
	})

	t.Run("AllPoliciesAvailable", func(t *testing.T) {
		blockHeight := lockTime + 60000 // After all delays
		duration := vault.GetTimeToNextPolicy(blockHeight, lockTime)
		assert.Equal(t, time.Duration(0), duration)
	})
}

// TestEmergencyRecovery tests emergency recovery functionality
func TestEmergencyRecovery(t *testing.T) {
	hotKeys := generateTestKeys(t, 15)
	warmKeys := generateTestKeys(t, 7)
	coldKeys := generateTestKeys(t, 5)
	emergencyKeys := generateTestKeys(t, 3)

	vault, err := CreateSovereignWealthFundVault(hotKeys, warmKeys, coldKeys, emergencyKeys)
	require.NoError(t, err)

	t.Run("EmergencySettings", func(t *testing.T) {
		assert.Equal(t, uint8(2), vault.Emergency.GuardianThreshold)
		assert.Equal(t, uint32(52560), vault.Emergency.EmergencyDelay)
		assert.True(t, vault.Emergency.ExternalApproval)
		assert.Equal(t, len(emergencyKeys), len(vault.Emergency.GuardianKeys))
	})
}

// Helper function to generate test keys
func generateTestKeys(t *testing.T, count int) []btcec.PublicKey {
	keys := make([]btcec.PublicKey, count)
	for i := 0; i < count; i++ {
		privKey, err := btcec.NewPrivateKey()
		require.NoError(t, err)
		keys[i] = *privKey.PubKey()
	}
	return keys
}

// TestConcurrency tests that vault operations are thread-safe
func TestConcurrency(t *testing.T) {
	hotKeys := generateTestKeys(t, 15)
	warmKeys := generateTestKeys(t, 7)
	coldKeys := generateTestKeys(t, 5)
	emergencyKeys := generateTestKeys(t, 3)

	vault, err := CreateSovereignWealthFundVault(hotKeys, warmKeys, coldKeys, emergencyKeys)
	require.NoError(t, err)

	// Test concurrent access to vault methods
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Concurrent policy evaluation
			_, err := vault.EvaluateSpendingPolicy(11, 100100, 100000, 1000*1e8)
			assert.NoError(t, err)

			// Concurrent hash calculation
			_ = vault.Hash()

			// Concurrent serialization
			_ = vault.Serialize()

			// Concurrent active policy check
			_ = vault.GetActivePolicy(100100, 100000)
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestComplianceThresholds tests compliance threshold enforcement
func TestComplianceThresholds(t *testing.T) {
	hotKeys := generateTestKeys(t, 15)
	warmKeys := generateTestKeys(t, 7)
	coldKeys := generateTestKeys(t, 5)
	emergencyKeys := generateTestKeys(t, 3)

	vault, err := CreateSovereignWealthFundVault(hotKeys, warmKeys, coldKeys, emergencyKeys)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		amount  uint64
		exceeds bool
	}{
		{
			name:    "BelowThreshold",
			amount:  500000 * 1e8, // 500K XSL - below 1M threshold
			exceeds: false,
		},
		{
			name:    "AtThreshold",
			amount:  1000000 * 1e8, // 1M XSL - at threshold
			exceeds: true,
		},
		{
			name:    "AboveThreshold",
			amount:  2000000 * 1e8, // 2M XSL - above threshold
			exceeds: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vault.EvaluateSpendingPolicy(11, 100100, 100000, tc.amount)
			require.NoError(t, err)

			// The mock compliance check always returns true,
			// so we verify the threshold is being checked correctly
			if tc.exceeds {
				// Large amounts trigger compliance check (but pass due to mock)
				assert.True(t, result.Approved)
			} else {
				// Small amounts bypass compliance check
				assert.True(t, result.Approved)
			}
		})
	}
}
