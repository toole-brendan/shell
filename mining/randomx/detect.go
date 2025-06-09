package randomx

import (
	"fmt"
	"runtime"
)

// IsRealImplementation returns true if the real RandomX implementation is available
func IsRealImplementation() bool {
	// Check if we're compiled with CGO support
	// We can't directly detect CGO, but we can check if our real types exist
	cache, err := NewCache([]byte("test"))
	if err != nil {
		return false
	}
	defer cache.Close()

	// If this is the stub, CalcHash will return a simple hash
	// If this is real RandomX, it will be more complex
	vm, err := NewVM(cache, nil)
	if err != nil {
		return false
	}
	defer vm.Close()

	hash := vm.CalcHash([]byte("test"))

	// Stub returns input as hash (simplified check)
	// Real RandomX produces different hash
	if len(hash) != 32 {
		return false
	}

	// Simple heuristic: if first 4 bytes match input, likely stub
	input := []byte("test")
	if len(hash) >= len(input) {
		match := true
		for i := 0; i < len(input); i++ {
			if hash[i] != input[i] {
				match = false
				break
			}
		}
		if match {
			return false // Probably stub
		}
	}

	return true
}

// GetImplementationInfo returns information about the RandomX implementation
func GetImplementationInfo() string {
	if IsRealImplementation() {
		flags := GetFlags()
		return fmt.Sprintf("RandomX C++ v1.2.1 (flags: 0x%x, arch: %s)", flags, runtime.GOARCH)
	}
	return "RandomX Stub (development only)"
}
