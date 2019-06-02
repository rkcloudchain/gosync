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
	checksums := r.Sign(bytes.NewReader(nil))
	assert.Len(t, checksums.Checksums, 0)
}

func TestGenerateChecksums(t *testing.T) {
	data := make([]byte, 4*1024*1024)
	n, err := rand.Read(data)
	require.NoError(t, err)
	require.Equal(t, 4*1024*1024, n)

	r := &rsync{blockSize: 512 * 1024, strongHasher: md5.New()}
	checksums := r.Sign(bytes.NewReader(data))
	assert.Len(t, checksums.Checksums, 8)
}

func TestGenerateChecksums2(t *testing.T) {
	data := make([]byte, 8520959)
	n, err := rand.Read(data)
	require.NoError(t, err)
	require.Equal(t, 8520959, n)

	r := &rsync{blockSize: 512 * 1024, strongHasher: sha256.New()}
	checksums := r.Sign(bytes.NewReader(data))
	checksum := checksums.Checksums[len(checksums.Checksums)-1]
	assert.NotEqual(t, 512*1024, checksum.Size())
}

func TestGenerateChecksums3(t *testing.T) {
	r := &rsync{blockSize: 2, strongHasher: md5.New()}
	checksums := r.Sign(bytes.NewReader([]byte("hello world")))
	assert.Len(t, checksums.Checksums, 6)
	assert.Equal(t, int64(1), checksums.Checksums[5].BlockSize)
}

func TestMatch(t *testing.T) {
	reader := bytes.NewReader([]byte("123abcdefg"))
	r := &rsync{blockSize: 3, strongHasher: md5.New()}

	checksums := r.Sign(reader)
	assert.Len(t, checksums.Checksums, 4)

	source := bytes.NewReader([]byte("123xxabc def"))
	results, err := r.match(source, r.blockSize, checksums.Checksums)
	assert.NoError(t, err)
	assert.Len(t, results, 3)

	assert.Equal(t, int64(0), results[0].ComparisonOffset)
	assert.Equal(t, int64(5), results[1].ComparisonOffset)
	assert.Equal(t, int64(9), results[2].ComparisonOffset)
}

func TestMatch2(t *testing.T) {
	reader := bytes.NewReader([]byte("hello"))
	r := &rsync{blockSize: 2, strongHasher: sha256.New(), sizeFunc: func() (int64, error) { return int64(2), nil }}

	checksums := r.Sign(reader)
	assert.Len(t, checksums.Checksums, 3)

	source := bytes.NewReader([]byte("helllo"))
	results, err := r.match(source, 2, checksums.Checksums)
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

	r := &rsync{blockSize: 4, strongHasher: md5.New(), sizeFunc: func() (int64, error) { return int64(len(bs)), nil }}
	checksums := r.Sign(dst)
	assert.Len(t, checksums.Checksums, 4)

	patcher, err := r.Delta(src, checksums)
	assert.NoError(t, err)
	assert.Len(t, patcher.Found, 3)
	assert.Len(t, patcher.Missing, 3)

	assert.Equal(t, uint32(0), patcher.Found[0].StartIndex)
	assert.Equal(t, uint32(0), patcher.Found[0].EndIndex)
	assert.Equal(t, uint32(1), patcher.Found[1].StartIndex)
	assert.Equal(t, uint32(1), patcher.Found[1].EndIndex)
	assert.Equal(t, uint32(3), patcher.Found[2].StartIndex)
	assert.Equal(t, uint32(3), patcher.Found[2].StartIndex)

	assert.Equal(t, int64(0), patcher.Missing[0].StartOffset)
	assert.Equal(t, int64(2), patcher.Missing[0].EndOffset)
	assert.Equal(t, int64(7), patcher.Missing[1].StartOffset)
	assert.Equal(t, int64(9), patcher.Missing[1].EndOffset)
	assert.Equal(t, int64(14), patcher.Missing[2].StartOffset)
	assert.Equal(t, int64(25), patcher.Missing[2].EndOffset)
}

func TestEmptyData(t *testing.T) {
	dst := bytes.NewReader(nil)
	src := []byte("abcdefghijklmn")
	reader := bytes.NewReader(src)

	r := &rsync{blockSize: 4, strongHasher: md5.New(), sizeFunc: func() (int64, error) { return int64(len(src)), nil }}
	checksums := r.Sign(dst)
	assert.Len(t, checksums.Checksums, 0)

	patcher, err := r.Delta(reader, checksums)
	assert.NoError(t, err)
	assert.Len(t, patcher.Found, 0)
	assert.Len(t, patcher.Missing, 1)
	assert.Equal(t, int64(0), patcher.Missing[0].StartOffset)
	assert.Equal(t, int64(len(src)-1), patcher.Missing[0].EndOffset)
}

func TestSplitMissingBlocks(t *testing.T) {
	dst := bytes.NewReader([]byte("hello"))
	src := []byte("he1234567890llo")
	reader := bytes.NewReader(src)

	r := &rsync{blockSize: 2, strongHasher: md5.New(), sizeFunc: func() (int64, error) { return int64(len(src)), nil }, requestBlockSize: 2}
	checksums := r.Sign(dst)
	assert.Len(t, checksums.Checksums, 3)

	patcher, err := r.Delta(reader, checksums)
	assert.NoError(t, err)
	assert.Len(t, patcher.Found, 2)
	assert.Len(t, patcher.Missing, 5)
}
