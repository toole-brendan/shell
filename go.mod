module github.com/toole-brendan/shell

go 1.23.2

toolchain go1.24.1

require (
	github.com/btcsuite/btcd v0.24.0
	github.com/btcsuite/btcd/btcec/v2 v2.3.5
	github.com/btcsuite/btcd/btcutil v1.1.5
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.0
	github.com/btcsuite/btcd/v2transport v1.0.1
	github.com/btcsuite/btclog v1.0.0
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792
	github.com/davecgh/go-spew v1.1.1
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1
	github.com/decred/dcrd/lru v1.1.3
	github.com/jessevdk/go-flags v1.6.1
	github.com/jrick/logrotate v1.1.2
	github.com/stretchr/testify v1.8.4
	github.com/syndtr/goleveldb v1.0.0
	golang.org/x/crypto v0.25.0
	golang.org/x/sys v0.22.0
	pgregory.net/rapid v1.2.0
)

require (
	github.com/aead/siphash v1.0.1 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.0 // indirect
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/kkdai/bstream v0.0.0-20161212061736-f391b8402d23 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Replace btcd imports with local modules during development
replace github.com/btcsuite/btcd => ./

replace github.com/btcsuite/btcd/wire => ./wire

replace github.com/btcsuite/btcd/chaincfg => ./chaincfg
