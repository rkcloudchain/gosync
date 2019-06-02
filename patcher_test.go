package gosync

import (
	"bytes"
	"crypto/sha256"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// NewReadSeekerRequester ...
func NewReadSeekerRequester(r io.ReadSeeker) BlockRequester {
	return &ReadSeekerRequester{rs: r}
}

// ReadSeekerRequester ...
type ReadSeekerRequester struct {
	rs io.ReadSeeker
}

// DoRequest ...
func (r *ReadSeekerRequester) DoRequest(startOffset int64, endOffset int64) (data []byte, err error) {
	length := endOffset - startOffset + 1
	buffer := make([]byte, length)

	if _, err = r.rs.Seek(startOffset, io.SeekStart); err != nil {
		return
	}

	n, err := io.ReadFull(r.rs, buffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		return
	}

	return buffer[:n], nil
}

func TestPatch(t *testing.T) {
	dst := bytes.NewReader([]byte("hello world"))
	src := []byte("Hello world: xqlun")
	reader := bytes.NewReader(src)

	output := bytes.NewBuffer(nil)

	r := &rsync{
		blockSize:        2,
		strongHasher:     sha256.New(),
		requestBlockSize: 4,
		sizeFunc:         func() (int64, error) { return int64(len(src)), nil },
		reference:        NewReadSeekerRequester(reader),
	}

	checksums := r.Sign(dst)
	assert.Len(t, checksums.Checksums, 6)

	patcher, err := r.Delta(reader, checksums)
	assert.NoError(t, err)

	err = r.Patch(dst, patcher, output)
	assert.NoError(t, err)
	assert.Equal(t, src, output.Bytes())
}

func TestPatchWithLocalNil(t *testing.T) {
	local := bytes.NewReader(nil)
	reference := []byte("Raft is a consensus algorithm that is designed to be easy to understand. It's equivalent to Paxos in fault-tolerance and performance.")
	reader := bytes.NewReader(reference)

	r, err := New(&Config{BlockSize: 4, StrongHasher: sha256.New(), MaxRequestBlockSize: 128, Requester: NewReadSeekerRequester(reader), SizeFunc: func() (int64, error) { return int64(len(reference)), nil }})
	assert.NoError(t, err)

	checksums := r.Sign(local)
	assert.Len(t, checksums.Checksums, 0)

	patcher, err := r.Delta(reader, checksums)
	assert.NoError(t, err)
	assert.Len(t, patcher.Found, 0)

	output := bytes.NewBuffer(nil)
	err = r.Patch(local, patcher, output)
	assert.NoError(t, err)
	assert.Equal(t, reference, output.Bytes())
}
