// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2025 Shell Reserve developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"errors"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

// These variables are the chain proof-of-work limit parameters for each default
// network.
var (
	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// mainPowLimit is the highest proof of work value a Shell block can
	// have for the main network. RandomX has different characteristics than SHA256.
	mainPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)

	// regressionPowLimit is the highest proof of work value a Shell block
	// can have for the regression test network.  It is the value 2^255 - 1.
	regressionPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	// testNet3PowLimit is the highest proof of work value a Shell block
	// can have for the test network (version 3).  It is the value
	// 2^224 - 1.
	testNet3PowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)

	// simNetPowLimit is the highest proof of work value a Shell block
	// can have for the simulation test network.  It is the value 2^255 - 1.
	simNetPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)
)

// Checkpoint identifies a known good point in the block chain.  Using
// checkpoints allows a few optimizations for old blocks during initial download
// and also prevents forks from old blocks.
//
// Each checkpoint is selected based upon several factors.  See the
// documentation for blockchain.IsCheckpointCandidate for details on the
// selection criteria.
type Checkpoint struct {
	Height int32
	Hash   *chainhash.Hash
}

// EffectiveAlwaysActiveHeight returns the effective activation height for the
// deployment. If AlwaysActiveHeight is unset (i.e. zero), it returns
// the maximum uint32 value to indicate that it does not force activation.
func (d *ConsensusDeployment) EffectiveAlwaysActiveHeight() uint32 {
	if d.AlwaysActiveHeight == 0 {
		return math.MaxUint32
	}
	return d.AlwaysActiveHeight
}

// DNSSeed identifies a DNS seed.
type DNSSeed struct {
	// Host defines the hostname of the seed.
	Host string

	// HasFiltering defines whether the seed supports filtering
	// by service flags (wire.ServiceFlag).
	HasFiltering bool
}

// ConsensusDeployment defines details related to a specific consensus rule
// change that is voted in.  This is part of BIP0009.
type ConsensusDeployment struct {
	// BitNumber defines the specific bit number within the block version
	// this particular soft-fork deployment refers to.
	BitNumber uint8

	// MinActivationHeight is an optional field that when set (default
	// value being zero), modifies the traditional BIP 9 state machine by
	// only transitioning from LockedIn to Active once the block height is
	// greater than (or equal to) thus specified height.
	MinActivationHeight uint32

	// CustomActivationThreshold if set (non-zero), will _override_ the
	// existing RuleChangeActivationThreshold value set at the
	// network/chain level. This value divided by the active
	// MinerConfirmationWindow denotes the threshold required for
	// activation. A value of 1815 block denotes a 90% threshold.
	CustomActivationThreshold uint32

	// AlwaysActiveHeight defines an optional block threshold at which the
	// deployment is forced to be active. If unset (0), it defaults to
	// math.MaxUint32, meaning the deployment does not force activation.
	AlwaysActiveHeight uint32

	// DeploymentStarter is used to determine if the given
	// ConsensusDeployment has started or not.
	DeploymentStarter ConsensusDeploymentStarter

	// DeploymentEnder is used to determine if the given
	// ConsensusDeployment has ended or not.
	DeploymentEnder ConsensusDeploymentEnder
}

// Constants that define the deployment offset in the deployments field of the
// parameters for each deployment.  This is useful to be able to get the details
// of a specific deployment by name.
const (
	// DeploymentTestDummy defines the rule change deployment ID for testing
	// purposes.
	DeploymentTestDummy = iota

	// DeploymentCSV defines the rule change deployment ID for the CSV
	// soft-fork package. The CSV package includes the deployment of BIPS
	// 68, 112, and 113.
	DeploymentCSV

	// DeploymentSegwit defines the rule change deployment ID for the
	// Segregated Witness (segwit) soft-fork package. The segwit package
	// includes the deployment of BIPS 141, 142, 144, 145, 147 and 173.
	DeploymentSegwit

	// DeploymentTaproot defines the rule change deployment ID for the
	// Taproot (+Schnorr) soft-fork package. Active from genesis in Shell.
	DeploymentTaproot

	// DeploymentConfidentialTx defines the rule change deployment for
	// confidential transactions. Active from genesis in Shell.
	DeploymentConfidentialTx

	// DeploymentPaymentChannels defines the rule change deployment for
	// Layer 1 payment channels. Active from genesis.
	DeploymentPaymentChannels

	// DeploymentPrivacyLayer defines the rule change deployment for
	// Layer 0.5 privacy features (ring signatures, stealth addresses).
	// Activates after ~2 years of operation.
	DeploymentPrivacyLayer

	// DeploymentVaultCovenants defines the rule change deployment for
	// institutional vault covenants.
	DeploymentVaultCovenants

	// NOTE: DefinedDeployments must always come last since it is used to
	// determine how many defined deployments there currently are.

	// DefinedDeployments is the number of currently defined deployments.
	DefinedDeployments
)

// Params defines a Shell network by its parameters.  These parameters may be
// used by Shell applications to differentiate networks as well as addresses
// and keys for one network from those intended for use on another network.
type Params struct {
	// Name defines a human-readable identifier for the network.
	Name string

	// Net defines the magic bytes used to identify the network.
	Net wire.BitcoinNet

	// DefaultPort defines the default peer-to-peer port for the network.
	DefaultPort string

	// DNSSeeds defines a list of DNS seeds for the network that are used
	// as one method to discover peers.
	DNSSeeds []DNSSeed

	// GenesisBlock defines the first block of the chain.
	GenesisBlock *wire.MsgBlock

	// GenesisHash is the starting block hash.
	GenesisHash *chainhash.Hash

	// PowLimit defines the highest allowed proof of work value for a block
	// as a uint256.
	PowLimit *big.Int

	// PowLimitBits defines the highest allowed proof of work value for a
	// block in compact form.
	PowLimitBits uint32

	// PoWNoRetargeting defines whether the network has difficulty
	// retargeting enabled or not. This should only be set to true for
	// regtest like networks.
	PoWNoRetargeting bool

	// EnforceBIP94 specifies whether BIP94 (testnet difficulty retargeting
	// rules) should be enforced.
	EnforceBIP94 bool

	// Shell-specific parameters
	MaxSupply int64 // 100,000,000 XSL maximum supply

	// RandomX parameters for CPU-friendly mining
	RandomXSeedRotation int32 // Blocks between seed changes
	RandomXMemory       int64 // Memory requirement (2GB)

	// Layer activation heights
	L1ActivationHeight  int32 // Payment channels from genesis
	L05ActivationHeight int32 // Privacy layer after ~2 years

	// These fields define the block heights at which the specified softfork
	// BIP became active.
	BIP0034Height int32 // Not applicable for Shell (starts with v2+ blocks)
	BIP0065Height int32 // CHECKLOCKTIMEVERIFY
	BIP0066Height int32 // Strict DER signatures

	// CoinbaseMaturity is the number of blocks required before newly mined
	// coins (coinbase transactions) can be spent.
	CoinbaseMaturity uint16

	// SubsidyReductionInterval is the interval of blocks before the subsidy
	// is reduced (halving). Shell uses ~10 year intervals.
	SubsidyReductionInterval int32

	// TargetTimespan is the desired amount of time that should elapse
	// before the block difficulty requirement is examined to determine how
	// it should be changed in order to maintain the desired block
	// generation rate. Shell uses daily adjustments.
	TargetTimespan time.Duration

	// TargetTimePerBlock is the desired amount of time to generate each
	// block. Shell targets 5 minutes instead of Bitcoin's 10.
	TargetTimePerBlock time.Duration

	// RetargetAdjustmentFactor is the adjustment factor used to limit
	// the minimum and maximum amount of adjustment that can occur between
	// difficulty retargets.
	RetargetAdjustmentFactor int64

	// ReduceMinDifficulty defines whether the network should reduce the
	// minimum required difficulty after a long enough period of time has
	// passed without finding a block.  This is really only useful for test
	// networks and should not be set on a main network.
	ReduceMinDifficulty bool

	// MinDiffReductionTime is the amount of time after which the minimum
	// required difficulty should be reduced when a block hasn't been found.
	//
	// NOTE: This only applies if ReduceMinDifficulty is true.
	MinDiffReductionTime time.Duration

	// GenerateSupported specifies whether or not CPU mining is allowed.
	// Shell supports CPU mining with RandomX.
	GenerateSupported bool

	// Checkpoints ordered from oldest to newest.
	Checkpoints []Checkpoint

	// These fields are related to voting on consensus rule changes as
	// defined by BIP0009.
	//
	// RuleChangeActivationThreshold is the number of blocks in a threshold
	// state retarget window for which a positive vote for a rule change
	// must be cast in order to lock in a rule change. It should typically
	// be 95% for the main network and 75% for test networks.
	//
	// MinerConfirmationWindow is the number of blocks in each threshold
	// state retarget window.
	//
	// Deployments define the specific consensus rule changes to be voted
	// on.
	RuleChangeActivationThreshold uint32
	MinerConfirmationWindow       uint32
	Deployments                   [DefinedDeployments]ConsensusDeployment

	// Mempool parameters
	RelayNonStdTxs bool

	// Human-readable part for Bech32 encoded segwit addresses, as defined
	// in BIP 173. Shell uses "xsl" prefix.
	Bech32HRPSegwit string

	// Address encoding magics for Shell Reserve
	PubKeyHashAddrID        byte // First byte of a P2PKH address
	ScriptHashAddrID        byte // First byte of a P2SH address
	PrivateKeyID            byte // First byte of a WIF private key
	WitnessPubKeyHashAddrID byte // First byte of a P2WPKH address
	WitnessScriptHashAddrID byte // First byte of a P2WSH address

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID [4]byte
	HDPublicKeyID  [4]byte

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation. Shell will use coin type 8533.
	HDCoinType uint32
}

// MainNetParams defines the network parameters for the main Shell network.
var MainNetParams = Params{
	Name:        "mainnet",
	Net:         wire.ShellMainNet, // Custom network magic
	DefaultPort: "8533",            // Shell default port
	DNSSeeds: []DNSSeed{
		{"seed1.shell.org", true},
		{"seed2.shell.org", true},
		{"seed3.shell.org", true},
		{"seed4.shell.org", true},
		{"seed5.shell.org", true},
	},

	// Chain parameters
	GenesisBlock:     &shellGenesisBlock,
	GenesisHash:      &shellGenesisHash,
	PowLimit:         mainPowLimit,
	PowLimitBits:     0x1d00ffff, // Initial difficulty
	PoWNoRetargeting: false,
	EnforceBIP94:     false, // Not a testnet
	BIP0034Height:    0,     // Shell starts with v2+ blocks
	BIP0065Height:    0,     // Active from genesis
	BIP0066Height:    0,     // Active from genesis
	CoinbaseMaturity: 100,   // Same as Bitcoin

	// Shell-specific economic parameters
	MaxSupply:                100000000 * 1e8, // 100M XSL maximum
	SubsidyReductionInterval: 262800,          // ~10 years (5min * 12 * 24 * 365 * 10)
	TargetTimespan:           time.Hour * 24,  // Daily difficulty adjustment
	TargetTimePerBlock:       time.Minute * 5, // 5-minute blocks
	RetargetAdjustmentFactor: 4,               // ±25% max adjustment
	ReduceMinDifficulty:      false,
	MinDiffReductionTime:     0,
	GenerateSupported:        true, // RandomX CPU mining supported

	// RandomX parameters
	RandomXSeedRotation: 2048,                   // Seed rotation every 2048 blocks
	RandomXMemory:       2 * 1024 * 1024 * 1024, // 2GB memory requirement

	// Layer activation heights
	L1ActivationHeight:  0,      // Payment channels from genesis
	L05ActivationHeight: 525600, // Privacy layer after ~10 years

	// Checkpoints ordered from oldest to newest (empty for new network)
	Checkpoints: []Checkpoint{},

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 274, // 95% of MinerConfirmationWindow
	MinerConfirmationWindow:       288, // Daily retarget window (288 blocks * 5min = 24h)
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber: 28,
			DeploymentStarter: NewMedianTimeDeploymentStarter(
				time.Time{}, // Available for testing
			),
			DeploymentEnder: NewMedianTimeDeploymentEnder(
				time.Time{}, // Never expires
			),
		},
		DeploymentCSV: {
			BitNumber:          0,
			AlwaysActiveHeight: 0, // Active from genesis in Shell
			DeploymentStarter: NewMedianTimeDeploymentStarter(
				time.Time{}, // Always active
			),
			DeploymentEnder: NewMedianTimeDeploymentEnder(
				time.Time{}, // Never expires
			),
		},
		DeploymentSegwit: {
			BitNumber:          1,
			AlwaysActiveHeight: 0, // Active from genesis in Shell
			DeploymentStarter: NewMedianTimeDeploymentStarter(
				time.Time{}, // Always active
			),
			DeploymentEnder: NewMedianTimeDeploymentEnder(
				time.Time{}, // Never expires
			),
		},
		DeploymentTaproot: {
			BitNumber:          2,
			AlwaysActiveHeight: 0, // Active from genesis
			DeploymentStarter: NewMedianTimeDeploymentStarter(
				time.Time{}, // Always active
			),
			DeploymentEnder: NewMedianTimeDeploymentEnder(
				time.Time{}, // Never expires
			),
		},
		DeploymentConfidentialTx: {
			BitNumber:          3,
			AlwaysActiveHeight: 0, // Active from genesis
			DeploymentStarter: NewMedianTimeDeploymentStarter(
				time.Time{}, // Always active
			),
			DeploymentEnder: NewMedianTimeDeploymentEnder(
				time.Time{}, // Never expires
			),
		},
		DeploymentPaymentChannels: {
			BitNumber:          4,
			AlwaysActiveHeight: 0, // L1 active from genesis
			DeploymentStarter: NewMedianTimeDeploymentStarter(
				time.Time{}, // Always active
			),
			DeploymentEnder: NewMedianTimeDeploymentEnder(
				time.Time{}, // Never expires
			),
		},
		DeploymentPrivacyLayer: {
			BitNumber:                 5,
			MinActivationHeight:       525600, // ~10 years
			CustomActivationThreshold: 274,    // 95% threshold
			DeploymentStarter: NewMedianTimeDeploymentStarter(
				time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC), // 2 years after launch
			),
			DeploymentEnder: NewMedianTimeDeploymentEnder(
				time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), // 2-year window
			),
		},
		DeploymentVaultCovenants: {
			BitNumber:          6,
			AlwaysActiveHeight: 0, // Active from genesis for institutional use
			DeploymentStarter: NewMedianTimeDeploymentStarter(
				time.Time{}, // Always active
			),
			DeploymentEnder: NewMedianTimeDeploymentEnder(
				time.Time{}, // Never expires
			),
		},
	},

	// Mempool parameters
	RelayNonStdTxs: false,

	// Human-readable part for Bech32 encoded segwit addresses
	Bech32HRPSegwit: "xsl", // Shell Reserve prefix

	// Address encoding magics for Shell Reserve
	PubKeyHashAddrID:        0x78, // starts with 'X' for Shell
	ScriptHashAddrID:        0x7D, // starts with 'x' for Shell scripts
	PrivateKeyID:            0xF8, // WIF private keys
	WitnessPubKeyHashAddrID: 0x06, // Taproot addresses
	WitnessScriptHashAddrID: 0x0A, // Taproot script addresses

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x88, 0xE1, 0x37}, // starts with xslv
	HDPublicKeyID:  [4]byte{0x04, 0x88, 0xE5, 0x6A}, // starts with xslu

	// BIP44 coin type for Shell Reserve
	HDCoinType: 8533, // Shell's port number as coin type
}

var (
	// ErrDuplicateNet describes an error where the parameters for a Bitcoin
	// network could not be set due to the network already being a standard
	// network or previously-registered into this package.
	ErrDuplicateNet = errors.New("duplicate Bitcoin network")

	// ErrUnknownHDKeyID describes an error where the provided id which
	// is intended to identify the network for a hierarchical deterministic
	// private extended key is not registered.
	ErrUnknownHDKeyID = errors.New("unknown hd private extended key bytes")

	// ErrInvalidHDKeyID describes an error where the provided hierarchical
	// deterministic version bytes, or hd key id, is malformed.
	ErrInvalidHDKeyID = errors.New("invalid hd extended key version bytes")
)

var (
	registeredNets       = make(map[wire.BitcoinNet]struct{})
	pubKeyHashAddrIDs    = make(map[byte]struct{})
	scriptHashAddrIDs    = make(map[byte]struct{})
	bech32SegwitPrefixes = make(map[string]struct{})
	hdPrivToPubKeyIDs    = make(map[[4]byte][]byte)
)

// String returns the hostname of the DNS seed in human-readable form.
func (d DNSSeed) String() string {
	return d.Host
}

// Register registers the network parameters for a Bitcoin network.  This may
// error with ErrDuplicateNet if the network is already registered (either
// due to a previous Register call, or the network being one of the default
// networks).
//
// Network parameters should be registered into this package by a main package
// as early as possible.  Then, library packages may lookup networks or network
// parameters based on inputs and work regardless of the network being standard
// or not.
func Register(params *Params) error {
	if _, ok := registeredNets[params.Net]; ok {
		return ErrDuplicateNet
	}
	registeredNets[params.Net] = struct{}{}
	pubKeyHashAddrIDs[params.PubKeyHashAddrID] = struct{}{}
	scriptHashAddrIDs[params.ScriptHashAddrID] = struct{}{}

	err := RegisterHDKeyID(params.HDPublicKeyID[:], params.HDPrivateKeyID[:])
	if err != nil {
		return err
	}

	// A valid Bech32 encoded segwit address always has as prefix the
	// human-readable part for the given net followed by '1'.
	bech32SegwitPrefixes[params.Bech32HRPSegwit+"1"] = struct{}{}
	return nil
}

// mustRegister performs the same function as Register except it panics if there
// is an error.  This should only be called from package init functions.
func mustRegister(params *Params) {
	if err := Register(params); err != nil {
		panic("failed to register network: " + err.Error())
	}
}

// IsPubKeyHashAddrID returns whether the id is an identifier known to prefix a
// pay-to-pubkey-hash address on any default or registered network.  This is
// used when decoding an address string into a specific address type.  It is up
// to the caller to check both this and IsScriptHashAddrID and decide whether an
// address is a pubkey hash address, script hash address, neither, or
// undeterminable (if both return true).
func IsPubKeyHashAddrID(id byte) bool {
	_, ok := pubKeyHashAddrIDs[id]
	return ok
}

// IsScriptHashAddrID returns whether the id is an identifier known to prefix a
// pay-to-script-hash address on any default or registered network.  This is
// used when decoding an address string into a specific address type.  It is up
// to the caller to check both this and IsPubKeyHashAddrID and decide whether an
// address is a pubkey hash address, script hash address, neither, or
// undeterminable (if both return true).
func IsScriptHashAddrID(id byte) bool {
	_, ok := scriptHashAddrIDs[id]
	return ok
}

// IsBech32SegwitPrefix returns whether the prefix is a known prefix for segwit
// addresses on any default or registered network.  This is used when decoding
// an address string into a specific address type.
func IsBech32SegwitPrefix(prefix string) bool {
	prefix = strings.ToLower(prefix)
	_, ok := bech32SegwitPrefixes[prefix]
	return ok
}

// RegisterHDKeyID registers a public and private hierarchical deterministic
// extended key ID pair.
//
// Non-standard HD version bytes, such as the ones documented in SLIP-0132,
// should be registered using this method for library packages to lookup key
// IDs (aka HD version bytes). When the provided key IDs are invalid, the
// ErrInvalidHDKeyID error will be returned.
//
// Reference:
//
//	SLIP-0132 : Registered HD version bytes for BIP-0032
//	https://github.com/satoshilabs/slips/blob/master/slip-0132.md
func RegisterHDKeyID(hdPublicKeyID []byte, hdPrivateKeyID []byte) error {
	if len(hdPublicKeyID) != 4 || len(hdPrivateKeyID) != 4 {
		return ErrInvalidHDKeyID
	}

	var keyID [4]byte
	copy(keyID[:], hdPrivateKeyID)
	hdPrivToPubKeyIDs[keyID] = hdPublicKeyID

	return nil
}

// HDPrivateKeyToPublicKeyID accepts a private hierarchical deterministic
// extended key id and returns the associated public key id.  When the provided
// id is not registered, the ErrUnknownHDKeyID error will be returned.
func HDPrivateKeyToPublicKeyID(id []byte) ([]byte, error) {
	if len(id) != 4 {
		return nil, ErrUnknownHDKeyID
	}

	var key [4]byte
	copy(key[:], id)
	pubBytes, ok := hdPrivToPubKeyIDs[key]
	if !ok {
		return nil, ErrUnknownHDKeyID
	}

	return pubBytes, nil
}

// newHashFromStr converts the passed big-endian hex string into a
// chainhash.Hash.  It only differs from the one available in chainhash in that
// it panics on an error since it will only (and must only) be called with
// hard-coded, and therefore known good, hashes.
func newHashFromStr(hexStr string) *chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(hexStr)
	if err != nil {
		// Ordinarily I don't like panics in library code since it
		// can take applications down without them having a chance to
		// recover which is extremely annoying, however an exception is
		// being made in this case because the only way this can panic
		// is if there is an error in the hard-coded hashes.  Thus it
		// will only ever potentially panic on init and therefore is
		// 100% predictable.
		panic(err)
	}
	return hash
}

// TestNet3Params defines the network parameters for the test Bitcoin network
// (version 3).  Not applicable for Shell Reserve but included for compatibility.
var TestNet3Params = MainNetParams

// RegressionNetParams defines the network parameters for the regression test
// Bitcoin network.  Not applicable for Shell Reserve but included for compatibility.
var RegressionNetParams = MainNetParams

func init() {
	// Register all default networks when the package is initialized.
	mustRegister(&MainNetParams)
}
