// Package liquidity implements Shell Reserve's alliance coordination system
// for professional market makers and institutional trading partners.
package liquidity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
)

// Fee constants (copied to avoid import cycle)
const (
	BaseFeeRate      = 0.0003     // 0.0003 XSL per byte (burned)
	MakerRebate      = 0.0001     // 0.0001 XSL per byte rebate for makers
	ChannelOpenFee   = 0.1 * 1e8  // 0.1 XSL to open payment channel
	ChannelUpdateFee = 0.01 * 1e8 // 0.01 XSL for channel updates
	AtomicSwapFee    = 0.05 * 1e8 // 0.05 XSL for atomic swaps
	ClaimableFee     = 0.02 * 1e8 // 0.02 XSL for claimable balances
)

// FeeCalculatorInterface defines interface to avoid import cycle
type FeeCalculatorInterface interface {
	GetFeeRate() float64
	GetMakerRebateRate() float64
	EstimateFee(txSize int, hasShellOpcodes bool, isMaker bool) int64
}

// SimpleFeeCalculator provides basic fee calculation without import cycle
type SimpleFeeCalculator struct{}

func (fc *SimpleFeeCalculator) GetFeeRate() float64 {
	return BaseFeeRate
}

func (fc *SimpleFeeCalculator) GetMakerRebateRate() float64 {
	return MakerRebate
}

func (fc *SimpleFeeCalculator) EstimateFee(txSize int, hasShellOpcodes bool, isMaker bool) int64 {
	baseFee := int64(float64(txSize) * BaseFeeRate * 1e8)

	var rebate int64
	if isMaker {
		rebate = int64(float64(txSize) * MakerRebate * 1e8)
	}

	var operationFee int64
	if hasShellOpcodes {
		operationFee = int64(ChannelOpenFee) // Conservative estimate
	}

	netFee := baseFee + operationFee - rebate
	if netFee < 0 {
		netFee = 0
	}

	return netFee
}

// AllianceAPI provides professional trading interfaces for market makers
type AllianceAPI struct {
	liquidityManager *LiquidityManager
	attestorClient   *AttestorClient
	feeCalculator    FeeCalculatorInterface

	// Alliance member management
	members      map[string]*AllianceMember
	membersMutex sync.RWMutex

	// API server
	server *http.Server
}

// AllianceMember represents a professional market making partner
type AllianceMember struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	PublicKey        *btcec.PublicKey `json:"-"` // Not serialized
	PublicKeyHex     string           `json:"public_key"`
	TradingAddresses []string         `json:"trading_addresses"`
	APIEndpoint      string           `json:"api_endpoint"`
	Status           MemberStatus     `json:"status"`
	JoinDate         time.Time        `json:"join_date"`
	LastActivity     time.Time        `json:"last_activity"`

	// Performance metrics
	TotalVolume   uint64 `json:"total_volume"`
	AvgSpread     uint32 `json:"avg_spread"`
	UptimePercent uint16 `json:"uptime_percent"`
	RewardsEarned uint64 `json:"rewards_earned"`
}

// MemberStatus represents the status of an alliance member
type MemberStatus string

const (
	StatusActive    MemberStatus = "active"
	StatusInactive  MemberStatus = "inactive"
	StatusSuspended MemberStatus = "suspended"
	StatusPending   MemberStatus = "pending"
)

// NewAllianceAPI creates a new alliance coordination API
func NewAllianceAPI(liquidityManager *LiquidityManager) *AllianceAPI {
	return &AllianceAPI{
		liquidityManager: liquidityManager,
		attestorClient:   NewAttestorClient(),
		feeCalculator:    &SimpleFeeCalculator{},
		members:          make(map[string]*AllianceMember),
	}
}

// StartServer starts the alliance API HTTP server
func (api *AllianceAPI) StartServer(port int) error {
	mux := http.NewServeMux()

	// Core API endpoints
	mux.HandleFunc("/alliance/members", api.handleMembers)
	mux.HandleFunc("/alliance/rewards", api.handleRewards)
	mux.HandleFunc("/alliance/attestation", api.handleAttestation)
	mux.HandleFunc("/alliance/fees", api.handleFees)
	mux.HandleFunc("/alliance/status", api.handleStatus)

	// Market making endpoints
	mux.HandleFunc("/alliance/quotes", api.handleQuotes)
	mux.HandleFunc("/alliance/trades", api.handleTrades)
	mux.HandleFunc("/alliance/metrics", api.handleMetrics)

	// Administrative endpoints
	mux.HandleFunc("/alliance/register", api.handleRegister)
	mux.HandleFunc("/alliance/health", api.handleHealth)

	api.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	fmt.Printf("Starting Shell Alliance API server on port %d\n", port)
	return api.server.ListenAndServe()
}

// StopServer stops the alliance API server
func (api *AllianceAPI) StopServer() error {
	if api.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return api.server.Shutdown(ctx)
}

// handleMembers handles requests for alliance member information
func (api *AllianceAPI) handleMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.getMembers(w, r)
	case http.MethodPost:
		api.addMember(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getMembers returns the list of alliance members
func (api *AllianceAPI) getMembers(w http.ResponseWriter, r *http.Request) {
	api.membersMutex.RLock()
	defer api.membersMutex.RUnlock()

	members := make([]*AllianceMember, 0, len(api.members))
	for _, member := range api.members {
		members = append(members, member)
	}

	response := map[string]interface{}{
		"members": members,
		"count":   len(members),
	}

	api.writeJSONResponse(w, response)
}

// addMember adds a new alliance member
func (api *AllianceAPI) addMember(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name             string   `json:"name"`
		PublicKeyHex     string   `json:"public_key"`
		TradingAddresses []string `json:"trading_addresses"`
		APIEndpoint      string   `json:"api_endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate public key
	pubKeyBytes, err := parseHexSignature(request.PublicKeyHex)
	if err != nil {
		http.Error(w, "Invalid public key format", http.StatusBadRequest)
		return
	}

	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		http.Error(w, "Invalid public key", http.StatusBadRequest)
		return
	}

	// Create member ID from public key hash
	pubKeyHash := chainhash.HashB(pubKey.SerializeCompressed())
	memberID := fmt.Sprintf("%x", pubKeyHash[:8])

	member := &AllianceMember{
		ID:               memberID,
		Name:             request.Name,
		PublicKey:        pubKey,
		PublicKeyHex:     request.PublicKeyHex,
		TradingAddresses: request.TradingAddresses,
		APIEndpoint:      request.APIEndpoint,
		Status:           StatusPending,
		JoinDate:         time.Now(),
		LastActivity:     time.Now(),
	}

	api.membersMutex.Lock()
	api.members[memberID] = member
	api.membersMutex.Unlock()

	response := map[string]interface{}{
		"member_id": memberID,
		"status":    "pending",
		"message":   "Member registration submitted for review",
	}

	api.writeJSONResponse(w, response)
}

// handleRewards handles liquidity reward queries and claims
func (api *AllianceAPI) handleRewards(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.getRewards(w, r)
	case http.MethodPost:
		api.claimReward(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getRewards returns reward information for a member
func (api *AllianceAPI) getRewards(w http.ResponseWriter, r *http.Request) {
	memberID := r.URL.Query().Get("member_id")
	if memberID == "" {
		http.Error(w, "member_id required", http.StatusBadRequest)
		return
	}

	api.membersMutex.RLock()
	member, exists := api.members[memberID]
	api.membersMutex.RUnlock()

	if !exists {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	// Get current epoch information
	epochInfo, err := api.liquidityManager.GetEpochInfo(0)
	if err != nil {
		http.Error(w, "Failed to get epoch info", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"member_id":       memberID,
		"current_epoch":   epochInfo.Index,
		"rewards_earned":  member.RewardsEarned,
		"total_volume":    member.TotalVolume,
		"avg_spread":      member.AvgSpread,
		"uptime_percent":  member.UptimePercent,
		"epoch_pool":      epochInfo.RewardPool,
		"epoch_remaining": api.liquidityManager.GetRewardPoolRemaining(epochInfo.Index),
	}

	api.writeJSONResponse(w, response)
}

// claimReward processes a liquidity reward claim
func (api *AllianceAPI) claimReward(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MemberID        string   `json:"member_id"`
		EpochIndex      uint8    `json:"epoch_index"`
		AttestationBlob string   `json:"attestation_blob"`
		MerklePath      []string `json:"merkle_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate member
	api.membersMutex.RLock()
	_, exists := api.members[request.MemberID]
	api.membersMutex.RUnlock()

	if !exists {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	// Process the claim (simplified)
	response := map[string]interface{}{
		"success":   true,
		"member_id": request.MemberID,
		"epoch":     request.EpochIndex,
		"message":   "Reward claim submitted for processing",
		"tx_hash":   "pending", // Would be actual transaction hash
	}

	api.writeJSONResponse(w, response)
}

// handleAttestation provides attestation status and requests
func (api *AllianceAPI) handleAttestation(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.getAttestationStatus(w, r)
	case http.MethodPost:
		api.requestAttestation(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAttestationStatus returns the status of all attestors
func (api *AllianceAPI) getAttestationStatus(w http.ResponseWriter, r *http.Request) {
	status := api.attestorClient.GetAttestorStatus()

	response := map[string]interface{}{
		"attestors": status,
		"healthy":   api.countHealthyAttestors(status),
		"required":  MinAttestorSigs,
	}

	api.writeJSONResponse(w, response)
}

// requestAttestation requests attestation for a member
func (api *AllianceAPI) requestAttestation(w http.ResponseWriter, r *http.Request) {
	var request AttestationRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Request attestation from all attestors
	responses, err := api.attestorClient.RequestAttestation(&request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Attestation request failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"responses": len(responses),
		"required":  MinAttestorSigs,
		"message":   "Attestation request completed",
	}

	api.writeJSONResponse(w, response)
}

// handleFees provides fee calculation and estimation services
func (api *AllianceAPI) handleFees(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.getFeeRates(w, r)
	case http.MethodPost:
		api.calculateFees(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getFeeRates returns current fee rates
func (api *AllianceAPI) getFeeRates(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"base_fee_rate":    api.feeCalculator.GetFeeRate(),
		"maker_rebate":     api.feeCalculator.GetMakerRebateRate(),
		"channel_open":     ChannelOpenFee,
		"channel_update":   ChannelUpdateFee,
		"atomic_swap":      AtomicSwapFee,
		"claimable_create": ClaimableFee,
	}

	api.writeJSONResponse(w, response)
}

// calculateFees calculates fees for a transaction
func (api *AllianceAPI) calculateFees(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TransactionSize int  `json:"transaction_size"`
		HasShellOpcodes bool `json:"has_shell_opcodes"`
		IsMaker         bool `json:"is_maker"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	estimatedFee := api.feeCalculator.EstimateFee(
		request.TransactionSize,
		request.HasShellOpcodes,
		request.IsMaker,
	)

	response := map[string]interface{}{
		"estimated_fee": estimatedFee,
		"size":          request.TransactionSize,
		"is_maker":      request.IsMaker,
		"has_opcodes":   request.HasShellOpcodes,
	}

	api.writeJSONResponse(w, response)
}

// handleStatus provides general API status
func (api *AllianceAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	api.membersMutex.RLock()
	memberCount := len(api.members)
	api.membersMutex.RUnlock()

	attestorStatus := api.attestorClient.GetAttestorStatus()
	healthyAttestors := api.countHealthyAttestors(attestorStatus)

	response := map[string]interface{}{
		"status":            "operational",
		"version":           "2.2",
		"members":           memberCount,
		"healthy_attestors": healthyAttestors,
		"total_attestors":   len(KnownAttestors),
		"timestamp":         time.Now().Unix(),
	}

	api.writeJSONResponse(w, response)
}

// handleQuotes provides quote management for market makers
func (api *AllianceAPI) handleQuotes(w http.ResponseWriter, r *http.Request) {
	// Placeholder for quote management
	response := map[string]interface{}{
		"message": "Quote management endpoint - to be implemented",
		"method":  r.Method,
	}

	api.writeJSONResponse(w, response)
}

// handleTrades provides trade reporting for market makers
func (api *AllianceAPI) handleTrades(w http.ResponseWriter, r *http.Request) {
	// Placeholder for trade reporting
	response := map[string]interface{}{
		"message": "Trade reporting endpoint - to be implemented",
		"method":  r.Method,
	}

	api.writeJSONResponse(w, response)
}

// handleMetrics provides performance metrics for market makers
func (api *AllianceAPI) handleMetrics(w http.ResponseWriter, r *http.Request) {
	memberID := r.URL.Query().Get("member_id")
	if memberID == "" {
		http.Error(w, "member_id required", http.StatusBadRequest)
		return
	}

	api.membersMutex.RLock()
	member, exists := api.members[memberID]
	api.membersMutex.RUnlock()

	if !exists {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"member_id":      memberID,
		"total_volume":   member.TotalVolume,
		"avg_spread":     member.AvgSpread,
		"uptime_percent": member.UptimePercent,
		"rewards_earned": member.RewardsEarned,
		"last_activity":  member.LastActivity,
		"status":         member.Status,
	}

	api.writeJSONResponse(w, response)
}

// handleRegister handles member registration
func (api *AllianceAPI) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	api.addMember(w, r)
}

// handleHealth provides API health check
func (api *AllianceAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "2.2",
	}

	api.writeJSONResponse(w, response)
}

// Helper functions

// writeJSONResponse writes a JSON response to the HTTP writer
func (api *AllianceAPI) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// countHealthyAttestors counts the number of healthy attestors
func (api *AllianceAPI) countHealthyAttestors(status map[string]AttestorStatus) int {
	count := 0
	for _, s := range status {
		if s.Available {
			count++
		}
	}
	return count
}

// GetMember returns an alliance member by ID
func (api *AllianceAPI) GetMember(memberID string) (*AllianceMember, bool) {
	api.membersMutex.RLock()
	defer api.membersMutex.RUnlock()

	member, exists := api.members[memberID]
	return member, exists
}

// UpdateMemberStatus updates the status of an alliance member
func (api *AllianceAPI) UpdateMemberStatus(memberID string, status MemberStatus) error {
	api.membersMutex.Lock()
	defer api.membersMutex.Unlock()

	member, exists := api.members[memberID]
	if !exists {
		return errors.New("member not found")
	}

	member.Status = status
	member.LastActivity = time.Now()

	return nil
}

// UpdateMemberMetrics updates the performance metrics of an alliance member
func (api *AllianceAPI) UpdateMemberMetrics(memberID string, volume uint64, spread uint32, uptime uint16) error {
	api.membersMutex.Lock()
	defer api.membersMutex.Unlock()

	member, exists := api.members[memberID]
	if !exists {
		return errors.New("member not found")
	}

	member.TotalVolume += volume
	member.AvgSpread = (member.AvgSpread + spread) / 2 // Simple average
	member.UptimePercent = uptime
	member.LastActivity = time.Now()

	return nil
}
