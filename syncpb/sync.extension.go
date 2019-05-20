package syncpb

import "bytes"

// Match compares a chunksum to another based on the chunksums
func (chunk *ChunkChecksum) Match(other *ChunkChecksum) bool {
	weakEqual := chunk.WeakHash == other.WeakHash
	strongEqual := false
	if weakEqual {
		strongEqual = bytes.Compare(chunk.StrongHash, other.StrongHash) == 0
	}

	return weakEqual && strongEqual
}
