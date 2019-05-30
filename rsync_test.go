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
	r := &rsync{blockSize: 64 * 1024, strongHasher: md5.New()}
	checksums, err := r.Sign(bytes.NewReader(nil))
	assert.NoError(t, err)
	assert.Len(t, checksums, 0)
}

func TestGenerateChecksums(t *testing.T) {
	data := make([]byte, 4*1024*1024)
	n, err := rand.Read(data)
	require.NoError(t, err)
	require.Equal(t, 4*1024*1024, n)

	r := &rsync{blockSize: 512 * 1024, strongHasher: md5.New()}
	checksums, err := r.Sign(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Len(t, checksums, 8)
}

func TestGenerateChecksums2(t *testing.T) {
	data := make([]byte, 8520959)
	n, err := rand.Read(data)
	require.NoError(t, err)
	require.Equal(t, 8520959, n)

	r := &rsync{blockSize: 512 * 1024, strongHasher: sha256.New()}
	checksums, err := r.Sign(bytes.NewReader(data))
	assert.NoError(t, err)
	checksum := checksums[len(checksums)-1]
	assert.NotEqual(t, 512*1024, checksum.Size())
}

func TestGenerateChecksums3(t *testing.T) {
	r := &rsync{blockSize: 2, strongHasher: md5.New()}
	checksums, err := r.Sign(bytes.NewReader([]byte("hello world")))
	assert.NoError(t, err)
	assert.Len(t, checksums, 6)
	assert.Equal(t, int64(1), checksums[5].BlockSize)
}

func TestMatch(t *testing.T) {
	reader := bytes.NewReader([]byte("123abcdefg"))
	r := &rsync{blockSize: 3, strongHasher: md5.New()}

	checksums, err := r.Sign(reader)
	assert.NoError(t, err)
	assert.Len(t, checksums, 4)

	source := bytes.NewReader([]byte("123xxabc def"))
	results, err := r.match(source, r.blockSize, checksums)
	assert.NoError(t, err)
	assert.Len(t, results, 3)

	assert.Equal(t, int64(0), results[0].ComparisonOffset)
	assert.Equal(t, int64(5), results[1].ComparisonOffset)
	assert.Equal(t, int64(9), results[2].ComparisonOffset)
}

func TestMatch2(t *testing.T) {
	reader := bytes.NewReader([]byte("hello"))
	r := newRSync(2, sha256.New(), func() (int64, error) { return int64(2), nil })

	checksums, err := r.Sign(reader)
	assert.NoError(t, err)
	assert.Len(t, checksums, 3)

	source := bytes.NewReader([]byte("helllo"))
	results, err := r.match(source, 2, checksums)
	assert.NoError(t, err)
	assert.Len(t, results, 3)

	assert.Equal(t, int64(0), results[0].ComparisonOffset)
	assert.Equal(t, int64(2), results[1].ComparisonOffset)
	assert.Equal(t, int64(5), results[2].ComparisonOffset)
}

func TestDelta(t *testing.T) {
	dst := bytes.NewReader([]byte("aabbccddeeffgg"))

	bs := []byte("123aabb456ccdd789ee321ff21gg")
	src := bytes.NewReader(bs)

	r := newRSync(4, md5.New(), func() (int64, error) { return int64(len(bs)), nil })
	checksums, err := r.Sign(dst)
	assert.NoError(t, err)
	assert.Len(t, checksums, 4)

	patcher, err := r.Delta(src, 4, checksums)
	assert.NoError(t, err)
	assert.Len(t, patcher.Found, 3)
	assert.Len(t, patcher.Missing, 3)

	assert.Equal(t, uint32(0), patcher.Found[0].StartIndex)
	assert.Equal(t, uint32(0), patcher.Found[0].EndIndex)
	assert.Equal(t, uint32(1), patcher.Found[1].StartIndex)
	assert.Equal(t, uint32(1), patcher.Found[1].EndIndex)
	assert.Equal(t, uint32(2), patcher.Found[3].StartIndex)
	assert.Equal(t, uint32(2), patcher.Found[3].StartIndex)

	assert.Equal(t, int64(0), patcher.Missing[0].StartOffset)
	assert.Equal(t, int64(2), patcher.Missing[0].EndOffset)
	assert.Equal(t, int64(7), patcher.Missing[1].StartOffset)
	assert.Equal(t, int64(9), patcher.Missing[1].EndOffset)
	assert.Equal(t, int64(14), patcher.Missing[2].StartOffset)
	assert.Equal(t, int64(25), patcher.Missing[2].EndOffset)
}
