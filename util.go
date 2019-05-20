package gosync

import "hash/adler32"

// ComputeWeakHash computes a weak hash
func ComputeWeakHash(v []byte) uint32 {
	return adler32.Checksum(v)
}
