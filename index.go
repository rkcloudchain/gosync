package gosync

import (
	"bytes"
	"sort"

	"github.com/rkcloudchain/gosync/syncpb"
)

type strongChecksumList []*syncpb.ChunkChecksum

func (s strongChecksumList) Len() int {
	return len(s)
}

func (s strongChecksumList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s strongChecksumList) Less(i, j int) bool {
	return bytes.Compare(s[i].StrongHash, s[j].StrongHash) == -1
}

type checksumIndex struct {
	blockCount         int
	weakChecksumLookup []map[uint32]strongChecksumList
}

func makeChecksumIndex(checksums []*syncpb.ChunkChecksum) *checksumIndex {
	n := &checksumIndex{
		blockCount:         len(checksums),
		weakChecksumLookup: make([]map[uint32]strongChecksumList, 256),
	}

	for _, chunk := range checksums {
		offset := chunk.WeakHash & 255

		if n.weakChecksumLookup[offset] == nil {
			n.weakChecksumLookup[offset] = make(map[uint32]strongChecksumList)
		}

		n.weakChecksumLookup[offset][chunk.WeakHash] = append(n.weakChecksumLookup[offset][chunk.WeakHash], chunk)
	}

	for _, a := range n.weakChecksumLookup {
		for _, c := range a {
			sort.Sort(c)
		}
	}

	return n
}

func (index *checksumIndex) FindWeakChecksum(weak uint32) strongChecksumList {
	offset := weak & 255
	if index.weakChecksumLookup[offset] != nil {
		if v, ok := index.weakChecksumLookup[offset][weak]; ok {
			return v
		}
	}

	return nil
}

func (index *checksumIndex) FindStrongChecksum(weakMatchList strongChecksumList, strong []byte) []*syncpb.ChunkChecksum {
	return weakMatchList.FindStrongChecksum(strong)
}

func (s strongChecksumList) FindStrongChecksum(strong []byte) []*syncpb.ChunkChecksum {
	n := len(s)

	if n == 1 {
		if bytes.Compare(s[0].StrongHash, strong) == 0 {
			return s
		}

		return nil
	}

	firstChecksum := sort.Search(n, func(i int) bool {
		return bytes.Compare(s[i].StrongHash, strong) >= 0
	})

	if firstChecksum == -1 || firstChecksum == n {
		return nil
	}

	if bytes.Compare(s[firstChecksum].StrongHash, strong) != 0 {
		return nil
	}

	end := firstChecksum + 1
	for end < n {
		if bytes.Compare(s[end].StrongHash, strong) == 0 {
			end++
		} else {
			break
		}
	}

	return s[firstChecksum:end]
}
