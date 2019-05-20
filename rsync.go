package gosync

import (
	"hash"
	"hash/adler32"
	"io"

	"github.com/rkcloudchain/gosync/syncpb"
)

// consts...
const (
	ReadNextByte = iota
	ReadNextBlock
	ReadNone
)

type rsync struct {
	blockSize    int
	strongHasher hash.Hash
	fullChecksum hash.Hash
}

func (r *rsync) Sign(dest io.Reader) (*syncpb.ChunkChecksums, error) {
	defer r.fullChecksum.Reset()
	defer r.strongHasher.Reset()

	buffer := make([]byte, r.blockSize)
	checksums := make([]*syncpb.ChunkChecksum, 0)

	var index uint64
	for {
		n, err := io.ReadFull(dest, buffer)
		block := buffer[:n]

		if n == 0 {
			break
		}

		r.fullChecksum.Write(block)
		weak := ComputeWeakHash(block)
		strong := r.computeStrongHash(block)

		checksums = append(checksums, &syncpb.ChunkChecksum{BlockIndex: index, BlockSize: int64(n), WeakHash: weak, StrongHash: strong})
		index++

		if n != len(buffer) || err == io.EOF {
			break
		}
	}

	return &syncpb.ChunkChecksums{FileHash: r.fullChecksum.Sum(nil), Checksums: checksums}, nil
}

func (r *rsync) Delta(source io.ReadSeeker, checksum *syncpb.ChunkChecksums) {
	defer r.strongHasher.Reset()
	defer r.fullChecksum.Reset()

	index := makeChecksumIndex(checksum.Checksums)

	buffer := make([]byte, r.blockSize)
	n, _ := io.ReadFull(source, buffer)
	if n == 0 {
		return
	}

	block := buffer[:n]
	weak := adler32.Checksum(block)
	next := ReadNextByte

	for {
		if weakMatchList := index.FindWeakChecksum(weak); weakMatchList != nil {
			strong := r.computeStrongHash(block)
			strongList := index.FindStrongChecksum(weakMatchList, strong)

			if len(strongList) > 0 {
				if next == ReadNone {
					break
				}
				next = ReadNextBlock
			}
		}
	}
}

func (r *rsync) computeStrongHash(v []byte) []byte {
	r.strongHasher.Reset()
	r.strongHasher.Write(v)
	return r.strongHasher.Sum(nil)
}
