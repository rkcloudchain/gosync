package gosync

import (
	"github.com/rkcloudchain/gosync/syncpb"
)

// GoSync represents a rsync service
type GoSync interface {
	Delta(*syncpb.ChunkChecksums) (*syncpb.PatcherBlockSpan, error)

	Patch(*syncpb.PatcherBlockSpan) error

	SignReady() <-chan *syncpb.ChunkChecksums

	Stop()
}

// Start returns a new gosync instance given configuration.
func Start(c *Config) GoSync {
	if err := c.validate(); err != nil {
		panic(err.Error())
	}

	r := newRSync(c)
	n := newNode(c, r)
	go n.run()
	return n
}
