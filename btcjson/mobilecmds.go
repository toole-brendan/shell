// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson

// MobileX Mining RPC Commands
// These commands extend the standard RPC interface with mobile mining support.

// GetMobileBlockTemplateCmd defines the getmobileblocktemplate JSON-RPC command.
type GetMobileBlockTemplateCmd struct {
	Request *MobileTemplateRequest `jsonrpcusage:"{}"`
}

// MobileTemplateRequest defines the request parameters for getmobileblocktemplate.
type MobileTemplateRequest struct {
	Mode         string      `json:"mode,omitempty"`         // "template" or "proposal"
	Capabilities []string    `json:"capabilities,omitempty"` // Client capabilities
	DeviceInfo   *DeviceInfo `json:"device_info,omitempty"`  // Mobile device information
}

// DeviceInfo contains mobile device information for optimization.
type DeviceInfo struct {
	DeviceType   string  `json:"device_type"`   // iOS, Android
	SocModel     string  `json:"soc_model"`     // Snapdragon 8 Gen 3, A17 Pro, etc.
	MaxCores     int     `json:"max_cores"`     // Available CPU cores
	RAMSize      int     `json:"ram_size_mb"`   // RAM in MB
	NPUCapable   bool    `json:"npu_capable"`   // Has NPU support
	ThermalLimit float64 `json:"thermal_limit"` // Max operating temperature
}

// GetMobileMiningInfoCmd defines the getmobilemininginfo JSON-RPC command.
type GetMobileMiningInfoCmd struct{}

// SubmitMobileBlockCmd defines the submitmobileblock JSON-RPC command.
type SubmitMobileBlockCmd struct {
	HexBlock     string                  `json:"hexblock"`
	ThermalProof *ThermalProofSubmission `json:"thermal_proof,omitempty"`
}

// ThermalProofSubmission contains thermal compliance proof data.
type ThermalProofSubmission struct {
	Temperature float64 `json:"temperature"` // Current device temperature
	Frequency   uint64  `json:"frequency"`   // Operating frequency in MHz
	PowerUsage  float64 `json:"power_usage"` // Estimated power in watts
}

// GetMobileWorkCmd defines the getmobilework JSON-RPC command.
// Simplified interface for mobile miners that don't need full template.
type GetMobileWorkCmd struct {
	DeviceClass string `json:"device_class,omitempty"` // flagship, midrange, budget
}

// SubmitMobileWorkCmd defines the submitmobilework JSON-RPC command.
// Simplified work submission for mobile miners.
type SubmitMobileWorkCmd struct {
	WorkID       string `json:"work_id"`
	Nonce        string `json:"nonce"`
	ThermalProof string `json:"thermal_proof"`
}

// ValidateThermalProofCmd defines the validatethermalproof JSON-RPC command.
type ValidateThermalProofCmd struct {
	BlockHash    string `json:"blockhash"`
	ThermalProof uint64 `json:"thermal_proof"`
}

// GetMobileStatsCmd defines the getmobilestats JSON-RPC command.
type GetMobileStatsCmd struct {
	Window int `json:"window,omitempty"` // Time window in minutes
}

// Result types for mobile mining commands

// GetMobileBlockTemplateResult contains the result of getmobileblocktemplate.
type GetMobileBlockTemplateResult struct {
	// Standard template fields
	Bits         string   `json:"bits"`
	CurTime      int64    `json:"curtime"`
	Height       int64    `json:"height"`
	PreviousHash string   `json:"previousblockhash"`
	Target       string   `json:"target"`
	Transactions []string `json:"transactions"`

	// Mobile-specific fields
	MobileTarget    string         `json:"mobile_target"`      // Adjusted for mobile difficulty
	NPUWork         string         `json:"npu_work,omitempty"` // NPU computation parameters
	ThermalTarget   float64        `json:"thermal_target"`     // Target temperature
	WorkSize        MobileWorkSize `json:"work_size"`          // Optimized work parameters
	DeviceOptimized bool           `json:"device_optimized"`   // Whether optimized for device
}

// MobileWorkSize defines work parameters optimized for mobile devices.
type MobileWorkSize struct {
	SearchSpace   uint32 `json:"search_space"`   // Nonce search space
	NPUIterations uint32 `json:"npu_iterations"` // NPU call frequency
	CacheSize     uint32 `json:"cache_size"`     // Working memory size
}

// GetMobileMiningInfoResult contains mining information for mobile miners.
type GetMobileMiningInfoResult struct {
	Blocks             int64   `json:"blocks"`
	CurrentBlockSize   uint64  `json:"currentblocksize"`
	CurrentBlockTx     uint64  `json:"currentblocktx"`
	Difficulty         float64 `json:"difficulty"`
	MobileDifficulty   float64 `json:"mobile_difficulty"` // Mobile-adjusted difficulty
	Errors             string  `json:"errors"`
	Generate           bool    `json:"generate"`
	HashesPerSec       int64   `json:"hashespersec"`
	MobileMinersActive int64   `json:"mobile_miners_active"` // Active mobile miners
	MobileHashrate     float64 `json:"mobile_hashrate"`      // Mobile network hashrate
	NetworkHashPerSec  int64   `json:"networkhashps"`
	PooledTx           uint64  `json:"pooledtx"`
	TestNet            bool    `json:"testnet"`
	ThermalCompliance  float64 `json:"thermal_compliance"` // % of compliant blocks
}

// GetMobileWorkResult contains simplified work for mobile miners.
type GetMobileWorkResult struct {
	WorkID       string  `json:"work_id"`
	Data         string  `json:"data"`   // Block header data
	Target       string  `json:"target"` // Difficulty target
	NPUWork      string  `json:"npu_work,omitempty"`
	ThermalLimit float64 `json:"thermal_limit"`
}

// GetMobileStatsResult contains mobile mining statistics.
type GetMobileStatsResult struct {
	TotalMobileMiners   int64            `json:"total_mobile_miners"`
	MobileHashrate      float64          `json:"mobile_hashrate"`
	DeviceBreakdown     map[string]int64 `json:"device_breakdown"`     // By device type
	GeographicBreakdown map[string]int64 `json:"geographic_breakdown"` // By country/region
	ThermalViolations   int64            `json:"thermal_violations"`   // Rejected for thermal
	AverageTemperature  float64          `json:"average_temperature"`
	NPUUtilization      float64          `json:"npu_utilization"`     // % using NPU
	BlocksFoundMobile   int64            `json:"blocks_found_mobile"` // Blocks by mobile miners
}

// init registers all mobile mining commands.
func init() {
	// No special flags for mobile mining commands.
	flags := UsageFlag(0)

	// Register mobile mining commands
	MustRegisterCmd("getmobileblocktemplate", (*GetMobileBlockTemplateCmd)(nil), flags)
	MustRegisterCmd("getmobilemininginfo", (*GetMobileMiningInfoCmd)(nil), flags)
	MustRegisterCmd("submitmobileblock", (*SubmitMobileBlockCmd)(nil), flags)
	MustRegisterCmd("getmobilework", (*GetMobileWorkCmd)(nil), flags)
	MustRegisterCmd("submitmobilework", (*SubmitMobileWorkCmd)(nil), flags)
	MustRegisterCmd("validatethermalproof", (*ValidateThermalProofCmd)(nil), flags)
	MustRegisterCmd("getmobilestats", (*GetMobileStatsCmd)(nil), flags)
}
