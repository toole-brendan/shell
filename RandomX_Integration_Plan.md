# RandomX C++ Library Integration Plan for Shell Reserve

## Overview

This document outlines the complete implementation plan for replacing the current RandomX stub with the actual RandomX C++ library using CGO bindings. RandomX is a proof-of-work algorithm optimized for general-purpose CPUs, using random code execution and memory-hard techniques.

## Current State

- **Location**: `/mining/randomx/randomx_stub.go`
- **Status**: Stub implementation with placeholder functions
- **Dependencies**: None (pure Go)
- **Functionality**: Returns dummy hashes, no actual PoW computation

## Target State

- **Location**: `/mining/randomx/` directory with CGO bindings
- **Status**: Full RandomX implementation via C++ library
- **Dependencies**: RandomX C++ library, CGO
- **Functionality**: Complete RandomX proof-of-work with 2GB dataset

## Implementation Steps

### Phase 1: Environment Setup (Week 1)

#### 1.1 RandomX C++ Library Integration

```bash
# Add RandomX as git submodule
cd /Users/brendantoole/projects2/shell
git submodule add https://github.com/tevador/RandomX.git third_party/randomx
cd third_party/randomx
git checkout v1.2.1  # Use stable release
```

#### 1.2 Build Dependencies

**macOS (Darwin)**:
```bash
# Install build tools
brew install cmake
brew install boost

# Build RandomX static library
cd third_party/randomx
mkdir build && cd build
cmake .. -DARCH=native -DBUILD_SHARED_LIBS=OFF
make -j$(sysctl -n hw.ncpu)
```

**Linux**:
```bash
# Install dependencies
sudo apt-get update
sudo apt-get install -y cmake g++ libboost-all-dev

# Build RandomX
cd third_party/randomx
mkdir build && cd build
cmake .. -DARCH=native -DBUILD_SHARED_LIBS=OFF
make -j$(nproc)
```

### Phase 2: CGO Bindings Implementation (Week 2)

#### 2.1 Create C Wrapper

Create `mining/randomx/randomx_wrapper.h`:
```c
#ifndef RANDOMX_WRAPPER_H
#define RANDOMX_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stdbool.h>

// Opaque types
typedef struct randomx_cache randomx_cache;
typedef struct randomx_dataset randomx_dataset;
typedef struct randomx_vm randomx_vm;

// Flags for RandomX
typedef enum {
    RANDOMX_FLAG_DEFAULT = 0,
    RANDOMX_FLAG_LARGE_PAGES = 1,
    RANDOMX_FLAG_HARD_AES = 2,
    RANDOMX_FLAG_FULL_MEM = 4,
    RANDOMX_FLAG_JIT = 8,
    RANDOMX_FLAG_SECURE = 16,
    RANDOMX_FLAG_ARGON2_SSSE3 = 32,
    RANDOMX_FLAG_ARGON2_AVX2 = 64,
    RANDOMX_FLAG_ARGON2 = 96
} randomx_flags;

// Cache functions
randomx_cache* randomx_alloc_cache(randomx_flags flags);
void randomx_init_cache(randomx_cache* cache, const void* key, size_t keySize);
void randomx_release_cache(randomx_cache* cache);

// Dataset functions
randomx_dataset* randomx_alloc_dataset(randomx_flags flags);
uint32_t randomx_dataset_item_count(void);
void randomx_init_dataset(randomx_dataset* dataset, randomx_cache* cache, 
                         uint32_t startItem, uint32_t itemCount);
void randomx_release_dataset(randomx_dataset* dataset);

// VM functions
randomx_vm* randomx_create_vm(randomx_flags flags, randomx_cache* cache, 
                             randomx_dataset* dataset);
void randomx_vm_set_cache(randomx_vm* vm, randomx_cache* cache);
void randomx_vm_set_dataset(randomx_vm* vm, randomx_dataset* dataset);
void randomx_destroy_vm(randomx_vm* vm);

// Hash calculation
void randomx_calculate_hash(randomx_vm* vm, const void* input, size_t inputSize, 
                           void* output);
void randomx_calculate_hash_first(randomx_vm* vm, const void* input, size_t inputSize);
void randomx_calculate_hash_next(randomx_vm* vm, const void* nextInput, 
                                size_t nextInputSize, void* output);

// Utility functions
randomx_flags randomx_get_flags(void);

#ifdef __cplusplus
}
#endif

#endif // RANDOMX_WRAPPER_H
```

Create `mining/randomx/randomx_wrapper.cpp`:
```cpp
#include "randomx_wrapper.h"
#include "randomx.h"
#include <cstring>

extern "C" {

randomx_cache* randomx_alloc_cache(randomx_flags flags) {
    return randomx_alloc_cache(static_cast<randomx_flags>(flags));
}

void randomx_init_cache(randomx_cache* cache, const void* key, size_t keySize) {
    randomx_init_cache(cache, key, keySize);
}

void randomx_release_cache(randomx_cache* cache) {
    randomx_release_cache(cache);
}

randomx_dataset* randomx_alloc_dataset(randomx_flags flags) {
    return randomx_alloc_dataset(static_cast<randomx_flags>(flags));
}

uint32_t randomx_dataset_item_count(void) {
    return randomx_dataset_item_count();
}

void randomx_init_dataset(randomx_dataset* dataset, randomx_cache* cache, 
                         uint32_t startItem, uint32_t itemCount) {
    randomx_init_dataset(dataset, cache, startItem, itemCount);
}

void randomx_release_dataset(randomx_dataset* dataset) {
    randomx_release_dataset(dataset);
}

randomx_vm* randomx_create_vm(randomx_flags flags, randomx_cache* cache, 
                             randomx_dataset* dataset) {
    return randomx_create_vm(static_cast<randomx_flags>(flags), cache, dataset);
}

void randomx_vm_set_cache(randomx_vm* vm, randomx_cache* cache) {
    randomx_vm_set_cache(vm, cache);
}

void randomx_vm_set_dataset(randomx_vm* vm, randomx_dataset* dataset) {
    randomx_vm_set_dataset(vm, dataset);
}

void randomx_destroy_vm(randomx_vm* vm) {
    randomx_destroy_vm(vm);
}

void randomx_calculate_hash(randomx_vm* vm, const void* input, size_t inputSize, 
                           void* output) {
    randomx_calculate_hash(vm, input, inputSize, output);
}

void randomx_calculate_hash_first(randomx_vm* vm, const void* input, size_t inputSize) {
    randomx_calculate_hash_first(vm, input, inputSize);
}

void randomx_calculate_hash_next(randomx_vm* vm, const void* nextInput, 
                                size_t nextInputSize, void* output) {
    randomx_calculate_hash_next(vm, nextInput, nextInputSize, output);
}

randomx_flags randomx_get_flags(void) {
    return randomx_get_flags();
}

} // extern "C"
```

#### 2.2 Create Go Bindings

Create `mining/randomx/randomx_cgo.go`:
```go
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
    FlagDefault      Flags = C.RANDOMX_FLAG_DEFAULT
    FlagLargePages   Flags = C.RANDOMX_FLAG_LARGE_PAGES
    FlagHardAES      Flags = C.RANDOMX_FLAG_HARD_AES
    FlagFullMem      Flags = C.RANDOMX_FLAG_FULL_MEM
    FlagJIT          Flags = C.RANDOMX_FLAG_JIT
    FlagSecure       Flags = C.RANDOMX_FLAG_SECURE
    FlagArgon2SSSE3  Flags = C.RANDOMX_FLAG_ARGON2_SSSE3
    FlagArgon2AVX2   Flags = C.RANDOMX_FLAG_ARGON2_AVX2
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
```

### Phase 3: Build System Integration (Week 3)

#### 3.1 Update Build Configuration

Create `mining/randomx/build.sh`:
```bash
#!/bin/bash
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$SCRIPT_DIR/../.."
RANDOMX_DIR="$PROJECT_ROOT/third_party/randomx"

echo "Building RandomX C++ library..."

# Check if RandomX submodule exists
if [ ! -d "$RANDOMX_DIR" ]; then
    echo "RandomX submodule not found. Initializing..."
    cd "$PROJECT_ROOT"
    git submodule add https://github.com/tevador/RandomX.git third_party/randomx
    cd "$RANDOMX_DIR"
    git checkout v1.2.1
fi

# Build RandomX
cd "$RANDOMX_DIR"
if [ ! -d "build" ]; then
    mkdir build
fi

cd build

# Configure based on OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    cmake .. -DARCH=native -DBUILD_SHARED_LIBS=OFF -DCMAKE_C_COMPILER=clang -DCMAKE_CXX_COMPILER=clang++
else
    cmake .. -DARCH=native -DBUILD_SHARED_LIBS=OFF
fi

# Build with available cores
if [[ "$OSTYPE" == "darwin"* ]]; then
    make -j$(sysctl -n hw.ncpu)
else
    make -j$(nproc)
fi

echo "RandomX build complete!"
```

#### 3.2 Create Makefile

Create `mining/randomx/Makefile`:
```makefile
# RandomX CGO build configuration
RANDOMX_DIR := ../../third_party/randomx
RANDOMX_LIB := $(RANDOMX_DIR)/build/librandomx.a

.PHONY: all clean build-deps test

all: build-deps
	go build -tags cgo .

build-deps: $(RANDOMX_LIB)

$(RANDOMX_LIB):
	./build.sh

clean:
	rm -rf $(RANDOMX_DIR)/build
	go clean

test: build-deps
	go test -tags cgo -v .
```

### Phase 4: Testing & Validation (Week 4)

#### 4.1 Create Comprehensive Tests

Create `mining/randomx/randomx_test.go`:
```go
// +build cgo

package randomx

import (
    "bytes"
    "encoding/hex"
    "testing"
    "time"
)

// Test vectors from RandomX specification
var testVectors = []struct {
    seed     string
    input    string
    expected string
}{
    {
        seed:     "test key 000",
        input:    "This is a test",
        expected: "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f",
    },
    {
        seed:     "test key 001",
        input:    "Lorem ipsum dolor sit amet",
        expected: "300a0adb47603dedb42228ccb2b211104f4da45af709cd7547cd049e9489c969",
    },
}

func TestRandomXHash(t *testing.T) {
    for i, tv := range testVectors {
        t.Run(fmt.Sprintf("TestVector%d", i), func(t *testing.T) {
            // Create cache
            cache, err := NewCache([]byte(tv.seed))
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
            hash := vm.CalcHash([]byte(tv.input))
            hashHex := hex.EncodeToString(hash)

            if hashHex != tv.expected {
                t.Errorf("Hash mismatch:\nGot:      %s\nExpected: %s", hashHex, tv.expected)
            }
        })
    }
}

func TestRandomXDataset(t *testing.T) {
    // This test is memory-intensive (2GB+)
    if testing.Short() {
        t.Skip("Skipping dataset test in short mode")
    }

    seed := []byte("Shell Reserve RandomX Test")
    
    // Create cache
    cache, err := NewCache(seed)
    if err != nil {
        t.Fatalf("Failed to create cache: %v", err)
    }
    defer cache.Close()

    // Create dataset (this allocates 2GB+)
    t.Log("Creating dataset (this may take 1-2 minutes)...")
    start := time.Now()
    dataset, err := NewDataset(cache)
    if err != nil {
        t.Fatalf("Failed to create dataset: %v", err)
    }
    defer dataset.Close()
    t.Logf("Dataset created in %v", time.Since(start))

    // Create VM with dataset (fast mode)
    vm, err := NewVM(cache, dataset)
    if err != nil {
        t.Fatalf("Failed to create VM: %v", err)
    }
    defer vm.Close()

    // Benchmark hash calculation
    input := []byte("Benchmark input for Shell Reserve")
    iterations := 1000
    
    start = time.Now()
    for i := 0; i < iterations; i++ {
        _ = vm.CalcHash(input)
    }
    elapsed := time.Since(start)
    
    hashesPerSecond := float64(iterations) / elapsed.Seconds()
    t.Logf("Performance: %.2f hashes/second", hashesPerSecond)
}

func TestConcurrentHashing(t *testing.T) {
    seed := []byte("Concurrent test seed")
    
    cache, err := NewCache(seed)
    if err != nil {
        t.Fatalf("Failed to create cache: %v", err)
    }
    defer cache.Close()

    // Create multiple VMs for concurrent hashing
    numWorkers := 4
    done := make(chan bool, numWorkers)
    
    for i := 0; i < numWorkers; i++ {
        go func(workerID int) {
            vm, err := NewVM(cache, nil)
            if err != nil {
                t.Errorf("Worker %d: Failed to create VM: %v", workerID, err)
                done <- false
                return
            }
            defer vm.Close()

            // Each worker computes different hashes
            for j := 0; j < 100; j++ {
                input := []byte(fmt.Sprintf("Worker %d iteration %d", workerID, j))
                hash := vm.CalcHash(input)
                if len(hash) != 32 {
                    t.Errorf("Worker %d: Invalid hash length: %d", workerID, len(hash))
                    done <- false
                    return
                }
            }
            
            done <- true
        }(i)
    }

    // Wait for all workers
    for i := 0; i < numWorkers; i++ {
        if !<-done {
            t.Fatal("Concurrent hashing failed")
        }
    }
}
```

#### 4.2 Create Benchmarks

Create `mining/randomx/randomx_bench_test.go`:
```go
// +build cgo

package randomx

import (
    "testing"
)

func BenchmarkRandomXLight(b *testing.B) {
    cache, err := NewCache([]byte("benchmark seed"))
    if err != nil {
        b.Fatal(err)
    }
    defer cache.Close()

    vm, err := NewVM(cache, nil)
    if err != nil {
        b.Fatal(err)
    }
    defer vm.Close()

    input := []byte("benchmark input")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = vm.CalcHash(input)
    }
}

func BenchmarkRandomXFull(b *testing.B) {
    if testing.Short() {
        b.Skip("Skipping full dataset benchmark in short mode")
    }

    cache, err := NewCache([]byte("benchmark seed"))
    if err != nil {
        b.Fatal(err)
    }
    defer cache.Close()

    dataset, err := NewDataset(cache)
    if err != nil {
        b.Fatal(err)
    }
    defer dataset.Close()

    vm, err := NewVM(cache, dataset)
    if err != nil {
        b.Fatal(err)
    }
    defer vm.Close()

    input := []byte("benchmark input")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = vm.CalcHash(input)
    }
}
```

### Phase 5: Integration with Mining Code (Week 5)

#### 5.1 Update Miner Integration

Modify `mining/randomx/miner.go` to detect CGO availability:
```go
// +build !cgo

package randomx

// This file is compiled when CGO is not available
// It maintains the stub implementation for non-CGO builds
```

Create `mining/randomx/detect.go`:
```go
package randomx

import "runtime"

// IsRealImplementation returns true if the real RandomX implementation is available
func IsRealImplementation() bool {
    // Check if we have CGO support
    return runtime.Compiler != "gccgo" && runtime.GOOS != "js"
}

// GetImplementationInfo returns information about the RandomX implementation
func GetImplementationInfo() string {
    if IsRealImplementation() {
        flags := GetFlags()
        return fmt.Sprintf("RandomX C++ (flags: 0x%x)", flags)
    }
    return "RandomX Stub (development only)"
}
```

### Phase 6: Documentation & Deployment (Week 6)

#### 6.1 Create User Documentation

Create `mining/randomx/README.md`:
```markdown
# RandomX Integration for Shell Reserve

This package provides RandomX proof-of-work integration for Shell Reserve mining.

## Building

### Prerequisites

- Go 1.19+ with CGO enabled
- C++ compiler (GCC 7+ or Clang 5+)
- CMake 3.5+
- 4GB+ RAM for building
- 2GB+ RAM for mining with dataset

### Build Instructions

```bash
# Clone with submodules
git clone --recursive https://github.com/toole-brendan/shell.git

# Build RandomX
cd shell/mining/randomx
make build-deps

# Build Shell with RandomX
cd ../..
go build -tags cgo ./...
```

### Testing

```bash
# Run tests (light mode)
go test -tags cgo ./mining/randomx

# Run all tests including dataset tests (requires 2GB+ RAM)
go test -tags cgo -v ./mining/randomx -run TestRandomXDataset
```

## Performance

Expected hash rates on modern hardware:
- Light mode (cache only): 500-2000 H/s
- Fast mode (with dataset): 2000-10000 H/s

## Troubleshooting

### Large Pages Support

For optimal performance on Linux:
```bash
# Enable huge pages
sudo sysctl -w vm.nr_hugepages=1280

# Add to /etc/sysctl.conf for persistence
vm.nr_hugepages=1280
```

### Build Errors

If you encounter build errors:
1. Ensure all submodules are initialized: `git submodule update --init --recursive`
2. Clean and rebuild: `make clean && make`
3. Check compiler versions: `gcc --version` or `clang --version`
```

#### 6.2 CI/CD Integration

Create `.github/workflows/randomx-test.yml`:
```yaml
name: RandomX Tests

on:
  push:
    paths:
      - 'mining/randomx/**'
      - 'third_party/randomx/**'
  pull_request:
    paths:
      - 'mining/randomx/**'
      - 'third_party/randomx/**'

jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        go: ['1.19', '1.20', '1.21']
    
    steps:
    - uses: actions/checkout@v3
      with:
        submodules: recursive
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go }}
    
    - name: Install dependencies (Ubuntu)
      if: matrix.os == 'ubuntu-latest'
      run: |
        sudo apt-get update
        sudo apt-get install -y cmake g++ libboost-all-dev
    
    - name: Install dependencies (macOS)
      if: matrix.os == 'macos-latest'
      run: |
        brew install cmake boost
    
    - name: Build RandomX
      run: |
        cd mining/randomx
        make build-deps
    
    - name: Run tests
      run: |
        cd mining/randomx
        go test -tags cgo -v -short .
    
    - name: Run benchmarks
      run: |
        cd mining/randomx
        go test -tags cgo -bench=. -benchtime=10s
```

## Implementation Timeline

- **Week 1**: Environment setup, RandomX submodule integration
- **Week 2**: CGO bindings implementation
- **Week 3**: Build system integration
- **Week 4**: Testing and validation
- **Week 5**: Mining code integration
- **Week 6**: Documentation and deployment

## Security Considerations

1. **Memory Safety**: RandomX uses large memory allocations. Ensure proper error handling.
2. **Thread Safety**: Each VM instance should be used by a single goroutine.
3. **Resource Limits**: Implement proper resource limits to prevent DoS.
4. **Seed Rotation**: Implement proper seed rotation every 2048 blocks as specified.

## Performance Optimization

1. **Dataset Mode**: Use full dataset for mining, cache-only for verification
2. **CPU Affinity**: Pin mining threads to specific CPU cores
3. **Large Pages**: Enable huge pages on Linux for 10-15% performance boost
4. **Compiler Flags**: Use `-march=native` for CPU-specific optimizations

## Maintenance

1. **RandomX Updates**: Monitor RandomX repository for security updates
2. **Performance Testing**: Regular benchmarking on target hardware
3. **Compatibility Testing**: Test on various OS/CPU combinations
4. **Security Audits**: Regular review of CGO boundary code

## Conclusion

This implementation plan provides a complete path from the current stub to a production-ready RandomX integration. The modular approach allows for incremental testing and validation at each phase. 