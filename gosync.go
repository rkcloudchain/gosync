package gosync

import (
	"io"

	"github.com/rkcloudchain/gosync/syncpb"
)

const (
	maxBlockSize = 1 << 17 // 128kb
)

// GoSync represents a rsync service
type GoSync interface {
	// Sign reads each block of the input file, and returns the checksums for each block.
	Sign(dest io.Reader) (*syncpb.ChunkChecksums, error)

	Delta(source io.ReadSeeker, checksums *syncpb.ChunkChecksums)
}
