# Shell Reserve - Quick Test Guide

Quick reference for running specific test scenarios during development.

## Most Common Test Commands

### 1. Quick Smoke Test (5 minutes)
```bash
# Basic functionality check
make test-short

# Or manually:
go test ./blockchain/... ./txscript/... ./mempool/... -short
```

### 2. Feature-Specific Testing

#### Test Claimable Balances
```bash
go test ./settlement/claimable/... -v -run TestClaimable
```

#### Test Document Hashes
```bash
go test ./txscript/... -v -run "TestDocHash|TestOP_DOC_HASH"
```

#### Test Bilateral Channels
```bash
go test ./settlement/channels/... -v -run TestChannel
```

#### Test ISO 20022
```bash
go test ./settlement/iso20022/... -v -run "TestISO|TestSWIFT"
```

### 3. Performance Quick Check
```bash
# Run key benchmarks
go test -bench="BenchmarkBlock|BenchmarkTx|BenchmarkClaimable" -run=^$ ./...
```

### 4. Security Quick Scan
```bash
# Fast security check
gosec -tests ./... | grep "Severity: HIGH"
```

### 5. Network Simulation (Local)
```bash
# Start 3-node local testnet
make testnet-local

# In another terminal, run tests
go test ./test/... -run TestLocalNetwork -testnet
```

## Focused Test Scenarios

### Before Each PR
```bash
# Run this before creating any PR
./test-runner.sh -quick

# Or manually:
go test ./...
make lint
make security-scan
```

### After Major Changes
```bash
# Comprehensive test after significant modifications
RUN_CHAOS_TESTS=true ./test-runner.sh
```

### Testing Specific Attack Vectors
```bash
# Double spend attempts
go test ./test/... -run TestDoubleSpend -v

# Block propagation under attack
go test ./test/... -run "TestBlock.*Attack" -v

# Channel disputes
go test ./settlement/channels/... -run TestDispute -v
```

### Memory Leak Detection
```bash
# Run with memory profiling
go test -run TestMemoryLeak -memprofile mem.out ./test/...
go tool pprof -http=:8080 mem.out
```

## Debugging Failed Tests

### 1. Verbose Output
```bash
go test -v ./path/to/package/...
```

### 2. Run Single Test
```bash
go test -run TestSpecificFunction ./path/to/package/
```

### 3. With Race Detection
```bash
go test -race ./...
```

### 4. With Coverage
```bash
go test -coverprofile=cover.out ./...
go tool cover -html=cover.out
```

## CI/CD Test Commands

### GitHub Actions
```yaml
- name: Core Tests
  run: go test ./blockchain/... ./txscript/... ./mempool/...

- name: Settlement Tests  
  run: go test ./settlement/...

- name: Integration Tests
  run: go test ./test/... -tags=integration
```

### Local CI Simulation
```bash
# Simulate CI environment
docker run -v $(pwd):/go/src/shell golang:1.21 \
  bash -c "cd /go/src/shell && make test"
```

## Test Data Generation

### Generate Test Transactions
```bash
go run test/generators/main.go -type=tx -count=1000 > test_txs.json
```

### Generate Test Blocks
```bash
go run test/generators/main.go -type=block -size=500kb > test_block.dat
```

### Generate Document Hashes
```bash
go run test/generators/main.go -type=dochash -count=100 > test_docs.json
```

## Troubleshooting

### Tests Hanging?
```bash
# Run with timeout
go test -timeout 30s ./...
```

### Flaky Tests?
```bash
# Run multiple times to detect flakiness
go test -count=10 ./problematic/package/...
```

### Need More Details?
```bash
# Maximum verbosity
go test -v -logtostderr=true -vmodule=*=5 ./...
```

---

Remember: When in doubt, run `make test`. It's configured with sensible defaults. 