package blockchain

// SequenceLock represents a transaction sequence lock
type SequenceLock struct {
	Seconds     int64
	BlockHeight int32
}
