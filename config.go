package gosync

import (
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"time"

	"github.com/rkcloudchain/gosync/logging"
	"github.com/rkcloudchain/gosync/syncpb"
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

// BlockResolver is an interface used by the patchers to obtain blocks from the source.
type BlockResolver interface {
	RequestBlock(*syncpb.MissingBlockSpan) (*syncpb.BlockResponse, error)
}

// FileAccessor combines many of the interfaces that are needed
type FileAccessor interface {
	GetFileSize() (int64, error)
	GetFileModTime() (time.Time, error)
	ReadFile() (ReadSeekerAt, error)
	CopyFile(src io.Reader) error
}

// Config contains the parameters to start a gosync service.
type Config struct {
	// BlockSize force a fixed checksum block-size
	BlockSize int64

	// Logger is the logger used for gosync log.
	Logger logging.Logger

	// Directory for storing temporary files
	TempFileDir string

	// FileAccessor implementation
	FileAccessor FileAccessor

	// RequestUpgradeInterval determins frequency of gosync request update phases
	RequestUpdateInterval time.Duration

	// A hash function for calculating a strong checksum
	StrongHasher hash.Hash

	// MaxRequestBlockSize defines the maximum file block size for the remote transfer
	MaxRequestBlockSize int64

	// Resolver is an interface used by the patchers to obtain blocks from the source.
	Resolver BlockResolver
}

func (c *Config) validate() error {
	if c.BlockSize > maxBlockSize {
		return fmt.Errorf("Invalid block length %d", c.BlockSize)
	}

	if c.Resolver == nil {
		return errors.New("Block resolver must be specified")
	}

	if c.FileAccessor == nil {
		return errors.New("File accessor must be specified")
	}

	if c.BlockSize == 0 {
		c.BlockSize = defaultBlockSize
	}

	if c.TempFileDir == "" {
		c.TempFileDir = os.TempDir()
	}

	if c.Logger != nil {
		logging.SetLogger(c.Logger)
	}

	if c.RequestUpdateInterval == 0 {
		c.RequestUpdateInterval = 4 * time.Second
	}

	if c.StrongHasher == nil {
		c.StrongHasher = md5.New()
	}

	if c.MaxRequestBlockSize == 0 {
		c.MaxRequestBlockSize = defaultMaxRequestBlockSize
	}

	return nil
}
