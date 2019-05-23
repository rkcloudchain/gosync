package syncpb

import (
	"bytes"
	io "io"
	"sort"
)

// Match compares a chunksum to another based on the chunksums
func (chunk *ChunkChecksum) Match(other *ChunkChecksum) bool {
	weakEqual := chunk.WeakHash == other.WeakHash
	strongEqual := false
	if weakEqual {
		strongEqual = bytes.Compare(chunk.StrongHash, other.StrongHash) == 0
	}

	return weakEqual && strongEqual
}

type missingBlockList []*MissingBlockSpan

func (l missingBlockList) Len() int {
	return len(l)
}

func (l missingBlockList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l missingBlockList) Less(i, j int) bool {
	return l[i].StartOffset < l[j].StartOffset
}

// GetMissingBlocks creates a list of spans that are missing.
func (b *PatcherBlockSpan) GetMissingBlocks(reader io.ReaderAt, fileSize int64) error {
	sorted := make(missingBlockList, 0)

	offset := int64(0)
	length := int64(0)

	for _, blockSpan := range b.Found {
		if blockSpan.MatchOffset > offset {
			length++
			continue
		}

		if length > 0 {
			data := make([]byte, length)
			_, err := reader.ReadAt(data, offset)
			if err != nil {
				return err
			}

			sorted = append(sorted, &MissingBlockSpan{StartOffset: offset, Data: data})
		}

		offset += blockSpan.BlockSize + length
		length = 0
	}

	if offset < fileSize-1 {
		data := make([]byte, fileSize-offset)
		_, err := reader.ReadAt(data, offset)
		if err != nil {
			return err
		}

		sorted = append(sorted, &MissingBlockSpan{StartOffset: offset, Data: data})
	}

	sort.Sort(sorted)
	b.Missing = sorted
	return nil
}
