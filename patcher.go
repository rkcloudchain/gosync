package gosync

import "hash"

type foundBlockSpan struct {
	Start       uint32
	End         uint32
	BlockSize   int64
	MatchOffset int64
}

type missingBlockSpan struct {
	Start        uint32
	End          uint32
	BlockSize    int64
	Hasher       hash.Hash
	ExpectedSums [][]byte
}

func toPatcherFoundSpan(sl blockSpanList, blockSize int64) []foundBlockSpan {
	result := make([]foundBlockSpan, len(sl))

	for i, v := range sl {
		result[i].Start = v.Start
		result[i].End = v.End
		result[i].MatchOffset = v.StartOffset
		result[i].BlockSize = blockSize
	}

	return result
}

func toPatcherMissingSpan(sl blockSpanList, blockSize int64) []missingBlockSpan {
	result := make([]missingBlockSpan, len(sl))

	for i, v := range sl {
		result[i].Start = v.Start
		result[i].End = v.End
		result[i].BlockSize = blockSize
	}

	return result
}
