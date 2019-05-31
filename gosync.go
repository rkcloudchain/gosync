package gosync

import (
	"io"

	"github.com/rkcloudchain/gosync/syncpb"
)

// GoSync represents a rsync service
type GoSync interface {
	Sign(io.Reader) (*syncpb.ChunkChecksums, error)

	Delta(io.ReaderAt, *syncpb.ChunkChecksums) (*syncpb.PatcherBlockSpan, error)

	Patch(io.ReadSeeker, *syncpb.PatcherBlockSpan, io.Writer) error
}

// Start returns a new gosync instance given configuration.
func Start(c *Config) GoSync {
	if err := c.validate(); err != nil {
		panic(err.Error())
	}

	r := newRSync(c)
	return r
}
