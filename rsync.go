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

type blockMatchResult struct {
	Index            uint32
	Size             int64
	ComparisonOffset int64
}

func newRSync(blockSize int64, strongHasher hash.Hash, sizeFunc func() (int64, error)) *rsync {
	return &rsync{
		blockSize:    blockSize,
		strongHasher: strongHasher,
		sizeFunc:     sizeFunc,
	}
}

type rsync struct {
	blockSize    int64
	strongHasher hash.Hash
	sizeFunc     func() (int64, error)
}

// Sign reads each block of the input file, and returns the checksums for each block.
func (r *rsync) Sign(dest io.Reader) ([]*syncpb.ChunkChecksum, error) {
	defer r.strongHasher.Reset()

	buffer := make([]byte, r.blockSize)
	checksums := make([]*syncpb.ChunkChecksum, 0)

	var index uint32

	for {
		n, err := io.ReadFull(dest, buffer)
		block := buffer[:n]

		if n == 0 {
			break
		}

		weak := ComputeWeakHash(block)
		strong := r.computeStrongHash(block)

		checksums = append(checksums, &syncpb.ChunkChecksum{BlockIndex: index, WeakHash: weak, StrongHash: strong, BlockSize: int64(n)})

		if n != len(buffer) || err == io.EOF {
			break
		}

		index++
	}

	return checksums, nil
}

func (r *rsync) Patch(localFile io.ReadSeeker, localBlocks []*syncpb.FoundBlockSpan, remoteBlocks []*syncpb.MissingBlockSpan, output io.Writer) error {
	return nil
}

func (r *rsync) Delta(source io.ReaderAt, blockSize int64, checksums []*syncpb.ChunkChecksum) (*syncpb.PatcherBlockSpan, error) {
	matches, err := r.match(source, blockSize, checksums)
	if err != nil {
		return nil, err
	}

	merger := newMerger()
	merger.MergeResult(matches, blockSize)

	mergedBlocks := merger.GetMergedBlocks()
	missing, err := r.fetchMissingBlocks(mergedBlocks, blockSize)
	if err != nil {
		return nil, err
	}

	patcher := &syncpb.PatcherBlockSpan{Found: r.patchFoundSpan(mergedBlocks), Missing: missing}
	return patcher, nil
}

func (r *rsync) fetchMissingBlocks(sl blockSpanList, blockSize int64) ([]*syncpb.MissingBlockSpan, error) {
	sorted := make([]*syncpb.MissingBlockSpan, 0)
	size, err := r.sizeFunc()
	if err != nil {
		return nil, err
	}

	offset := int64(0)
	for _, blockSpan := range sl {
		if blockSpan.ComparisonOffset > offset {
			sorted = append(sorted, &syncpb.MissingBlockSpan{StartOffset: offset, EndOffset: blockSpan.ComparisonOffset - 1})
		}

		offset = blockSpan.ComparisonOffset + blockSpan.Size
	}

	if offset < size-1 {
		sorted = append(sorted, &syncpb.MissingBlockSpan{StartOffset: offset, EndOffset: size - 1})
	}

	return sorted, nil
}

func (r *rsync) match(source io.ReaderAt, blockSize int64, checksums []*syncpb.ChunkChecksum) ([]blockMatchResult, error) {
	defer r.strongHasher.Reset()

	index := makeChecksumIndex(checksums)
	matchResult := make([]blockMatchResult, 0)

	buffer := make([]byte, blockSize)
	next := ReadNextByte
	offset := int64(0)

	n, err := source.ReadAt(buffer, 0)
	if err != nil && n == 0 {
		return nil, err
	}

	block := buffer[:n]
	weak := adler32.Checksum(block)

	for {
		if weakMatchList := index.FindWeakChecksum(weak); weakMatchList != nil {
			strong := r.computeStrongHash(block)
			chunk := index.FindStrongChecksum(weakMatchList, strong)

			if chunk != nil {
				matchResult = append(matchResult, blockMatchResult{
					Index:            chunk.BlockIndex,
					Size:             chunk.BlockSize,
					ComparisonOffset: offset,
				})

				if next == ReadNone {
					break
				}
				next = ReadNextBlock
			}
		}

		switch next {
		case ReadNextBlock:
			offset += int64(n)
			n, err = source.ReadAt(buffer, offset)
			next = ReadNextByte

		case ReadNextByte:
			offset++
			n, err = source.ReadAt(buffer, offset)
		}

		if n > 0 {
			block = buffer[:n]
			weak = adler32.Checksum(block)
		}

		if next != ReadNone && err == io.EOF && n == 0 {
			next = ReadNone
		}

		if next == ReadNone {
			break
		}
	}

	return matchResult, nil
}

func (r *rsync) computeStrongHash(v []byte) []byte {
	r.strongHasher.Reset()
	r.strongHasher.Write(v)
	return r.strongHasher.Sum(nil)
}

func (r *rsync) patchFoundSpan(sl blockSpanList) []*syncpb.FoundBlockSpan {
	sorted := make([]*syncpb.FoundBlockSpan, len(sl))

	for i, v := range sl {
		s := &syncpb.FoundBlockSpan{ComparisonOffset: v.ComparisonOffset, BlockSize: v.Size, StartIndex: v.Start, EndIndex: v.End}
		sorted[i] = s
	}

	return sorted
}

func (r *rsync) findInLocalBlocks(currentOffset int64, localBlocks []*syncpb.FoundBlockSpan) bool {
	return len(localBlocks) > 0 && localBlocks[0].ComparisonOffset <= currentOffset
}

func (r *rsync) findInRemoteBlocks(currentOffset int64, remoteBlocks []*syncpb.MissingBlockSpan) bool {
	return len(remoteBlocks) > 0 && remoteBlocks[0].StartOffset <= currentOffset && remoteBlocks[0].EndOffset >= currentOffset
}

func newBlockSizeResolver(blockSize, fileSize int64) *blockSizeResolver {
	return &blockSizeResolver{BlockSize: blockSize, FileSize: fileSize}
}

type blockSizeResolver struct {
	BlockSize int64
	FileSize  int64
}

func (r *blockSizeResolver) GetBlockStartOffset(startIndex uint32) int64 {
	off := int64(startIndex) * r.BlockSize
	if r.FileSize != 0 && off > r.FileSize {
		return r.FileSize
	}

	return off
}

func (r *blockSizeResolver) GetBlockEndOffset(endIndex uint32) int64 {
	off := int64(endIndex) * r.BlockSize
	if r.FileSize != 0 && off > r.FileSize {
		return r.FileSize
	}

	return off
}
