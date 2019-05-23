package gosync

import (
	"fmt"
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
	Index     uint32
	Offset    int64
	BlockSize int64
}

func newRSync(c *Config) *rsync {
	return &rsync{
		blockSize:    c.BlockSize,
		strongHasher: c.StrongHasher,
	}
}

type rsync struct {
	blockSize    int64
	strongHasher hash.Hash
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

		checksums = append(checksums, &syncpb.ChunkChecksum{BlockIndex: index, BlockSize: int64(n), WeakHash: weak, StrongHash: strong})
		index++

		if n != len(buffer) || err == io.EOF {
			break
		}
	}

	return checksums, nil
}

func (r *rsync) Patch(localFile io.ReadSeeker, localBlocks []*syncpb.FoundBlockSpan, remoteBlocks []*syncpb.MissingBlockSpan, output io.Writer) error {
	currentOffset := int64(0)

	for len(localBlocks) > 0 && len(remoteBlocks) > 0 {
		if r.findInLocalBlocks(currentOffset, localBlocks) {
			firstMatched := localBlocks[0]

			localFile.Seek(currentOffset, io.SeekStart)
			if _, err := io.Copy(output, io.LimitReader(localFile, firstMatched.BlockSize)); err != nil {
				return fmt.Errorf("Could not copy %d bytes to output: %v", firstMatched.BlockSize, err)
			}

			currentOffset += firstMatched.BlockSize
			localBlocks = localBlocks[1:]

		} else if r.findInRemoteBlocks(currentOffset, remoteBlocks) {
			firstMissing := remoteBlocks[0]
			if _, err := output.Write(firstMissing.Data); err != nil {
				return fmt.Errorf("Could not write data to output: %v", err)
			}

			currentOffset += int64(len(firstMissing.Data))
			remoteBlocks = remoteBlocks[1:]

		} else {
			return fmt.Errorf("Could not found block in missing or matched list: %d", currentOffset)
		}
	}

	return nil
}

func (r *rsync) Delta(source io.ReaderAt, fileSize int64, checksums []*syncpb.ChunkChecksum) (*syncpb.PatcherBlockSpan, error) {
	matches, err := r.Match(source, checksums)
	if err != nil {
		return nil, err
	}

	merger := newMerger()
	merger.MergeResult(matches)

	mergedBlocks := merger.GetMergedBlocks()
	patcher := &syncpb.PatcherBlockSpan{Found: r.patchFoundSpan(mergedBlocks)}

	return patcher, patcher.GetMissingBlocks(source, fileSize)
}

func (r *rsync) Match(source io.ReaderAt, checksums []*syncpb.ChunkChecksum) ([]blockMatchResult, error) {
	defer r.strongHasher.Reset()

	index := makeChecksumIndex(checksums)
	matchResult := make([]blockMatchResult, 0)

	buffer := make([]byte, r.blockSize)
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
					Offset:    offset,
					Index:     chunk.BlockIndex,
					BlockSize: chunk.BlockSize,
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

		if next != ReadNone && (err == io.EOF || err == io.ErrUnexpectedEOF) {
			next = ReadNone
		}

		if next == ReadNone && n == 0 {
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
	result := make([]*syncpb.FoundBlockSpan, len(sl))

	for i, v := range sl {
		result[i].MatchOffset = v.StartOffset
		result[i].BlockSize = v.Size
	}

	return result
}

func (r *rsync) findInLocalBlocks(currentOffset int64, localBlocks []*syncpb.FoundBlockSpan) bool {
	return len(localBlocks) > 0 && localBlocks[0].MatchOffset == currentOffset
}

func (r *rsync) findInRemoteBlocks(currentOffset int64, remoteBlocks []*syncpb.MissingBlockSpan) bool {
	return len(remoteBlocks) > 0 && remoteBlocks[0].StartOffset == currentOffset
}
