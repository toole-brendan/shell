// Package liquidity implements Shell Reserve's attestor integration system
// for validating market making data from authorized providers.
package liquidity

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/toole-brendan/shell/chaincfg/chainhash"
)

// AttestorClient manages communication with authorized market data providers
type AttestorClient struct {
	httpClient *http.Client
	attestors  []AttestorInfo
}

// NewAttestorClient creates a new attestor client
func NewAttestorClient() *AttestorClient {
	return &AttestorClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		attestors: KnownAttestors,
	}
}

// MarketMakingData represents market making metrics from an attestor
type MarketMakingData struct {
	ParticipantID [32]byte `json:"participant_id"`
	EpochIndex    uint8    `json:"epoch_index"`

	// Trading metrics
	Volume        uint64 `json:"volume"`         // Total volume in satoshis
	TradeCount    uint32 `json:"trade_count"`    // Number of trades
	AverageSpread uint32 `json:"average_spread"` // Average spread in basis points

	// Uptime metrics
	UptimePercent uint16 `json:"uptime_percent"` // Uptime in basis points (0-10000)
	ResponseTime  uint32 `json:"response_time"`  // Average response time in milliseconds

	// Quality metrics
	QuoteDepth  uint64 `json:"quote_depth"`  // Average quote depth
	TightSpread uint16 `json:"tight_spread"` // Percentage of time at tight spread

	// Attestation metadata
	StartTime    uint32 `json:"start_time"`    // Epoch start timestamp
	EndTime      uint32 `json:"end_time"`      // Epoch end timestamp
	AttestorName string `json:"attestor_name"` // Name of attestor
	Signature    string `json:"signature"`     // Attestor signature (hex)
}

// AttestationRequest represents a request for market making attestation
type AttestationRequest struct {
	ParticipantID    [32]byte `json:"participant_id"`
	EpochIndex       uint8    `json:"epoch_index"`
	TradingAddresses []string `json:"trading_addresses"` // Addresses to analyze
	APIKeys          []string `json:"api_keys"`          // Exchange API keys (if provided)
}

// AttestationResponse represents the response from an attestor
type AttestationResponse struct {
	Success    bool              `json:"success"`
	Data       *MarketMakingData `json:"data,omitempty"`
	Error      string            `json:"error,omitempty"`
	AttestorID int               `json:"attestor_id"`
	Timestamp  uint32            `json:"timestamp"`
	Signature  string            `json:"signature"`
}

// RequestAttestation requests market making data attestation from all attestors
func (ac *AttestorClient) RequestAttestation(request *AttestationRequest) ([]*AttestationResponse, error) {
	if request == nil {
		return nil, errors.New("attestation request cannot be nil")
	}

	responses := make([]*AttestationResponse, 0, len(ac.attestors))

	// Request attestation from each authorized attestor
	for i, attestor := range ac.attestors {
		response, err := ac.requestFromAttestor(i, attestor, request)
		if err != nil {
			// Log error but continue with other attestors
			fmt.Printf("Attestor %s failed: %v\n", attestor.Name, err)
			continue
		}
		responses = append(responses, response)
	}

	if len(responses) < MinAttestorSigs {
		return nil, fmt.Errorf("insufficient attestor responses: got %d, need %d",
			len(responses), MinAttestorSigs)
	}

	return responses, nil
}

// requestFromAttestor makes an HTTP request to a specific attestor
func (ac *AttestorClient) requestFromAttestor(attestorID int, attestor AttestorInfo, request *AttestationRequest) (*AttestationResponse, error) {
	// Convert request to JSON
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/attestation", attestor.Endpoint)
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("User-Agent", "Shell-Reserve/2.2")

	// Make the request
	httpResponse, err := ac.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer httpResponse.Body.Close()

	// Read response
	responseData, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse response
	var response AttestationResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Verify attestor signature
	if err := ac.verifyAttestorSignature(attestorID, &response); err != nil {
		return nil, fmt.Errorf("signature verification failed: %v", err)
	}

	response.AttestorID = attestorID
	return &response, nil
}

// verifyAttestorSignature verifies the digital signature from an attestor
func (ac *AttestorClient) verifyAttestorSignature(attestorID int, response *AttestationResponse) error {
	if attestorID >= len(ac.attestors) {
		return errors.New("invalid attestor ID")
	}

	attestor := ac.attestors[attestorID]
	if attestor.PublicKey == nil {
		return errors.New("attestor public key not available")
	}

	if response.Signature == "" {
		return errors.New("missing attestor signature")
	}

	// Create hash of the response data for signature verification
	dataHash := ac.hashResponseData(response)

	// Parse signature from hex
	sigBytes, err := parseHexSignature(response.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature format: %v", err)
	}

	// Verify signature
	signature, err := ecdsa.ParseSignature(sigBytes)
	if err != nil {
		return fmt.Errorf("failed to parse signature: %v", err)
	}

	if !signature.Verify(dataHash[:], attestor.PublicKey) {
		return errors.New("signature verification failed")
	}

	return nil
}

// hashResponseData creates a deterministic hash of response data for signature verification
func (ac *AttestorClient) hashResponseData(response *AttestationResponse) chainhash.Hash {
	data := make([]byte, 0, 256)

	if response.Data != nil {
		// Include all market making data fields
		data = append(data, response.Data.ParticipantID[:]...)
		data = append(data, response.Data.EpochIndex)

		// Add volume (8 bytes, little endian)
		volumeBytes := make([]byte, 8)
		for i := 0; i < 8; i++ {
			volumeBytes[i] = byte(response.Data.Volume >> (i * 8))
		}
		data = append(data, volumeBytes...)

		// Add other metrics
		spreadBytes := make([]byte, 4)
		for i := 0; i < 4; i++ {
			spreadBytes[i] = byte(response.Data.AverageSpread >> (i * 8))
		}
		data = append(data, spreadBytes...)

		uptimeBytes := make([]byte, 2)
		for i := 0; i < 2; i++ {
			uptimeBytes[i] = byte(response.Data.UptimePercent >> (i * 8))
		}
		data = append(data, uptimeBytes...)
	}

	// Add timestamp
	timestampBytes := make([]byte, 4)
	for i := 0; i < 4; i++ {
		timestampBytes[i] = byte(response.Timestamp >> (i * 8))
	}
	data = append(data, timestampBytes...)

	hash := sha256.Sum256(data)
	return chainhash.Hash(hash)
}

// CreateAttestation creates a signed liquidity attestation from multiple attestor responses
func (ac *AttestorClient) CreateAttestation(responses []*AttestationResponse) (*LiquidityAttestation, error) {
	if len(responses) < MinAttestorSigs {
		return nil, fmt.Errorf("insufficient attestor responses: got %d, need %d",
			len(responses), MinAttestorSigs)
	}

	// Use data from the first successful response as base
	var baseData *MarketMakingData
	for _, response := range responses {
		if response.Success && response.Data != nil {
			baseData = response.Data
			break
		}
	}

	if baseData == nil {
		return nil, errors.New("no valid attestor data found")
	}

	// Create attestation with aggregated data
	attestation := &LiquidityAttestation{
		EpochIndex:    baseData.EpochIndex,
		ParticipantID: baseData.ParticipantID,
		Volume:        baseData.Volume,
		Spread:        baseData.AverageSpread,
		Uptime:        baseData.UptimePercent,
		Timestamp:     uint32(time.Now().Unix()),
	}

	// Collect signatures from successful attestors
	attestation.AttestorSigs = make([]ecdsa.Signature, 0, len(responses))
	for _, response := range responses {
		if response.Success && response.Signature != "" {
			sigBytes, err := parseHexSignature(response.Signature)
			if err != nil {
				continue
			}

			signature, err := ecdsa.ParseSignature(sigBytes)
			if err != nil {
				continue
			}

			attestation.AttestorSigs = append(attestation.AttestorSigs, *signature)
		}
	}

	return attestation, nil
}

// ValidateAttestorCoverage checks that we have sufficient attestor coverage
func (ac *AttestorClient) ValidateAttestorCoverage(responses []*AttestationResponse) error {
	if len(responses) < MinAttestorSigs {
		return fmt.Errorf("insufficient attestor responses: got %d, need %d",
			len(responses), MinAttestorSigs)
	}

	// Check for attestor diversity (at least 3 different providers)
	uniqueAttestors := make(map[int]bool)
	successfulResponses := 0

	for _, response := range responses {
		if response.Success {
			uniqueAttestors[response.AttestorID] = true
			successfulResponses++
		}
	}

	if successfulResponses < MinAttestorSigs {
		return fmt.Errorf("insufficient successful attestations: got %d, need %d",
			successfulResponses, MinAttestorSigs)
	}

	if len(uniqueAttestors) < 3 {
		return fmt.Errorf("insufficient attestor diversity: got %d unique attestors, need 3",
			len(uniqueAttestors))
	}

	return nil
}

// GetAttestorStatus returns the current status of all attestors
func (ac *AttestorClient) GetAttestorStatus() map[string]AttestorStatus {
	status := make(map[string]AttestorStatus)

	for _, attestor := range ac.attestors {
		status[attestor.Name] = ac.checkAttestorHealth(attestor)
	}

	return status
}

// AttestorStatus represents the health status of an attestor
type AttestorStatus struct {
	Name         string        `json:"name"`
	Endpoint     string        `json:"endpoint"`
	Available    bool          `json:"available"`
	ResponseTime time.Duration `json:"response_time"`
	LastCheck    time.Time     `json:"last_check"`
	Error        string        `json:"error,omitempty"`
}

// checkAttestorHealth performs a health check on an attestor
func (ac *AttestorClient) checkAttestorHealth(attestor AttestorInfo) AttestorStatus {
	status := AttestorStatus{
		Name:      attestor.Name,
		Endpoint:  attestor.Endpoint,
		LastCheck: time.Now(),
	}

	// Perform health check
	start := time.Now()
	url := fmt.Sprintf("%s/health", attestor.Endpoint)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to create request: %v", err)
		return status
	}

	request.Header.Set("User-Agent", "Shell-Reserve/2.2")

	response, err := ac.httpClient.Do(request)
	if err != nil {
		status.Error = fmt.Sprintf("Request failed: %v", err)
		return status
	}
	defer response.Body.Close()

	status.ResponseTime = time.Since(start)
	status.Available = response.StatusCode == http.StatusOK

	if !status.Available {
		status.Error = fmt.Sprintf("HTTP %d", response.StatusCode)
	}

	return status
}

// parseHexSignature parses a hex-encoded signature
func parseHexSignature(hexSig string) ([]byte, error) {
	if len(hexSig) < 2 || hexSig[:2] != "0x" {
		return nil, errors.New("signature must start with 0x")
	}

	// Remove 0x prefix and parse hex
	sigBytes := make([]byte, 0, (len(hexSig)-2)/2)
	for i := 2; i < len(hexSig); i += 2 {
		if i+1 >= len(hexSig) {
			break
		}

		var b byte
		_, err := fmt.Sscanf(hexSig[i:i+2], "%02x", &b)
		if err != nil {
			return nil, fmt.Errorf("invalid hex byte: %s", hexSig[i:i+2])
		}
		sigBytes = append(sigBytes, b)
	}

	return sigBytes, nil
}
