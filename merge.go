package gosync

import (
	"sort"

	"github.com/petar/GoLLRB/llrb"
)

func newMerger() *matchMerger {
	return &matchMerger{blockMap: llrb.New()}
}

type matchMerger struct {
	blockMap *llrb.LLRB
}

func (merger *matchMerger) MergeResult(results []blockMatchResult) {
	for _, result := range results {
		blockID := result.Index
		preceeding := merger.blockMap.Get(blockSpanKey(blockID - 1))
		following := merger.blockMap.Get(blockSpanKey(blockID + 1))

		span := merger.toBlockSpan(result)

		var found bool
		merger.blockMap.AscendGreaterOrEqual(blockSpanKey(blockID), func(i llrb.Item) bool {
			j, ok := i.(blockSpanIndex)
			if !ok {
				found = true
				return false
			}

			switch k := j.(type) {
			case blockSpanStart:
				found = k.Start == blockID
				return false
			case blockSpanEnd:
				found = true
				return false
			default:
				found = true
				return false
			}
		})

		if found {
			continue
		}

		merger.blockMap.ReplaceOrInsert(blockSpanStart(*merger.toBlockSpan(result)))

		if preceeding != nil && following != nil {
			a := merger.itemToBlockSpan(preceeding)
			merger.merge(span, &a)

			b := merger.itemToBlockSpan(following)
			merger.merge(&a, &b)
		} else if preceeding != nil {
			a := merger.itemToBlockSpan(preceeding)
			merger.merge(span, &a)
		} else if following != nil {
			b := merger.itemToBlockSpan(following)
			merger.merge(span, &b)
		}
	}
}

func (merger *matchMerger) GetMergedBlocks() blockSpanList {
	sorted := make(blockSpanList, 0)

	merger.blockMap.AscendGreaterOrEqual(merger.blockMap.Min(), func(item llrb.Item) bool {
		switch block := item.(type) {
		case blockSpanStart:
			sorted = append(sorted, blockSpan(block))
		}
		return true
	})

	sort.Sort(sorted)
	return sorted
}

func (merger *matchMerger) merge(block1, block2 *blockSpan) {
	var a, b *blockSpan = block1, block2

	if block1.Start > block2.Start {
		a, b = block2, block1
	}

	if a.End == b.Start-1 && a.EndOffset(a.Size) == b.StartOffset {
		merger.blockMap.Delete(blockSpanKey(a.End))
		merger.blockMap.Delete(blockSpanKey(b.Start))
		a.End = b.End
		a.Size += b.Size

		merger.blockMap.ReplaceOrInsert(blockSpanStart(*a))
		merger.blockMap.ReplaceOrInsert(blockSpanEnd(*a))
	}
}

func (merger *matchMerger) itemToBlockSpan(in llrb.Item) blockSpan {
	switch i := in.(type) {
	case blockSpanStart:
		return blockSpan(i)
	case blockSpanEnd:
		return blockSpan(i)
	}
	return blockSpan{}
}

func (merger *matchMerger) toBlockSpan(b blockMatchResult) *blockSpan {
	return &blockSpan{
		Start:       b.Index,
		End:         b.Index,
		StartOffset: b.Offset,
		Size:        b.BlockSize,
	}
}

type blockSpan struct {
	Start       uint32
	End         uint32
	StartOffset int64
	Size        int64
}

func (b blockSpan) EndOffset(blockSize int64) int64 {
	return b.StartOffset + blockSize*int64(b.End-b.Start+1)
}

type blockSpanIndex interface {
	Position() uint32
}

type blockSpanStart blockSpan

func (s blockSpanStart) Position() uint32 {
	return s.Start
}

func (s blockSpanStart) Less(than llrb.Item) bool {
	return s.Start < than.(blockSpanIndex).Position()
}

type blockSpanEnd blockSpan

func (s blockSpanEnd) Position() uint32 {
	return s.End
}

func (s blockSpanEnd) Less(than llrb.Item) bool {
	return s.End < than.(blockSpanIndex).Position()
}

type blockSpanKey uint32

func (s blockSpanKey) Position() uint32 {
	return uint32(s)
}

func (s blockSpanKey) Less(than llrb.Item) bool {
	return uint32(s) < than.(blockSpanIndex).Position()
}

type blockSpanList []blockSpan

func (l blockSpanList) Len() int {
	return len(l)
}

func (l blockSpanList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l blockSpanList) Less(i, j int) bool {
	return l[i].Start < l[j].Start
}
