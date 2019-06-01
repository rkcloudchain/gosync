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

// New returns a new gosync instance given configuration.
func New(c *Config) (GoSync, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	r := newRSync(c)
	return r, nil
}
