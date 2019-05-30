package gosync

import (
	"testing"

	"github.com/rkcloudchain/gosync/syncpb"
	"github.com/stretchr/testify/assert"
)

var weakA = uint32(1)
var weakB = uint32(2)

func TestMakeChecksumIndex(t *testing.T) {
	i := makeChecksumIndex([]*syncpb.ChunkChecksum{})
	assert.Equal(t, 0, i.blockCount)

	i = makeChecksumIndex([]*syncpb.ChunkChecksum{
		{BlockIndex: 0, WeakHash: weakA, StrongHash: []byte("b")},
		{BlockIndex: 1, WeakHash: weakB, StrongHash: []byte("c")},
	})

	assert.Equal(t, 2, i.blockCount)
}

func TestFindWeakChecksum(t *testing.T) {
	i := makeChecksumIndex([]*syncpb.ChunkChecksum{
		{BlockIndex: 0, WeakHash: weakA, StrongHash: []byte("b")},
		{BlockIndex: 1, WeakHash: weakB, StrongHash: []byte("c")},
		{BlockIndex: 2, WeakHash: weakB, StrongHash: []byte("d")},
	})

	result := i.FindWeakChecksum(weakA)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, uint32(0), result[0].BlockIndex)

	result = i.FindWeakChecksum(weakB)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, uint32(1), result[0].BlockIndex)

	result = i.FindWeakChecksum(uint32(3))
	assert.Nil(t, result)
}

func TestFindStrongChecksum(t *testing.T) {
	i := makeChecksumIndex([]*syncpb.ChunkChecksum{
		{BlockIndex: 0, WeakHash: weakA, StrongHash: []byte("b")},
		{BlockIndex: 1, WeakHash: weakB, StrongHash: []byte("c")},
		{BlockIndex: 2, WeakHash: weakB, StrongHash: []byte("d")},
	})

	result := i.FindWeakChecksum(weakB)
	strong := result.FindStrongChecksum([]byte("s"))
	assert.Nil(t, strong)

	strong = result.FindStrongChecksum([]byte("d"))
	assert.NotNil(t, strong)
	assert.Equal(t, uint32(2), strong.BlockIndex)
}

func TestFindStrongChecksum2(t *testing.T) {
	i := makeChecksumIndex([]*syncpb.ChunkChecksum{
		{BlockIndex: 0, WeakHash: weakA, StrongHash: []byte("b")},
		{BlockIndex: 1, WeakHash: weakB, StrongHash: []byte("c")},
		{BlockIndex: 2, WeakHash: weakB, StrongHash: []byte("d")},
	})

	result := i.FindWeakChecksum(weakA)
	strong := i.FindStrongChecksum(result, []byte("b"))
	assert.NotNil(t, strong)
	assert.Equal(t, uint32(0), strong.BlockIndex)
}
