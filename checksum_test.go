package gosync

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyReaders(t *testing.T) {
	r := &rsync{blockSize: 64 * 1024, strongHasher: md5.New(), fullChecksum: md5.New()}
	checksums, err := r.Sign(bytes.NewReader(nil))
	assert.NoError(t, err)
	assert.Len(t, checksums, 0)
}

func TestGenerateChecksums(t *testing.T) {
	data := make([]byte, 4*1024*1024)
	n, err := rand.Read(data)
	require.NoError(t, err)
	require.Equal(t, 4*1024*1024, n)

	r := &rsync{blockSize: 512 * 1024, strongHasher: md5.New(), fullChecksum: md5.New()}
	checksums, err := r.Sign(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Len(t, checksums, 8)
}

func TestGenerateChecksums2(t *testing.T) {
	data := make([]byte, 8520959)
	n, err := rand.Read(data)
	require.NoError(t, err)
	require.Equal(t, 8520959, n)

	r := &rsync{blockSize: 512 * 1024, strongHasher: sha256.New(), fullChecksum: md5.New()}
	checksums, err := r.Sign(bytes.NewReader(data))
	assert.NoError(t, err)
	checksum := checksums.Checksums[len(checksums.Checksums)-1]
	assert.NotEqual(t, 512*1024, checksum.Size())
}
