package gosync

import (
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"io"

	"github.com/rkcloudchain/gosync/logging"
)

const (
	maxBlockSize               = 128 * 1024
	defaultBlockSize           = 64 * 1024
	defaultMaxRequestBlockSize = 512 * 1024
)

// ReadSeekerAt is the combination of ReadSeeker and ReaderAt interfaces
type ReadSeekerAt interface {
	io.ReadSeeker
	io.ReaderAt
}

// BlockRequester does synchronous requests on a remote source of blocks
type BlockRequester interface {
	DoRequest(startOffset int64, enfOffset int64) (data []byte, err error)
}

// Config contains the parameters to start a gosync service.
type Config struct {
	// BlockSize force a fixed checksum block-size
	BlockSize int64

	// Logger is the logger used for gosync log.
	Logger logging.Logger

	// A hash function for calculating a strong checksum
	StrongHasher hash.Hash

	// MaxRequestBlockSize defines the maximum file block size for the remote transfer
	MaxRequestBlockSize int64

	// Resolver is an interface used by the patchers to obtain blocks from the source.
	Requester BlockRequester

	// Function for getting the file size
	SizeFunc func() (int64, error)
}

func (c *Config) validate() error {
	if c.BlockSize > maxBlockSize {
		return fmt.Errorf("Invalid block length %d", c.BlockSize)
	}

	if c.Requester == nil {
		return errors.New("Block requester must be specified")
	}

	if c.SizeFunc == nil {
		return errors.New("File size function must be specified")
	}

	if c.BlockSize == 0 {
		c.BlockSize = defaultBlockSize
	}

	if c.Logger != nil {
		logging.SetLogger(c.Logger)
	}

	if c.StrongHasher == nil {
		c.StrongHasher = md5.New()
	}

	if c.MaxRequestBlockSize == 0 {
		c.MaxRequestBlockSize = defaultMaxRequestBlockSize
	}

	return nil
}
