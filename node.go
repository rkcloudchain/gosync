package gosync

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/rkcloudchain/gosync/logging"
	"github.com/rkcloudchain/gosync/syncpb"
)

// errors
var (
	ErrTargetNewer = errors.New("The target file is more recent than the source file")
)

func newNode(c *Config, r *rsync) *node {
	return &node{
		requestChan:     make(chan *syncpb.ChunkChecksums),
		stop:            make(chan struct{}, 1),
		r:               r,
		requestInterval: c.RequestUpdateInterval,
		fileAccessor:    c.FileAccessor,
		tempFileDir:     c.TempFileDir,
	}
}

// node represents a gosync node
type node struct {
	requestChan     chan *syncpb.ChunkChecksums
	stop            chan struct{}
	r               *rsync
	lock            sync.RWMutex
	requestInterval time.Duration
	fileAccessor    FileAccessor
	tempFileDir     string
}

func (n *node) SignReady() <-chan *syncpb.ChunkChecksums {
	return n.requestChan
}

func (n *node) Delta(checksums *syncpb.ChunkChecksums) (*syncpb.PatcherBlockSpan, error) {
	modTime, err := n.fileAccessor.GetFileModTime()
	if err != nil {
		return nil, err
	}
	modTime = modTime.UTC()

	if modTime.UnixNano() <= checksums.ModTime {
		return nil, ErrTargetNewer
	}

	reader, err := n.fileAccessor.ReadFile()
	if err != nil {
		return nil, err
	}

	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	result, err := n.r.Delta(reader, checksums.ConfigBlockSize, checksums.Checksums)
	if err != nil {
		return nil, err
	}

	result.ModTime = modTime.UnixNano()
	return result, nil
}

func (n *node) Patch(blocks *syncpb.PatcherBlockSpan) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	modTime, err := n.fileAccessor.GetFileModTime()
	if err != nil {
		return err
	}
	modTime = modTime.UTC()

	if modTime.UnixNano() > blocks.ModTime {
		return ErrTargetNewer
	}

	localFile, err := n.fileAccessor.ReadFile()
	if err != nil {
		return err
	}

	output, filename, err := n.createTempFile()
	if err != nil {
		return err
	}
	defer output.Close()
	defer os.Remove(filename)

	err = n.r.Patch(localFile, blocks.Found, blocks.Missing, output)
	if closer, ok := localFile.(io.Closer); ok {
		closer.Close()
	}
	if err != nil {
		return err
	}

	return n.fileAccessor.CopyFile(output)
}

func (n *node) Stop() {
	n.stop <- struct{}{}
}

func (n *node) run() {
	for {
		select {
		case <-time.After(n.requestInterval):
			checksums, err := n.createSignature()
			if err != nil {
				logging.Errorf("Failed creating signature: %s", err)
				continue
			}

			n.requestChan <- checksums

		case <-n.stop:
			logging.Info("Stop gosync service")
			return
		}
	}
}

func (n *node) createTempFile() (io.ReadWriteCloser, string, error) {
	ft, err := ioutil.TempFile(n.tempFileDir, "gosync_")
	return ft, ft.Name(), err
}

func (n *node) createSignature() (*syncpb.ChunkChecksums, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	reader, err := n.fileAccessor.ReadFile()
	if err != nil {
		return nil, err
	}

	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	checksums, err := n.r.Sign(reader)
	if err != nil {
		return nil, err
	}

	modTime, err := n.fileAccessor.GetFileModTime()
	if err != nil {
		return nil, err
	}
	modTime = modTime.UTC()

	return &syncpb.ChunkChecksums{ModTime: modTime.UnixNano(), Checksums: checksums, ConfigBlockSize: n.r.blockSize}, nil
}
