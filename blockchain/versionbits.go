// Copyright (c) 2016-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/toole-brendan/shell/chaincfg"
)

const (
	// vbLegacyBlockVersion is the highest legacy block version before the
	// version bits scheme became active.
	vbLegacyBlockVersion = 4

	// vbTopBits defines the bits to set in the version to signal that the
	// version bits scheme is being used.
	vbTopBits = 0x20000000

	// vbTopMask is the bitmask to use to determine whether or not the
	// version bits scheme is in use.
	vbTopMask = 0xe0000000

	// vbNumBits is the total number of bits available for use with the
	// version bits scheme.
	vbNumBits = 29
)

// bitConditionChecker provides a thresholdConditionChecker which can be used to
// test whether or not a specific bit is set when it's not supposed to be
// according to the expected version based on the known deployments and the
// current state of the chain.  This is useful for detecting and warning about
// unknown rule activations.
type bitConditionChecker struct {
	bit   uint32
	chain *BlockChain
}

// Ensure the bitConditionChecker type implements the thresholdConditionChecker
// interface.
var _ thresholdConditionChecker = bitConditionChecker{}

// HasStarted returns true if based on the passed block blockNode the consensus
// is eligible for deployment.
//
// Since this implementation checks for unknown rules, it returns true so
// is always treated as active.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) HasStarted(_ *blockNode) bool {
	return true
}

// HasStarted returns true if based on the passed block blockNode the consensus
// is eligible for deployment.
//
// Since this implementation checks for unknown rules, it returns false so the
// rule is always treated as active.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) HasEnded(_ *blockNode) bool {
	return false
}

// RuleChangeActivationThreshold is the number of blocks for which the condition
// must be true in order to lock in a rule change.
//
// This implementation returns the value defined by the chain params the checker
// is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) RuleChangeActivationThreshold() uint32 {
	return c.chain.chainParams.RuleChangeActivationThreshold
}

// MinerConfirmationWindow is the number of blocks in each threshold state
// retarget window.
//
// This implementation returns the value defined by the chain params the checker
// is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) MinerConfirmationWindow() uint32 {
	return c.chain.chainParams.MinerConfirmationWindow
}

// Condition returns true when the specific bit associated with the checker is
// set and it's not supposed to be according to the expected version based on
// the known deployments and the current state of the chain.
//
// This function MUST be called with the chain state lock held (for writes).
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) Condition(node *blockNode) (bool, error) {
	conditionMask := uint32(1) << c.bit
	version := uint32(node.version)
	if version&vbTopMask != vbTopBits {
		return false, nil
	}
	if version&conditionMask == 0 {
		return false, nil
	}

	expectedVersion, err := c.chain.calcNextBlockVersion(node.parent)
	if err != nil {
		return false, err
	}
	return uint32(expectedVersion)&conditionMask == 0, nil
}

// EligibleToActivate returns true if a custom deployment can transition from
// the LockedIn to the Active state. For normal deployments, this always
// returns true. However, some deployments add extra rules like a minimum
// activation height, which can be abstracted into a generic arbitrary check at
// the final state via this method.
//
// This implementation always returns true, as it's used to warn about other
// unknown deployments.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) EligibleToActivate(blkNode *blockNode) bool {
	return true
}

// IsSpeedy returns true if this is to be a "speedy" deployment. A speedy
// deployment differs from a regular one in that only after a miner block
// confirmation window can the deployment expire.
//
// This implementation returns false, as we want to always be warned if
// something is about to activate.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c bitConditionChecker) IsSpeedy() bool {
	return false
}

// ForceActive returns if the deployment should be forced to transition to the
// active state. This is useful on certain testnet, where we we'd like for a
// deployment to always be active.
func (c bitConditionChecker) ForceActive(node *blockNode) bool {
	return false
}

// deploymentChecker provides a thresholdConditionChecker which can be used to
// test a specific deployment rule.  This is required for properly detecting
// and activating consensus rule changes.
type deploymentChecker struct {
	deployment *chaincfg.ConsensusDeployment
	chain      *BlockChain
}

// Ensure the deploymentChecker type implements the thresholdConditionChecker
// interface.
var _ thresholdConditionChecker = deploymentChecker{}

// HasEnded returns true if the target consensus rule change has expired
// or timed out (at the next window).
//
// This implementation returns the value defined by the specific deployment the
// checker is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) HasStarted(blkNode *blockNode) bool {
	// Can't fail as we make sure to set the clock above when we
	// instantiate *BlockChain.
	header := blkNode.Header()
	started, _ := c.deployment.DeploymentStarter.HasStarted(&header)

	return started
}

// HasEnded returns true if the target consensus rule change has expired
// or timed out.
//
// This implementation returns the value defined by the specific deployment the
// checker is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) HasEnded(blkNode *blockNode) bool {
	// Can't fail as we make sure to set the clock above when we
	// instantiate *BlockChain.
	header := blkNode.Header()
	ended, _ := c.deployment.DeploymentEnder.HasEnded(&header)

	return ended
}

// RuleChangeActivationThreshold is the number of blocks for which the condition
// must be true in order to lock in a rule change.
//
// This implementation returns the value defined by the chain params the checker
// is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) RuleChangeActivationThreshold() uint32 {
	// Some deployments like taproot used a custom activation threshold
	// that overrides the network level threshold.
	if c.deployment.CustomActivationThreshold != 0 {
		return c.deployment.CustomActivationThreshold
	}

	return c.chain.chainParams.RuleChangeActivationThreshold
}

// MinerConfirmationWindow is the number of blocks in each threshold state
// retarget window.
//
// This implementation returns the value defined by the chain params the checker
// is associated with.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) MinerConfirmationWindow() uint32 {
	return c.chain.chainParams.MinerConfirmationWindow
}

// EligibleToActivate returns true if a custom deployment can transition from
// the LockedIn to the Active state. In addition to the traditional minimum
// activation height (MinActivationHeight), an optional AlwaysActiveHeight can
// force the deployment to be active after a specified height.
func (c deploymentChecker) EligibleToActivate(blkNode *blockNode) bool {
	// No activation height, so it's always ready to go.
	if c.deployment.MinActivationHeight == 0 {
		return true
	}

	// If the _next_ block (as this is the prior block to the one being
	// connected is the min height or beyond, then this can activate.
	return uint32(blkNode.height)+1 >= c.deployment.MinActivationHeight
}

// IsSpeedy returns true if this is to be a "speedy" deployment. A speedy
// deployment differs from a regular one in that only after a miner block
// confirmation window can the deployment expire.  This implementation returns
// true if a min activation height is set.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) IsSpeedy() bool {
	return (c.deployment.MinActivationHeight != 0 ||
		c.deployment.CustomActivationThreshold != 0)
}

// Condition returns true when the specific bit defined by the deployment
// associated with the checker is set.
//
// This is part of the thresholdConditionChecker interface implementation.
func (c deploymentChecker) Condition(node *blockNode) (bool, error) {
	conditionMask := uint32(1) << c.deployment.BitNumber
	version := uint32(node.version)
	return (version&vbTopMask == vbTopBits) && (version&conditionMask != 0),
		nil
}

// ForceActive returns if the deployment should be forced to transition to the
// active state. This is useful on certain testnet, where we we'd like for a
// deployment to always be active.
func (c deploymentChecker) ForceActive(node *blockNode) bool {
	if node == nil {
		return false
	}

	// If the deployment has a nonzero AlwaysActiveHeight and the next
	// block’s height is at or above that threshold, then force the state
	// to Active.
	effectiveHeight := c.deployment.EffectiveAlwaysActiveHeight()
	if uint32(node.height)+1 >= effectiveHeight {
		log.Debugf("Force activating deployment: next block "+
			"height %d >= EffectiveAlwaysActiveHeight %d",
			uint32(node.height)+1, effectiveHeight)
		return true
	}

	return false
}

// calcNextBlockVersion calculates the expected version of the block after the
// passed previous block node based on the state of started and locked in
// rule change deployments.
//
// This function differs from the exported CalcNextBlockVersion in that the
// exported version uses the current best chain as the previous block node
// while this function accepts any block node.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) calcNextBlockVersion(prevNode *blockNode) (int32, error) {
	// Set the appropriate bits for each actively defined rule deployment
	// that is either in the process of being voted on, or locked in for the
	// activation at the next threshold window change.
	expectedVersion := uint32(vbTopBits)
	for id := 0; id < len(b.chainParams.Deployments); id++ {
		deployment := &b.chainParams.Deployments[id]
		cache := &b.deploymentCaches[id]
		checker := deploymentChecker{deployment: deployment, chain: b}
		state, err := b.thresholdState(prevNode, checker, cache)
		if err != nil {
			return 0, err
		}
		if state == ThresholdStarted || state == ThresholdLockedIn {
			expectedVersion |= uint32(1) << deployment.BitNumber
		}
	}
	return int32(expectedVersion), nil
}

// CalcNextBlockVersion calculates the expected version of the block after the
// end of the current best chain based on the state of started and locked in
// rule change deployments.
//
// This function is safe for concurrent access.
func (b *BlockChain) CalcNextBlockVersion() (int32, error) {
	b.chainLock.Lock()
	version, err := b.calcNextBlockVersion(b.bestChain.Tip())
	b.chainLock.Unlock()
	return version, err
}

// warnUnknownRuleActivations displays a warning when any unknown new rules are
// either about to activate or have been activated.  This will only happen once
// when new rules have been activated and every block for those about to be
// activated.
//
// This function MUST be called with the chain state lock held (for writes)
func (b *BlockChain) warnUnknownRuleActivations(node *blockNode) error {
	// Warn if any unknown new rules are either about to activate or have
	// already been activated.
	for bit := uint32(0); bit < vbNumBits; bit++ {
		checker := bitConditionChecker{bit: bit, chain: b}
		cache := &b.warningCaches[bit]
		state, err := b.thresholdState(node.parent, checker, cache)
		if err != nil {
			return err
		}

		switch state {
		case ThresholdActive:
			if !b.unknownRulesWarned {
				log.Warnf("Unknown new rules activated (bit %d)",
					bit)
				b.unknownRulesWarned = true
			}

		case ThresholdLockedIn:
			window := int32(checker.MinerConfirmationWindow())
			activationHeight := window - (node.height % window)
			log.Warnf("Unknown new rules are about to activate in "+
				"%d blocks (bit %d)", activationHeight, bit)
		}
	}

	return nil
}
