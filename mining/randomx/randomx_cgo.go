//go:build cgo
// +build cgo

package randomx

/*
#cgo CFLAGS: -I../../third_party/randomx/src
#cgo LDFLAGS: -L../../third_party/randomx/build -lrandomx -lstdc++ -lm
#cgo darwin LDFLAGS: -framework IOKit

#include "randomx_wrapper.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"runtime"
	"sync"
	"unsafe"
)

// Flags for RandomX configuration
type Flags int

const (
	FlagDefault     Flags = C.RANDOMX_FLAG_DEFAULT
	FlagLargePages  Flags = C.RANDOMX_FLAG_LARGE_PAGES
	FlagHardAES     Flags = C.RANDOMX_FLAG_HARD_AES
	FlagFullMem     Flags = C.RANDOMX_FLAG_FULL_MEM
	FlagJIT         Flags = C.RANDOMX_FLAG_JIT
	FlagSecure      Flags = C.RANDOMX_FLAG_SECURE
	FlagArgon2SSSE3 Flags = C.RANDOMX_FLAG_ARGON2_SSSE3
	FlagArgon2AVX2  Flags = C.RANDOMX_FLAG_ARGON2_AVX2
)

// RealCache implements the RandomX cache using CGO
type RealCache struct {
	ptr  *C.randomx_cache
	mu   sync.Mutex
	seed []byte
}

// NewCache creates a new RandomX cache with the given seed
func NewCache(seed []byte) (*Cache, error) {
	if len(seed) == 0 {
		return nil, errors.New("seed cannot be empty")
	}

	flags := GetFlags()
	cachePtr := C.randomx_alloc_cache(C.randomx_flags(flags))
	if cachePtr == nil {
		return nil, errors.New("failed to allocate RandomX cache")
	}

	// Initialize cache with seed
	seedPtr := C.CBytes(seed)
	defer C.free(seedPtr)
	C.randomx_init_cache(cachePtr, seedPtr, C.size_t(len(seed)))

	realCache := &RealCache{
		ptr:  cachePtr,
		seed: append([]byte(nil), seed...), // Copy seed
	}

	// Set finalizer to ensure cleanup
	runtime.SetFinalizer(realCache, (*RealCache).finalize)

	// Return wrapped cache
	return &Cache{impl: realCache}, nil
}

func (c *RealCache) finalize() {
	if c.ptr != nil {
		C.randomx_release_cache(c.ptr)
		c.ptr = nil
	}
}

// RealDataset implements the RandomX dataset using CGO
type RealDataset struct {
	ptr *C.randomx_dataset
	mu  sync.Mutex
}

// NewDataset creates a new RandomX dataset from a cache
func NewDataset(cache *Cache) (*Dataset, error) {
	if cache == nil || cache.impl == nil {
		return nil, errors.New("cache cannot be nil")
	}

	realCache := cache.impl.(*RealCache)
	flags := GetFlags() | FlagFullMem

	datasetPtr := C.randomx_alloc_dataset(C.randomx_flags(flags))
	if datasetPtr == nil {
		return nil, errors.New("failed to allocate RandomX dataset")
	}

	// Initialize dataset (this is memory-intensive and takes time)
	itemCount := C.randomx_dataset_item_count()
	C.randomx_init_dataset(datasetPtr, realCache.ptr, 0, itemCount)

	realDataset := &RealDataset{
		ptr: datasetPtr,
	}

	runtime.SetFinalizer(realDataset, (*RealDataset).finalize)

	return &Dataset{impl: realDataset}, nil
}

func (d *RealDataset) finalize() {
	if d.ptr != nil {
		C.randomx_release_dataset(d.ptr)
		d.ptr = nil
	}
}

// RealVM implements the RandomX virtual machine using CGO
type RealVM struct {
	ptr     *C.randomx_vm
	cache   *RealCache
	dataset *RealDataset
	mu      sync.Mutex
}

// NewVM creates a new RandomX VM with the given cache and dataset
func NewVM(cache *Cache, dataset *Dataset) (*VM, error) {
	if cache == nil || cache.impl == nil {
		return nil, errors.New("cache cannot be nil")
	}

	realCache := cache.impl.(*RealCache)
	var realDataset *RealDataset
	var datasetPtr *C.randomx_dataset

	if dataset != nil && dataset.impl != nil {
		realDataset = dataset.impl.(*RealDataset)
		datasetPtr = realDataset.ptr
	}

	flags := GetFlags()
	if datasetPtr != nil {
		flags |= FlagFullMem
	}

	vmPtr := C.randomx_create_vm(C.randomx_flags(flags), realCache.ptr, datasetPtr)
	if vmPtr == nil {
		return nil, errors.New("failed to create RandomX VM")
	}

	realVM := &RealVM{
		ptr:     vmPtr,
		cache:   realCache,
		dataset: realDataset,
	}

	runtime.SetFinalizer(realVM, (*RealVM).finalize)

	return &VM{impl: realVM}, nil
}

// CalcHash calculates the RandomX hash of the input
func (vm *VM) CalcHash(input []byte) []byte {
	if vm == nil || vm.impl == nil {
		return nil
	}

	realVM := vm.impl.(*RealVM)
	realVM.mu.Lock()
	defer realVM.mu.Unlock()

	if len(input) == 0 {
		return nil
	}

	output := make([]byte, 32) // RandomX produces 32-byte hashes
	inputPtr := C.CBytes(input)
	defer C.free(inputPtr)

	C.randomx_calculate_hash(realVM.ptr, inputPtr, C.size_t(len(input)),
		unsafe.Pointer(&output[0]))

	return output
}

func (vm *RealVM) finalize() {
	if vm.ptr != nil {
		C.randomx_destroy_vm(vm.ptr)
		vm.ptr = nil
	}
}

// GetFlags returns the recommended flags for the current CPU
func GetFlags() Flags {
	return Flags(C.randomx_get_flags())
}

// Wrapper types to maintain API compatibility
type Cache struct {
	impl interface{}
}

type Dataset struct {
	impl interface{}
}

type VM struct {
	impl interface{}
}

func (c *Cache) Close() {
	if c.impl != nil {
		if realCache, ok := c.impl.(*RealCache); ok {
			realCache.finalize()
		}
	}
}

func (d *Dataset) Close() {
	if d.impl != nil {
		if realDataset, ok := d.impl.(*RealDataset); ok {
			realDataset.finalize()
		}
	}
}

func (vm *VM) Close() {
	if vm.impl != nil {
		if realVM, ok := vm.impl.(*RealVM); ok {
			realVM.finalize()
		}
	}
}
