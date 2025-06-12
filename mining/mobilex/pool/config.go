// Copyright (c) 2025 The Shell developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool

import (
	"time"
)

// PoolConfig defines configuration for the mobile mining pool.
type PoolConfig struct {
	// Network configuration
	StratumEndpoint   string        // TCP endpoint for Stratum protocol
	HTTPEndpoint      string        // HTTP endpoint for REST API
	ConnectionTimeout time.Duration // Connection timeout

	// Pool parameters
	PoolAddress     string  // Pool's Shell address for rewards
	PoolFeePercent  float64 // Pool fee percentage (e.g., 1.0 for 1%)
	PayoutThreshold float64 // Minimum payout amount in XSL

	// Difficulty settings
	InitialDifficulty   float64       // Starting difficulty for new miners
	MinMobileDifficulty float64       // Minimum difficulty for mobile devices
	MaxMobileDifficulty float64       // Maximum difficulty for mobile devices
	DifficultyRetarget  time.Duration // How often to adjust difficulty

	// Mobile-specific settings
	ThermalCompliance  bool    // Enforce thermal proof validation
	NPUBonus           float64 // Bonus multiplier for NPU-enabled devices
	DeviceOptimization bool    // Enable device-specific optimizations

	// Database configuration
	DatabasePath string // Path to pool database

	// Node connection
	NodeRPCHost string // Shell node RPC host
	NodeRPCPort int    // Shell node RPC port
	NodeRPCUser string // RPC username
	NodeRPCPass string // RPC password
}

// DefaultPoolConfig returns default pool configuration.
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		StratumEndpoint:   ":3333",
		HTTPEndpoint:      ":8080",
		ConnectionTimeout: 30 * time.Second,

		PoolFeePercent:  1.0,
		PayoutThreshold: 1.0, // 1 XSL minimum

		InitialDifficulty:   1.0,
		MinMobileDifficulty: 0.1,
		MaxMobileDifficulty: 100.0,
		DifficultyRetarget:  5 * time.Minute,

		ThermalCompliance:  true,
		NPUBonus:           1.1, // 10% bonus for NPU miners
		DeviceOptimization: true,

		DatabasePath: "pool.db",

		NodeRPCHost: "localhost",
		NodeRPCPort: 8534,
	}
}
