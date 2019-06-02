package gosync

import (
	"fmt"
	"hash"
	"hash/adler32"
	"io"

	"github.com/rkcloudchain/gosync/logging"
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

func newRSync(c *Config) *rsync {
	return &rsync{
		blockSize:        c.BlockSize,
		strongHasher:     c.StrongHasher,
		requestBlockSize: c.MaxRequestBlockSize,
		sizeFunc:         c.SizeFunc,
		reference:        c.Requester,
	}
}

type rsync struct {
	blockSize        int64
	strongHasher     hash.Hash
	requestBlockSize int64
	sizeFunc         func() (int64, error)
	reference        BlockRequester
}

func (r *rsync) Sign(dest io.Reader) *syncpb.ChunkChecksums {
	checksums := r.createSign(dest)
	logging.Debugf("Config block size: %d, generate %d checksums: %v", r.blockSize, len(checksums), checksums)
	return &syncpb.ChunkChecksums{ConfigBlockSize: r.blockSize, Checksums: checksums}
}

// Sign reads each block of the input file, and returns the checksums for each block.
func (r *rsync) createSign(dest io.Reader) []*syncpb.ChunkChecksum {
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

	return checksums
}

func (r *rsync) Patch(localFile io.ReadSeeker, patcher *syncpb.PatcherBlockSpan, output io.Writer) error {

	currentOffset := int64(0)
	localBlocks := patcher.Found[:]
	remoteBlocks := patcher.Missing[:]

	for len(localBlocks) > 0 || len(remoteBlocks) > 0 {
		if r.findInLocalBlocks(currentOffset, localBlocks) {
			logging.Debugf("Found local block %d", currentOffset)
			firstMatched := localBlocks[0]

			matchOffset := r.blockSize * int64(firstMatched.StartIndex)
			localFile.Seek(matchOffset, io.SeekStart)

			if _, err := io.Copy(output, io.LimitReader(localFile, firstMatched.BlockSize)); err != nil {
				return fmt.Errorf("Could not copy %d bytes to output: %v", firstMatched.BlockSize, err)
			}

			currentOffset += firstMatched.BlockSize
			localBlocks = localBlocks[1:]
		} else if r.findInRemoteBlocks(currentOffset, remoteBlocks) {
			logging.Debugf("Found remote block: %d", currentOffset)

			firstMissing := remoteBlocks[0]
			data, err := r.reference.DoRequest(firstMissing.StartOffset, firstMissing.EndOffset)
			if err != nil {
				return fmt.Errorf("Failed to read from reference file: %v", err)
			}

			if _, err := output.Write(data); err != nil {
				return fmt.Errorf("Could not write data to output: %v", err)
			}

			currentOffset += int64(len(data))
			remoteBlocks = remoteBlocks[1:]

		} else {
			logging.Errorf("Can't find any block with offset: %d", currentOffset)
			return fmt.Errorf("Could not find block offset in missing or matched list: %d", currentOffset)
		}
	}

	return nil
}

func (r *rsync) Delta(source io.ReaderAt, checksums *syncpb.ChunkChecksums) (*syncpb.PatcherBlockSpan, error) {
	matches, err := r.match(source, checksums.ConfigBlockSize, checksums.Checksums)
	if err != nil {
		return nil, err
	}
	logging.Debugf("Found %d match blocks: %v", len(matches), matches)

	merger := newMerger()
	merger.MergeResult(matches, checksums.ConfigBlockSize)

	mergedBlocks := merger.GetMergedBlocks()
	missing, err := r.fetchMissingBlocks(mergedBlocks, checksums.ConfigBlockSize)
	if err != nil {
		return nil, err
	}
	logging.Debugf("Found %d missing blocks: %v", len(missing), missing)

	patcher := &syncpb.PatcherBlockSpan{Found: r.patchFoundSpan(mergedBlocks), Missing: r.splitMissingBlocks(missing)}
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

func (r *rsync) splitMissingBlocks(blocks []*syncpb.MissingBlockSpan) []*syncpb.MissingBlockSpan {
	if r.requestBlockSize == 0 {
		return blocks
	}

	sorted := make([]*syncpb.MissingBlockSpan, 0)
	for _, block := range blocks {
		size := block.EndOffset - block.StartOffset + 1
		if size <= r.requestBlockSize {
			sorted = append(sorted, block)
		} else {
			start := block.StartOffset
			for size > r.requestBlockSize {
				b := &syncpb.MissingBlockSpan{StartOffset: start, EndOffset: start + r.requestBlockSize - 1}
				sorted = append(sorted, b)
				start = b.EndOffset + 1
				size = block.EndOffset - start + 1
			}

			sorted = append(sorted, &syncpb.MissingBlockSpan{StartOffset: start, EndOffset: block.EndOffset})
		}
	}

	return sorted
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
	return len(localBlocks) > 0 && localBlocks[0].ComparisonOffset == currentOffset
}

func (r *rsync) findInRemoteBlocks(currentOffset int64, remoteBlocks []*syncpb.MissingBlockSpan) bool {
	return len(remoteBlocks) > 0 && remoteBlocks[0].StartOffset == currentOffset
}
