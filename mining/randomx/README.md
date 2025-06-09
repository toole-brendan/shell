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

## API Usage

### Basic Example

```go
package main

import (
    "fmt"
    "encoding/hex"
    "github.com/toole-brendan/shell/mining/randomx"
)

func main() {
    // Create cache with seed
    cache, err := randomx.NewCache([]byte("my seed"))
    if err != nil {
        panic(err)
    }
    defer cache.Close()

    // Create VM (light mode)
    vm, err := randomx.NewVM(cache, nil)
    if err != nil {
        panic(err)
    }
    defer vm.Close()

    // Calculate hash
    input := []byte("data to hash")
    hash := vm.CalcHash(input)
    
    fmt.Printf("Hash: %s\n", hex.EncodeToString(hash))
}
```

### Full Dataset Mode

```go
// Create dataset for better performance (requires 2GB+ RAM)
dataset, err := randomx.NewDataset(cache)
if err != nil {
    panic(err)
}
defer dataset.Close()

// Create VM with dataset
vm, err := randomx.NewVM(cache, dataset)
```

## Implementation Details

- Uses RandomX v1.2.1 stable release
- Supports both light mode (256MB cache) and full mode (2GB+ dataset)
- Thread-safe VM instances (each goroutine needs its own VM)
- Automatic memory cleanup via Go finalizers
- Falls back to stub implementation when CGO is unavailable

## Benchmarking

Run benchmarks:
```bash
# Light mode benchmark
go test -tags cgo -bench=BenchmarkRandomXLight -benchtime=30s ./mining/randomx

# Full dataset benchmark (requires 2GB+ RAM)
go test -tags cgo -bench=BenchmarkRandomXFull -benchtime=30s ./mining/randomx
``` 