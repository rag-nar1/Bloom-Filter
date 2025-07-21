package blockedbloom

import (
	"math"

	"github.com/rag-nar1/Filters/filter"
	"github.com/zeebo/xxh3"
)

const (
	BlockSize      = 256 // in bits
    WordSize       = 6 // in power of 2
	Uint64PerBlock = BlockSize >> WordSize
    WordMask       = 1 << WordSize - 1
)

type BlockedBloomFilter struct {
	BloomFilters []uint64 // 256 bits per block
	k            uint64
	BlockCount   uint64 // in blocks
	BlockMask    uint64
	BitMask      uint64
}

func NewBlockedBloomFilter(n uint64, fpRate float64) *BlockedBloomFilter {
	m := uint32(math.Ceil(-float64(n) * math.Log(fpRate) / (math.Log(2) * math.Log(2))))
	m = filter.NextPowerOfTwo(m)
	blockCount := uint64(m / BlockSize)
	bf := &BlockedBloomFilter{
		BloomFilters: make([]uint64, m>>WordSize),
		k:            Uint64PerBlock,
		BlockCount:   blockCount,
		BlockMask:    blockCount - 1,
		BitMask:      BlockSize - 1,
	}
	return bf
}

func (bf *BlockedBloomFilter) Insert(data []byte) {
	hash := xxh3.Hash128(data)
	blockIdx := hash.Lo & bf.BlockMask
	blockOffset := blockIdx * Uint64PerBlock
	h1 := uint64(hash.Hi)
	h2 := uint64(hash.Hi >> 32)

	for i := uint64(0); i < bf.k; i++ {
		bitIdx := (h1 + i*h2) & bf.BitMask
		bf.BloomFilters[blockOffset + bitIdx >> WordSize] |= 1 << (bitIdx & WordMask)
	}

}

func (bf *BlockedBloomFilter) Exist(data []byte) bool {
	hash := xxh3.Hash128(data)
	blockIdx := hash.Lo & bf.BlockMask
	blockOffset := blockIdx * Uint64PerBlock
	h1 := uint64(hash.Hi)
	h2 := uint64(hash.Hi >> 32)

	for i := uint64(0); i < bf.k; i++ {
		bitIdx := (h1 + i*h2) & bf.BitMask
		if bf.BloomFilters[blockOffset + bitIdx >> WordSize] & (1 << (bitIdx & WordMask)) == 0 {
			return false
		}
	}
	return true
}
