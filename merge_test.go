package gosync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeBlocksAfter(t *testing.T) {
	result := []blockMatchResult{
		{Index: 0, ComparisonOffset: 0},
		{Index: 1, ComparisonOffset: 4},
	}

	merger := newMerger()
	merger.MergeResult(result, 4)

	merged := merger.GetMergedBlocks()
	assert.Len(t, merged, 1)
	assert.Equal(t, uint32(1), merged[0].End)
}

func TestMergeBlocksBefore(t *testing.T) {
	result := []blockMatchResult{
		{Index: 1, ComparisonOffset: 4},
		{Index: 0, ComparisonOffset: 0},
	}

	merger := newMerger()
	merger.MergeResult(result, 4)

	merged := merger.GetMergedBlocks()
	assert.Len(t, merged, 1)
	assert.Equal(t, uint32(1), merged[0].End)
}

func TestMergeBlocksBetween(t *testing.T) {
	result := []blockMatchResult{
		{Index: 2, ComparisonOffset: 8},
		{Index: 0, ComparisonOffset: 0},
		{Index: 1, ComparisonOffset: 4},
	}

	merger := newMerger()
	merger.MergeResult(result, 4)

	merged := merger.GetMergedBlocks()
	assert.Len(t, merged, 1)
	assert.Equal(t, uint32(2), merged[0].End)
}
