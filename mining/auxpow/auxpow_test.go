package auxpow

import (
	"crypto/sha256"
	"math/big"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toole-brendan/shell/chaincfg"
)

// TestAuxPoWValidator tests the basic validator functionality
func TestAuxPoWValidator(t *testing.T) {
	config := DefaultAuxPoWConfig()
	params := &chaincfg.MainNetParams // Using a placeholder
	validator := NewAuxPoWValidator(config, params)

	t.Run("InitialState", func(t *testing.T) {
		assert.NotNil(t, validator)
		assert.True(t, config.Enabled)
		assert.False(t, validator.sunsetActivated)
		assert.Equal(t, uint64(0), validator.nativeHashrate)
		assert.Equal(t, uint64(0), validator.totalAuxBlocks)
		assert.Equal(t, uint64(0), validator.totalNativeBlocks)
	})

	t.Run("DefaultConfig", func(t *testing.T) {
		assert.Equal(t, uint32(0x58534C), config.ChainID)
		assert.Equal(t, "XSLTAG", config.CommitmentTag)
		assert.Equal(t, uint64(1000), config.SunsetHashrateThreshold)
		assert.Equal(t, uint32(1008), config.MonitoringBlocks)
		assert.Equal(t, uint32(25920), config.SunsetNoticeBlocks)
	})
}

// TestAuxPoWValidation tests the complete auxiliary proof-of-work validation
func TestAuxPoWValidation(t *testing.T) {
	config := DefaultAuxPoWConfig()
	params := &chaincfg.MainNetParams
	validator := NewAuxPoWValidator(config, params)

	t.Run("ValidAuxPoWBlock", func(t *testing.T) {
		auxBlock := createTestAuxPoWBlock(t, true)

		err := validator.ValidateAuxPoW(auxBlock)
		require.NoError(t, err)
		assert.True(t, auxBlock.IsValid)
		assert.False(t, auxBlock.ValidatedAt.IsZero())
	})

	t.Run("MissingAuxData", func(t *testing.T) {
		auxBlock := &AuxPoWBlock{
			Header:  createTestShellHeader(t),
			AuxData: nil,
		}

		err := validator.ValidateAuxPoW(auxBlock)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing auxiliary proof-of-work data")
	})

	t.Run("DisabledAuxPoW", func(t *testing.T) {
		config.Enabled = false
		auxBlock := createTestAuxPoWBlock(t, true)

		err := validator.ValidateAuxPoW(auxBlock)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "auxiliary proof-of-work is disabled")

		// Re-enable for other tests
		config.Enabled = true
	})

	t.Run("SunsetActivated", func(t *testing.T) {
		validator.sunsetActivated = true
		auxBlock := createTestAuxPoWBlock(t, true)

		err := validator.ValidateAuxPoW(auxBlock)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "auxiliary proof-of-work has been sunset")

		// Reset for other tests
		validator.sunsetActivated = false
	})
}

// TestShellCommitmentValidation tests Shell block hash commitment verification
func TestShellCommitmentValidation(t *testing.T) {
	config := DefaultAuxPoWConfig()
	params := &chaincfg.MainNetParams
	validator := NewAuxPoWValidator(config, params)

	t.Run("ValidCommitment", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)

		err := validator.verifyShellCommitment(auxData)
		require.NoError(t, err)
	})

	t.Run("MissingCommitmentTag", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, false) // No commitment

		err := validator.verifyShellCommitment(auxData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shell commitment tag not found")
	})

	t.Run("HashMismatch", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)

		// Change the shell hash to create mismatch
		auxData.ShellBlockHash = createTestHash(t, "different_hash")

		err := validator.verifyShellCommitment(auxData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "committed shell block hash mismatch")
	})

	t.Run("InvalidCoinbase", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)
		auxData.ParentCoinbase = nil

		err := validator.verifyShellCommitment(auxData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid coinbase transaction")
	})
}

// TestMerkleBranchValidation tests merkle branch verification
func TestMerkleBranchValidation(t *testing.T) {
	config := DefaultAuxPoWConfig()
	params := &chaincfg.MainNetParams
	validator := NewAuxPoWValidator(config, params)

	t.Run("ValidMerkleBranch", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)

		// Create a valid merkle branch
		coinbaseHash := auxData.ParentCoinbase.TxHash()
		auxData.MerkleBranch = []chainhash.Hash{
			createTestHash(t, "merkle_1"),
			createTestHash(t, "merkle_2"),
		}

		// Compute expected merkle root
		expectedRoot := validator.computeMerkleRoot(coinbaseHash, auxData.MerkleBranch, auxData.ParentBlockTxCount)
		auxData.ParentBlock.MerkleRoot = expectedRoot

		err := validator.verifyMerkleBranch(auxData)
		require.NoError(t, err)
	})

	t.Run("InvalidMerkleBranch", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)

		// Use wrong merkle root
		auxData.ParentBlock.MerkleRoot = createTestHash(t, "wrong_merkle_root")

		err := validator.verifyMerkleBranch(auxData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merkle root mismatch")
	})
}

// TestWorkSufficiency tests proof-of-work sufficiency validation
func TestWorkSufficiency(t *testing.T) {
	config := DefaultAuxPoWConfig()
	params := &chaincfg.MainNetParams
	validator := NewAuxPoWValidator(config, params)

	t.Run("SufficientWork", func(t *testing.T) {
		shellHeader := createTestShellHeader(t)
		bitcoinHeader := createTestBitcoinHeader(t)

		// Set Bitcoin difficulty lower than Shell (more work)
		bitcoinHeader.Bits = 0x1d00ffff // Easier target = more work
		shellHeader.Bits = 0x1d00ffff   // Same target for this test

		err := validator.verifyWorkSufficiency(shellHeader, bitcoinHeader)
		require.NoError(t, err)
	})

	t.Run("InsufficientWork", func(t *testing.T) {
		shellHeader := createTestShellHeader(t)
		bitcoinHeader := createTestBitcoinHeader(t)

		// Set Bitcoin difficulty much higher than Shell (less work)
		bitcoinHeader.Bits = 0x1d7fffff // Harder target = less work
		shellHeader.Bits = 0x1d00ffff   // Easier target = requires more work

		err := validator.verifyWorkSufficiency(shellHeader, bitcoinHeader)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient work")
	})
}

// TestAuxPoWParameters tests auxiliary parameter validation
func TestAuxPoWParameters(t *testing.T) {
	config := DefaultAuxPoWConfig()
	params := &chaincfg.MainNetParams
	validator := NewAuxPoWValidator(config, params)

	t.Run("ValidParameters", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)

		err := validator.verifyAuxParams(auxData)
		require.NoError(t, err)
	})

	t.Run("InvalidChainIndex", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)
		auxData.ChainIndex = 1 // Should be 0 for Bitcoin

		err := validator.verifyAuxParams(auxData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid chain index")
	})

	t.Run("MissingParentBlock", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)
		auxData.ParentBlock = nil

		err := validator.verifyAuxParams(auxData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing parent block header")
	})

	t.Run("InvalidBitcoinVersion", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)
		auxData.ParentBlock.Version = 0

		err := validator.verifyAuxParams(auxData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid Bitcoin block version")
	})

	t.Run("InvalidTimestamp", func(t *testing.T) {
		shellHash := createTestHash(t, "shell_block_hash")
		auxData := createTestAuxPoWData(t, shellHash, true)
		auxData.ParentBlock.Timestamp = time.Date(2008, 1, 1, 0, 0, 0, 0, time.UTC) // Before Bitcoin

		err := validator.verifyAuxParams(auxData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Bitcoin block timestamp too early")
	})
}

// TestSunsetMechanism tests the automatic sunset functionality
func TestSunsetMechanism(t *testing.T) {
	config := DefaultAuxPoWConfig()
	config.SunsetHashrateThreshold = 50 // Low threshold for testing
	config.MonitoringBlocks = 10        // Short monitoring period
	config.SunsetNoticeBlocks = 20      // Short notice period

	params := &chaincfg.MainNetParams
	validator := NewAuxPoWValidator(config, params)

	t.Run("HashrateTracking", func(t *testing.T) {
		blockHash := createTestHash(t, "block_1")

		// Add some AuxPoW blocks
		for i := uint32(0); i < 5; i++ {
			validator.UpdateHashrateMetrics(i, true, blockHash)
		}

		// Add some native blocks
		for i := uint32(5); i < 15; i++ {
			validator.UpdateHashrateMetrics(i, false, blockHash)
		}

		stats := validator.GetStatistics()
		assert.Equal(t, uint64(5), stats["aux_blocks"])
		assert.Equal(t, uint64(10), stats["native_blocks"])
		assert.InDelta(t, 0.333, stats["aux_block_ratio"], 0.01)    // 5/15
		assert.InDelta(t, 0.667, stats["native_block_ratio"], 0.01) // 10/15
	})

	t.Run("SunsetNotice", func(t *testing.T) {
		// Reset validator
		validator = NewAuxPoWValidator(config, params)

		// Simulate high native hashrate
		validator.totalNativeBlocks = 80
		validator.totalAuxBlocks = 20

		blockHash := createTestHash(t, "block_sunset")
		validator.UpdateHashrateMetrics(20, false, blockHash)

		// Should trigger sunset notice
		sunsetActivated, noticeHeight, nativeHashrate, _ := validator.GetSunsetStatus()
		assert.False(t, sunsetActivated)             // Not activated yet, just notice
		assert.Greater(t, noticeHeight, uint32(0))   // Notice should be set
		assert.Greater(t, nativeHashrate, uint64(0)) // Hashrate should be calculated
	})

	t.Run("SunsetActivation", func(t *testing.T) {
		// Simulate reaching sunset height
		validator.sunsetNoticeHeight = 15

		blockHash := createTestHash(t, "block_sunset_activation")
		validator.UpdateHashrateMetrics(25, false, blockHash) // Past notice height

		// Should activate sunset
		sunsetActivated, _, _, _ := validator.GetSunsetStatus()
		assert.True(t, sunsetActivated)
		assert.False(t, config.Enabled) // Should disable AuxPoW
	})
}

// TestCommitmentFunctions tests commitment creation and extraction
func TestCommitmentFunctions(t *testing.T) {
	shellHash := createTestHash(t, "test_shell_block")
	tag := "XSLTAG"

	t.Run("CreateCommitment", func(t *testing.T) {
		commitment := CreateShellCommitment(shellHash, tag)

		assert.Equal(t, len(tag)+32, len(commitment))
		assert.Equal(t, []byte(tag), commitment[:len(tag)])
		assert.Equal(t, shellHash[:], commitment[len(tag):])
	})

	t.Run("ExtractCommitment", func(t *testing.T) {
		// Create coinbase script with commitment
		commitment := CreateShellCommitment(shellHash, tag)
		coinbaseScript := append([]byte("some_prefix"), commitment...)
		coinbaseScript = append(coinbaseScript, []byte("some_suffix")...)

		extractedHash, err := ExtractShellCommitment(coinbaseScript, tag)
		require.NoError(t, err)
		assert.Equal(t, shellHash, *extractedHash)
	})

	t.Run("ExtractMissingCommitment", func(t *testing.T) {
		coinbaseScript := []byte("no_commitment_here")

		_, err := ExtractShellCommitment(coinbaseScript, tag)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shell commitment tag not found")
	})

	t.Run("ExtractInsufficientData", func(t *testing.T) {
		// Coinbase with tag but insufficient hash data
		coinbaseScript := []byte(tag + "short")

		_, err := ExtractShellCommitment(coinbaseScript, tag)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient data for shell block hash")
	})
}

// TestMergeMinable tests the merge mining suitability check
func TestMergeMinable(t *testing.T) {
	t.Run("MergeMinableBlock", func(t *testing.T) {
		bitcoinHeader := createTestBitcoinHeader(t)
		bitcoinHeader.Bits = 0x1d00ffff // Easy target

		shellHash := createTestHash(t, "shell_block")
		coinbase := createTestCoinbaseWithCommitment(t, shellHash)

		shellTarget := big.NewInt(0x00000000ffff0000000000000000000000000000000000000000000000000000)

		isMergeable := MergeMinable(bitcoinHeader, coinbase, shellTarget)
		assert.True(t, isMergeable)
	})

	t.Run("InsufficientWork", func(t *testing.T) {
		bitcoinHeader := createTestBitcoinHeader(t)
		bitcoinHeader.Bits = 0x1d7fffff // Very hard target (little work)

		shellHash := createTestHash(t, "shell_block")
		coinbase := createTestCoinbaseWithCommitment(t, shellHash)

		shellTarget := big.NewInt(0x0000ffff) // Easy target (requires lots of work)

		isMergeable := MergeMinable(bitcoinHeader, coinbase, shellTarget)
		assert.False(t, isMergeable)
	})

	t.Run("MissingCommitment", func(t *testing.T) {
		bitcoinHeader := createTestBitcoinHeader(t)
		bitcoinHeader.Bits = 0x1d00ffff

		// Coinbase without Shell commitment
		coinbase := &wire.MsgTx{
			TxIn: []*wire.TxIn{{
				SignatureScript: []byte("no_shell_commitment"),
			}},
		}

		shellTarget := big.NewInt(0x00000000ffff0000000000000000000000000000000000000000000000000000)

		isMergeable := MergeMinable(bitcoinHeader, coinbase, shellTarget)
		assert.False(t, isMergeable)
	})
}

// TestStatistics tests the statistics collection
func TestStatistics(t *testing.T) {
	config := DefaultAuxPoWConfig()
	params := &chaincfg.MainNetParams
	validator := NewAuxPoWValidator(config, params)

	t.Run("EmptyStatistics", func(t *testing.T) {
		stats := validator.GetStatistics()

		assert.Equal(t, uint64(0), stats["aux_blocks"])
		assert.Equal(t, uint64(0), stats["native_blocks"])
		assert.Equal(t, 0.0, stats["aux_block_ratio"])
		assert.Equal(t, 0.0, stats["native_block_ratio"])
		assert.False(t, stats["sunset_activated"].(bool))
	})

	t.Run("WithData", func(t *testing.T) {
		validator.totalAuxBlocks = 30
		validator.totalNativeBlocks = 70
		validator.nativeHashrate = 500
		validator.mergeHashrate = 200

		stats := validator.GetStatistics()

		assert.Equal(t, uint64(30), stats["aux_blocks"])
		assert.Equal(t, uint64(70), stats["native_blocks"])
		assert.Equal(t, 0.3, stats["aux_block_ratio"])
		assert.Equal(t, 0.7, stats["native_block_ratio"])
		assert.Equal(t, uint64(500), stats["native_hashrate_ths"])
		assert.Equal(t, uint64(200), stats["merge_hashrate_ths"])
	})
}

// Helper functions for testing

func createTestHash(t *testing.T, data string) chainhash.Hash {
	hash := sha256.Sum256([]byte(data))
	return chainhash.Hash(hash)
}

func createTestShellHeader(t *testing.T) *wire.BlockHeader {
	return &wire.BlockHeader{
		Version:    1,
		PrevBlock:  createTestHash(t, "prev_shell_block"),
		MerkleRoot: createTestHash(t, "shell_merkle_root"),
		Timestamp:  time.Now(),
		Bits:       0x1d00ffff,
		Nonce:      12345,
	}
}

func createTestBitcoinHeader(t *testing.T) *wire.BlockHeader {
	return &wire.BlockHeader{
		Version:    1,
		PrevBlock:  createTestHash(t, "prev_bitcoin_block"),
		MerkleRoot: createTestHash(t, "bitcoin_merkle_root"),
		Timestamp:  time.Now(),
		Bits:       0x1d00ffff,
		Nonce:      67890,
	}
}

func createTestCoinbaseWithCommitment(t *testing.T, shellHash chainhash.Hash) *wire.MsgTx {
	commitment := CreateShellCommitment(shellHash, "XSLTAG")

	coinbaseScript := make([]byte, 0)
	coinbaseScript = append(coinbaseScript, []byte("coinbase_prefix")...)
	coinbaseScript = append(coinbaseScript, commitment...)
	coinbaseScript = append(coinbaseScript, []byte("coinbase_suffix")...)

	return &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: coinbaseScript,
			Sequence:        0xffffffff,
		}},
		TxOut: []*wire.TxOut{{
			Value:    5000000000, // 50 BTC
			PkScript: []byte{},
		}},
	}
}

func createTestAuxPoWData(t *testing.T, shellHash chainhash.Hash, includeCommitment bool) *AuxPoWData {
	var coinbase *wire.MsgTx

	if includeCommitment {
		coinbase = createTestCoinbaseWithCommitment(t, shellHash)
	} else {
		coinbase = &wire.MsgTx{
			TxIn: []*wire.TxIn{{
				SignatureScript: []byte("no_commitment"),
			}},
		}
	}

	return &AuxPoWData{
		ParentCoinbase:     coinbase,
		MerkleBranch:       []chainhash.Hash{},
		ParentBlockTxCount: 1,
		ParentBlock:        createTestBitcoinHeader(t),
		ChainIndex:         0,
		ShellBlockHash:     shellHash,
	}
}

func createTestAuxPoWBlock(t *testing.T, valid bool) *AuxPoWBlock {
	shellHash := createTestHash(t, "shell_block_for_auxpow")

	return &AuxPoWBlock{
		Header:  createTestShellHeader(t),
		AuxData: createTestAuxPoWData(t, shellHash, valid),
		IsValid: false,
	}
}
