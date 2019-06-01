package gosync

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorConfig(t *testing.T) {
	c := &Config{BlockSize: 512 * 1024 * 1024}
	err := c.validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid block length")

	c = &Config{BlockSize: 4}
	err = c.validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Block requester must be specified")

	c = &Config{BlockSize: 4, Requester: NewReadSeekerRequester(bytes.NewReader([]byte("")))}
	err = c.validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "File size function must be specified")

	c = &Config{Requester: NewReadSeekerRequester(bytes.NewReader([]byte(""))), SizeFunc: func() (int64, error) { return 0, nil }}
	err = c.validate()
	assert.NoError(t, err)
	assert.Equal(t, int64(defaultBlockSize), c.BlockSize)
	assert.NotNil(t, c.StrongHasher)
	assert.Equal(t, int64(defaultMaxRequestBlockSize), c.MaxRequestBlockSize)
}

func TestNewWithErrorConfig(t *testing.T) {
	c := &Config{}
	_, err := New(c)
	assert.Error(t, err)
}

func TestGoRSync(t *testing.T) {
	local := bytes.NewReader([]byte("The qwik brown fox jumped 0v3r the lazy"))
	reference := []byte("The quick brown fox jumped over the lazy dog")
	reader := bytes.NewReader(reference)

	cfg := &Config{
		BlockSize:           4,
		StrongHasher:        sha256.New(),
		MaxRequestBlockSize: 16,
		Requester:           NewReadSeekerRequester(reader),
		SizeFunc:            func() (int64, error) { return int64(len(reference)), nil },
	}

	r, err := New(cfg)
	assert.NoError(t, err)

	checksums, err := r.Sign(local)
	assert.NoError(t, err)

	patcher, err := r.Delta(reader, checksums)
	assert.NoError(t, err)

	output := bytes.NewBuffer(nil)
	err = r.Patch(local, patcher, output)
	assert.NoError(t, err)
	assert.Equal(t, reference, output.Bytes())
}
