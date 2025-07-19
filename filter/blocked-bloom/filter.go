package blockedbloom

import (
	"math"
	"math/rand"

	"github.com/rag-nar1/Filters/filter"
	"github.com/zeebo/xxh3"
)

const (
	BlockSize      = 256 // in bits
	Uint64PerBlock = BlockSize >> 6
)

type BlockedBloomFilter struct {
	BloomFilters []uint64 // 256 bits per block
	k            uint32
	BlockCount   uint64 // in blocks
	Seed         uint64
	BlockSeed    uint64
}

func NewBlockedBloomFilter(n uint64, fpRate float64) *BlockedBloomFilter {
	m := uint32(math.Ceil(-float64(n) * math.Log(fpRate) / (math.Log(2) * math.Log(2))))
	m = filter.NextPowerOfTwo(m)
	blockCount := uint64(m / BlockSize)
	bf := &BlockedBloomFilter{
		BloomFilters: make([]uint64, m>>6),
		k:            4,
		BlockCount:   blockCount,
		Seed:         rand.Uint64(),
		BlockSeed:    rand.Uint64(),
	}
	return bf
}

func (bf *BlockedBloomFilter) Insert(data []byte) {
	hash := xxh3.Hash128(data)
	blockIdx := hash.Lo & (bf.BlockCount - 1)
	blockOffset := blockIdx * Uint64PerBlock
	h1 := uint32(hash.Hi)
	h2 := uint32(hash.Hi >> 32)

	var masks [Uint64PerBlock]uint64
	pos := h1 & (BlockSize - 1)
	masks[pos>>6] |= uint64(1) << (pos & 63)
	pos = (h1 + h2) & (BlockSize - 1)
	masks[pos>>6] |= uint64(1) << (pos & 63)
	pos = (h1 + 2*h2) & (BlockSize - 1)
	masks[pos>>6] |= uint64(1) << (pos & 63)
	pos = (h1 + 3*h2) & (BlockSize - 1)
	masks[pos>>6] |= uint64(1) << (pos & 63)

	bf.BloomFilters[blockOffset+0] |= masks[0]
	bf.BloomFilters[blockOffset+1] |= masks[1]
	bf.BloomFilters[blockOffset+2] |= masks[2]
	bf.BloomFilters[blockOffset+3] |= masks[3]
}

func (bf *BlockedBloomFilter) Exist(data []byte) bool {
	hash := xxh3.Hash128(data)
	blockIdx := hash.Lo & (bf.BlockCount - 1)
	blockOffset := blockIdx * Uint64PerBlock
	h1 := uint32(hash.Hi)
	h2 := uint32(hash.Hi >> 32)
	for i := uint32(0); i < bf.k; i++ {
		pos := (h1 + i*h2) & (BlockSize - 1)
		uintIdx := pos >> 6
		bitIdx := pos & 63
		if (bf.BloomFilters[blockOffset+uint64(uintIdx)]>>bitIdx)&1 == 0 {
			return false
		}
	}
	return true
}
