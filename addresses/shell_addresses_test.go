// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package addresses

import (
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/toole-brendan/shell/chaincfg"
)

func TestShellTaprootAddress(t *testing.T) {
	// Create a test private key
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create private key: %v", err)
	}
	pubKey := privKey.PubKey()

	// Use Shell mainnet parameters from our chaincfg
	params := &chaincfg.MainNetParams

	t.Run("CreateTaprootAddress", func(t *testing.T) {
		addr, err := NewShellTaprootAddress(pubKey, params)
		if err != nil {
			t.Fatalf("Failed to create Taproot address: %v", err)
		}

		if addr == nil {
			t.Fatal("Address should not be nil")
		}

		// Check address string format
		addrStr := addr.String()
		if !strings.HasPrefix(addrStr, "xsl1") {
			t.Errorf("Taproot address should start with 'xsl1', got: %s", addrStr)
		}

		t.Logf("Generated Taproot address: %s", addrStr)
	})

	t.Run("TaprootAddressType", func(t *testing.T) {
		addr, err := NewShellTaprootAddress(pubKey, params)
		if err != nil {
			t.Fatalf("Failed to create Taproot address: %v", err)
		}

		if addr.AddressType() != AddressTypeTaproot {
			t.Errorf("Expected address type %s, got %s", AddressTypeTaproot, addr.AddressType())
		}
	})

	t.Run("TaprootNetworkCheck", func(t *testing.T) {
		addr, err := NewShellTaprootAddress(pubKey, params)
		if err != nil {
			t.Fatalf("Failed to create Taproot address: %v", err)
		}

		if !addr.IsForNetwork(params) {
			t.Error("Address should be for mainnet")
		}
	})

	t.Run("InvalidPublicKey", func(t *testing.T) {
		_, err := NewShellTaprootAddress(nil, params)
		if err != ErrInvalidPublicKey {
			t.Errorf("Expected ErrInvalidPublicKey, got %v", err)
		}
	})
}

func TestShellP2PKHAddress(t *testing.T) {
	// Create a test private key
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create private key: %v", err)
	}
	pubKey := privKey.PubKey()

	params := &chaincfg.MainNetParams

	t.Run("CreateP2PKHAddress", func(t *testing.T) {
		addr, err := GenerateShellAddress(pubKey, AddressTypeP2PKH, params)
		if err != nil {
			t.Fatalf("Failed to create P2PKH address: %v", err)
		}

		if addr == nil {
			t.Fatal("Address should not be nil")
		}

		// Check address type
		if addr.AddressType() != AddressTypeP2PKH {
			t.Errorf("Expected address type %s, got %s", AddressTypeP2PKH, addr.AddressType())
		}

		t.Logf("Generated P2PKH address: %s", addr.String())
	})

	t.Run("P2PKHFromHash", func(t *testing.T) {
		// Create a 20-byte hash
		hash := make([]byte, 20)
		for i := range hash {
			hash[i] = byte(i)
		}

		addr, err := NewShellP2PKHAddress(hash, params)
		if err != nil {
			t.Fatalf("Failed to create P2PKH address from hash: %v", err)
		}

		// Check script address matches input hash
		scriptAddr := addr.ScriptAddress()
		if len(scriptAddr) != 20 {
			t.Errorf("Expected script address length 20, got %d", len(scriptAddr))
		}

		for i, b := range scriptAddr {
			if b != byte(i) {
				t.Errorf("Script address mismatch at index %d: expected %d, got %d", i, i, b)
			}
		}
	})

	t.Run("InvalidHashLength", func(t *testing.T) {
		invalidHash := make([]byte, 19) // Wrong length
		_, err := NewShellP2PKHAddress(invalidHash, params)
		if err == nil {
			t.Error("Expected error for invalid hash length")
		}
	})
}

func TestGenerateShellAddress(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create private key: %v", err)
	}
	pubKey := privKey.PubKey()
	params := &chaincfg.MainNetParams

	t.Run("GenerateTaproot", func(t *testing.T) {
		addr, err := GenerateShellAddress(pubKey, AddressTypeTaproot, params)
		if err != nil {
			t.Fatalf("Failed to generate Taproot address: %v", err)
		}

		if addr.AddressType() != AddressTypeTaproot {
			t.Errorf("Expected Taproot address type, got %s", addr.AddressType())
		}

		addrStr := addr.String()
		if !strings.HasPrefix(addrStr, "xsl1") {
			t.Errorf("Taproot address should start with 'xsl1', got: %s", addrStr)
		}
	})

	t.Run("GenerateP2PKH", func(t *testing.T) {
		addr, err := GenerateShellAddress(pubKey, AddressTypeP2PKH, params)
		if err != nil {
			t.Fatalf("Failed to generate P2PKH address: %v", err)
		}

		if addr.AddressType() != AddressTypeP2PKH {
			t.Errorf("Expected P2PKH address type, got %s", addr.AddressType())
		}
	})

	t.Run("UnsupportedType", func(t *testing.T) {
		_, err := GenerateShellAddress(pubKey, "invalid_type", params)
		if err != ErrUnsupportedAddressType {
			t.Errorf("Expected ErrUnsupportedAddressType, got %v", err)
		}
	})
}

func TestParseShellAddress(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create private key: %v", err)
	}
	pubKey := privKey.PubKey()
	params := &chaincfg.MainNetParams

	t.Run("ParseTaprootAddress", func(t *testing.T) {
		// Generate a Taproot address
		origAddr, err := GenerateShellAddress(pubKey, AddressTypeTaproot, params)
		if err != nil {
			t.Fatalf("Failed to generate Taproot address: %v", err)
		}

		// Parse it back
		parsedAddr, err := ParseShellAddress(origAddr.String(), params)
		if err != nil {
			t.Fatalf("Failed to parse Taproot address: %v", err)
		}

		// Check they match
		if parsedAddr.String() != origAddr.String() {
			t.Errorf("Parsed address doesn't match original: %s != %s",
				parsedAddr.String(), origAddr.String())
		}

		if parsedAddr.AddressType() != AddressTypeTaproot {
			t.Errorf("Expected Taproot address type, got %s", parsedAddr.AddressType())
		}
	})

	t.Run("ParseP2PKHAddress", func(t *testing.T) {
		// Generate a P2PKH address
		origAddr, err := GenerateShellAddress(pubKey, AddressTypeP2PKH, params)
		if err != nil {
			t.Fatalf("Failed to generate P2PKH address: %v", err)
		}

		// Parse it back
		parsedAddr, err := ParseShellAddress(origAddr.String(), params)
		if err != nil {
			t.Fatalf("Failed to parse P2PKH address: %v", err)
		}

		// Check they match
		if parsedAddr.String() != origAddr.String() {
			t.Errorf("Parsed address doesn't match original: %s != %s",
				parsedAddr.String(), origAddr.String())
		}

		if parsedAddr.AddressType() != AddressTypeP2PKH {
			t.Errorf("Expected P2PKH address type, got %s", parsedAddr.AddressType())
		}
	})

	t.Run("InvalidAddress", func(t *testing.T) {
		_, err := ParseShellAddress("invalid_address", params)
		if err != ErrInvalidAddress {
			t.Errorf("Expected ErrInvalidAddress, got %v", err)
		}
	})
}

func TestCreateShellScript(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create private key: %v", err)
	}
	pubKey := privKey.PubKey()
	params := &chaincfg.MainNetParams

	t.Run("TaprootScript", func(t *testing.T) {
		addr, err := GenerateShellAddress(pubKey, AddressTypeTaproot, params)
		if err != nil {
			t.Fatalf("Failed to generate Taproot address: %v", err)
		}

		script, err := CreateShellScript(addr)
		if err != nil {
			t.Fatalf("Failed to create Taproot script: %v", err)
		}

		if len(script) != 34 { // OP_1 + 32 bytes
			t.Errorf("Expected Taproot script length 34, got %d", len(script))
		}

		// Check OP_1
		if script[0] != 0x51 {
			t.Errorf("Expected OP_1 (0x51), got 0x%02x", script[0])
		}

		t.Logf("Taproot script: %x", script)
	})

	t.Run("P2PKHScript", func(t *testing.T) {
		addr, err := GenerateShellAddress(pubKey, AddressTypeP2PKH, params)
		if err != nil {
			t.Fatalf("Failed to generate P2PKH address: %v", err)
		}

		script, err := CreateShellScript(addr)
		if err != nil {
			t.Fatalf("Failed to create P2PKH script: %v", err)
		}

		if len(script) != 25 { // Standard P2PKH script length
			t.Errorf("Expected P2PKH script length 25, got %d", len(script))
		}

		t.Logf("P2PKH script: %x", script)
	})
}

func TestValidateShellAddress(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create private key: %v", err)
	}
	pubKey := privKey.PubKey()
	params := &chaincfg.MainNetParams

	t.Run("ValidAddress", func(t *testing.T) {
		addr, err := GenerateShellAddress(pubKey, AddressTypeTaproot, params)
		if err != nil {
			t.Fatalf("Failed to generate address: %v", err)
		}

		err = ValidateShellAddress(addr.String(), params)
		if err != nil {
			t.Errorf("Valid address should pass validation: %v", err)
		}
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		err := ValidateShellAddress("invalid_format", params)
		if err == nil {
			t.Error("Invalid address format should fail validation")
		}
	})
}

func TestIsValidShellAddressFormat(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to create private key: %v", err)
	}
	pubKey := privKey.PubKey()
	params := &chaincfg.MainNetParams

	t.Run("ValidFormats", func(t *testing.T) {
		// Test Taproot address format
		taprootAddr, err := GenerateShellAddress(pubKey, AddressTypeTaproot, params)
		if err != nil {
			t.Fatalf("Failed to generate Taproot address: %v", err)
		}

		if !IsValidShellAddressFormat(taprootAddr.String()) {
			t.Error("Valid Taproot address should pass format check")
		}

		// Test P2PKH address format
		p2pkhAddr, err := GenerateShellAddress(pubKey, AddressTypeP2PKH, params)
		if err != nil {
			t.Fatalf("Failed to generate P2PKH address: %v", err)
		}

		if !IsValidShellAddressFormat(p2pkhAddr.String()) {
			t.Error("Valid P2PKH address should pass format check")
		}
	})

	t.Run("InvalidFormats", func(t *testing.T) {
		invalidAddresses := []string{
			"invalid",
			"xsl1invalid",
			"1234567890",
			"bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4",
		}

		for _, addr := range invalidAddresses {
			if IsValidShellAddressFormat(addr) {
				t.Errorf("Invalid address should fail format check: %s", addr)
			}
		}
	})
}

func TestGenerateMultiSigAddress(t *testing.T) {
	// Generate test keys
	var pubKeys []*btcec.PublicKey
	for i := 0; i < 3; i++ {
		privKey, err := btcec.NewPrivateKey()
		if err != nil {
			t.Fatalf("Failed to create private key %d: %v", i, err)
		}
		pubKeys = append(pubKeys, privKey.PubKey())
	}

	params := &chaincfg.MainNetParams

	t.Run("Valid2of3", func(t *testing.T) {
		addr, err := GenerateMultiSigAddress(pubKeys, 2, params)
		if err != nil {
			t.Fatalf("Failed to generate 2-of-3 multisig address: %v", err)
		}

		if addr == nil {
			t.Fatal("Multisig address should not be nil")
		}

		t.Logf("Generated 2-of-3 multisig address: %s", addr.String())
	})

	t.Run("InvalidRequired", func(t *testing.T) {
		_, err := GenerateMultiSigAddress(pubKeys, 0, params)
		if err == nil {
			t.Error("Expected error for required=0")
		}

		_, err = GenerateMultiSigAddress(pubKeys, 4, params)
		if err == nil {
			t.Error("Expected error for required > pubKeys")
		}
	})

	t.Run("TooManyKeys", func(t *testing.T) {
		var manyKeys []*btcec.PublicKey
		for i := 0; i < 16; i++ {
			privKey, _ := btcec.NewPrivateKey()
			manyKeys = append(manyKeys, privKey.PubKey())
		}

		_, err := GenerateMultiSigAddress(manyKeys, 8, params)
		if err == nil {
			t.Error("Expected error for too many public keys")
		}
	})
}
