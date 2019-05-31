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

	checksums, err := r.Sign(dst)
	assert.NoError(t, err)
	assert.Len(t, checksums, 6)

	patcher, err := r.Delta(reader, 2, checksums)
	assert.NoError(t, err)

	err = r.Patch(dst, patcher.Found, patcher.Missing, output)
	assert.NoError(t, err)
	assert.Equal(t, src, output.Bytes())
}
