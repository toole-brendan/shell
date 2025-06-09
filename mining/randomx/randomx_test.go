//go:build cgo
// +build cgo

package randomx

import (
	"encoding/hex"
	"testing"
)

// Test basic RandomX functionality
func TestRandomXBasic(t *testing.T) {
	seed := []byte("test seed 123")

	// Create cache
	cache, err := NewCache(seed)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Create VM (light mode, no dataset)
	vm, err := NewVM(cache, nil)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}
	defer vm.Close()

	// Calculate hash
	input := []byte("Hello Shell Reserve!")
	hash := vm.CalcHash(input)

	if len(hash) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash))
	}

	// Verify hash is not all zeros
	allZeros := true
	for _, b := range hash {
		if b != 0 {
			allZeros = false
			break
		}
	}

	if allZeros {
		t.Error("Hash should not be all zeros")
	}

	t.Logf("RandomX hash: %s", hex.EncodeToString(hash))
}

// Test that the same input produces the same hash
func TestRandomXDeterministic(t *testing.T) {
	seed := []byte("deterministic test")
	input := []byte("test input")

	// First calculation
	cache1, err := NewCache(seed)
	if err != nil {
		t.Fatal(err)
	}
	defer cache1.Close()

	vm1, err := NewVM(cache1, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm1.Close()

	hash1 := vm1.CalcHash(input)

	// Second calculation with same seed
	cache2, err := NewCache(seed)
	if err != nil {
		t.Fatal(err)
	}
	defer cache2.Close()

	vm2, err := NewVM(cache2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm2.Close()

	hash2 := vm2.CalcHash(input)

	// Should be identical
	if hex.EncodeToString(hash1) != hex.EncodeToString(hash2) {
		t.Errorf("Hashes should be identical:\nHash1: %s\nHash2: %s",
			hex.EncodeToString(hash1), hex.EncodeToString(hash2))
	}

	t.Logf("Deterministic hash: %s", hex.EncodeToString(hash1))
}

// Test different inputs produce different hashes
func TestRandomXDifferentInputs(t *testing.T) {
	seed := []byte("test seed")

	cache, err := NewCache(seed)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	vm, err := NewVM(cache, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	hash1 := vm.CalcHash([]byte("input1"))
	hash2 := vm.CalcHash([]byte("input2"))

	if hex.EncodeToString(hash1) == hex.EncodeToString(hash2) {
		t.Error("Different inputs should produce different hashes")
	}

	t.Logf("Hash1: %s", hex.EncodeToString(hash1))
	t.Logf("Hash2: %s", hex.EncodeToString(hash2))
}

// Test detection functionality
func TestDetection(t *testing.T) {
	if !IsRealImplementation() {
		t.Error("Expected real RandomX implementation with CGO enabled")
	}

	info := GetImplementationInfo()
	t.Logf("Implementation: %s", info)

	if info == "RandomX Stub (development only)" {
		t.Error("Should not be using stub implementation with CGO")
	}
}
