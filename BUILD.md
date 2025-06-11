# Shell Reserve - Build Instructions

**Shell (XSL) - Digital Gold for Central Banks**

## Quick Start

```bash
# Clone and build
git clone https://github.com/toole-brendan/shell.git
cd shell
make build test
```

## Prerequisites

### Required
- **Go 1.23.2+** - [Download](https://golang.org/dl/)
- **GCC/Clang** - For RandomX compilation
- **Git** - For source control

### Platform Support
- ✅ **Linux** (Ubuntu 20.04+, RHEL 8+, CentOS 8+)
- ✅ **macOS** (10.15+, Intel/Apple Silicon)  
- ✅ **Windows** (Windows 10+, WSL2 recommended)

### Development Tools (Optional)
```bash
# Linting and security
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## Build Process

### 1. Clone Repository
```bash
git clone https://github.com/toole-brendan/shell.git
cd shell
```

### 2. Install Dependencies
```bash
make deps
```
This will:
- Download Go modules
- Build RandomX mining library
- Verify dependencies

### 3. Build Shell Reserve
```bash
make build
```

### 4. Run Tests
```bash
make test
```

### 5. Development Build (with race detection)
```bash
make build-race
```

## Network Configurations

### Testnet (Safe for Testing)
```bash
make testnet
```

### Regression Test (Local Development)
```bash
make regtest
```

### Simulation Network
```bash
make simnet
```

### Mainnet (PRODUCTION - January 1, 2026)
```bash
make mainnet  # Available after launch date
```

## Institutional Setup

### 1. Basic Configuration
```bash
# Generate configuration files
./shell --configfile=shell-institutional.conf --generate-config

# Edit configuration for your institution
nano shell-institutional.conf
```

### 2. Custody Configuration
```bash
# Generate multisig addresses
./shell --regtest --generate-multisig \
  --m=3 --n=5 \
  --pubkeys=pubkey1,pubkey2,pubkey3,pubkey4,pubkey5

# Time-locked addresses  
./shell --regtest --generate-timelock \
  --locktime=144 \  # 1 day (144 blocks)
  --pubkey=your_pubkey
```

### 3. Document Hash Commitments
```bash
# Commit trade document hash
./shell --regtest --commit-document \
  --hash=sha256_hash_of_document \
  --reference="Bill of Lading BOL-2025-001" \
  --fee=0.02
```

### 4. Bilateral Channels
```bash
# Open payment channel
./shell --regtest --open-channel \
  --counterparty=counterparty_pubkey \
  --amount=1000 \
  --capacity=10000
```

## Testing

### Unit Tests
```bash
make test
```

### Integration Tests
```bash
make test-integration
```

### Coverage Report
```bash
make test-coverage
# Opens coverage.html in browser
```

### Benchmarks
```bash
make bench
```

## Quality Assurance

### Code Linting
```bash
make lint
```

### Security Audit
```bash
make audit
```

### Vulnerability Check
```bash
make vuln-check
```

### Full Release Check
```bash
make release-check
```

## Docker Support

### Build Docker Image
```bash
make docker-build
```

### Run in Docker
```bash
make docker-run
```

### Custom Docker Run
```bash
docker run -d \
  --name shell-reserve \
  -p 8533:8533 \
  -p 8534:8534 \
  -v $(pwd)/data:/root/.btcd/data \
  shell-reserve:0.24.2-beta
```

## Production Deployment

### Hardware Requirements (Minimum)
- **CPU**: 2 cores, 2.0 GHz+
- **RAM**: 4 GB 
- **Storage**: 100 GB SSD
- **Network**: Stable internet connection

### Recommended (Institutional)
- **CPU**: 4-8 cores, 3.0 GHz+
- **RAM**: 16-32 GB
- **Storage**: 1 TB NVMe SSD
- **Network**: Redundant internet, low latency
- **Backup**: RAID configuration

### Security Considerations
1. **Cold Storage**: Use hardware security modules (HSMs)
2. **Multisig**: Implement 3-of-5 or 5-of-7 custody
3. **Time Locks**: Add delays for large transactions
4. **Network**: Run behind institutional firewall
5. **Monitoring**: Set up transaction monitoring
6. **Backup**: Regular encrypted backups

## Troubleshooting

### RandomX Build Issues
```bash
# Clean and rebuild
make randomx-clean
make randomx

# Check dependencies
sudo apt-get install build-essential cmake  # Ubuntu/Debian
brew install cmake                           # macOS
```

### Go Module Issues
```bash
# Clean module cache
go clean -modcache
go mod download
```

### Permission Issues
```bash
# Fix permissions
chmod +x shell
sudo chown -R $USER:$USER ~/.btcd/
```

### Network Connectivity
```bash
# Test P2P connectivity
./shell --regtest --debuglevel=debug 2>&1 | grep "peer"

# Check RPC connectivity  
curl -u user:pass -d '{"method":"getinfo"}' \
  http://localhost:8334/
```

## Configuration Examples

### Institutional Node (shell-institutional.conf)
```ini
# Shell Reserve Institutional Configuration

# Network
listen=0.0.0.0:8533
externalip=your.institution.ip

# RPC (Internal only)
rpclisten=127.0.0.1:8534
rpcuser=your_rpc_user
rpcpass=your_rpc_password

# Mining (if participating)
generate=false
miningaddr=your_mining_address

# Database
dbtype=ffldb
datadir=/data/shell-reserve

# Logging
debuglevel=info
logdir=/logs/shell-reserve

# Institutional settings
minrelaytxfee=0.001
limitfreerelay=0
blockmaxsize=500000
```

### High-Security Cold Storage (shell-cold.conf)
```ini
# Cold Storage Configuration

# Disable network
nolisten=true
noconnect=true

# Enable signing only
rpclisten=127.0.0.1:8534
rpcuser=cold_storage_user
rpcpass=your_secure_password

# Minimal logging
debuglevel=warn
```

## Monitoring and Maintenance

### Log Monitoring
```bash
# Real-time logs
tail -f ~/.btcd/logs/btcd.log

# Error monitoring
grep -i error ~/.btcd/logs/btcd.log
```

### Performance Monitoring
```bash
# Memory usage
./shell --regtest --debuglevel=debug 2>&1 | grep "memory"

# Block processing time
./shell --regtest --debuglevel=debug 2>&1 | grep "block"
```

### Backup Procedures
```bash
# Backup wallet and configuration
tar -czf shell-backup-$(date +%Y%m%d).tar.gz \
  ~/.btcd/data/ \
  shell-institutional.conf

# Secure backup storage
gpg --encrypt --recipient your@institution.org \
  shell-backup-$(date +%Y%m%d).tar.gz
```

## Support and Resources

### Documentation
- [White Paper](README.md) - Complete vision and architecture
- [Implementation Plan](Shell%20Implementation%20Plan.md) - Technical roadmap
- [API Documentation](http://localhost:6060) - Run `make docs`

### Community
- GitHub Issues: Report bugs and request features
- Institutional Support: Contact your Shell Reserve liaison

### Launch Information
- **Mainnet Launch**: January 1, 2026, 00:00:00 UTC
- **Genesis Block**: No premine, fair launch
- **Mining**: RandomX CPU mining available immediately

---

**Shell Reserve: Essential features, eternal reliability.**

*Built for institutions that think in decades, not quarters.* 