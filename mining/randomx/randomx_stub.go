//go:build !cgo
// +build !cgo

// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package randomx

// randomx package stub - This is a temporary implementation until we integrate
// the actual RandomX library. The real implementation would use CGO bindings
// to the RandomX C++ library.

// Cache represents the RandomX cache
type Cache struct {
	seed []byte
}

// NewCache creates a new RandomX cache with the given seed
func NewCache(seed []byte) (*Cache, error) {
	return &Cache{seed: seed}, nil
}

// Close releases the cache resources
func (c *Cache) Close() {
	// Cleanup would happen here
}

// Dataset represents the RandomX dataset
type Dataset struct {
	cache *Cache
}

// NewDataset creates a new RandomX dataset from a cache
func NewDataset(cache *Cache) (*Dataset, error) {
	return &Dataset{cache: cache}, nil
}

// Close releases the dataset resources
func (d *Dataset) Close() {
	// Cleanup would happen here
}

// VM represents the RandomX virtual machine
type VM struct {
	cache   *Cache
	dataset *Dataset
}

// NewVM creates a new RandomX VM with the given cache and dataset
func NewVM(cache *Cache, dataset *Dataset) (*VM, error) {
	return &VM{cache: cache, dataset: dataset}, nil
}

// CalcHash calculates the RandomX hash of the input
func (vm *VM) CalcHash(input []byte) []byte {
	// This is a stub implementation
	// The real RandomX would perform the actual proof-of-work calculation
	hash := make([]byte, 32)
	// For now, just return a dummy hash
	copy(hash, input)
	return hash
}

// Close releases the VM resources
func (vm *VM) Close() {
	// Cleanup would happen here
}

// GetFlags returns default flags for stub implementation
func GetFlags() Flags {
	return 0 // Default flags
}

// Flags type for compatibility
type Flags int
