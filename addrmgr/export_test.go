// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// This file exports internal functions for use in tests.
// It is compiled only when running tests.

package addrmgr

import (
	"time"

	"github.com/toole-brendan/shell/wire"
)

// TstKnownAddressIsBad makes the internal isBad method available to tests.
func TstKnownAddressIsBad(ka *KnownAddress) bool {
	return ka.isBad()
}

// TstKnownAddressChance makes the internal chance method available to tests.
func TstKnownAddressChance(ka *KnownAddress) float64 {
	return ka.chance()
}

// TstNewKnownAddress makes the internal KnownAddress constructor available to tests.
func TstNewKnownAddress(na *wire.NetAddressV2, attempts int,
	lastattempt, lastsuccess time.Time, tried bool, refs int) *KnownAddress {
	return &KnownAddress{na: na, attempts: attempts, lastattempt: lastattempt,
		lastsuccess: lastsuccess, tried: tried, refs: refs}
}
