package filter

import (
	"hash/maphash"
)

type BloomFilter struct {
	M int // size of bit-array
	K int // number of hash-functions

	Bits   []int64 // the filter actual storage
	Hashes []*maphash.Hash
}

func NewBloomFilter(m, k int) *BloomFilter {
	Hashes := make([]*maphash.Hash, k)
	for i := range Hashes {
		Hashes[i] = &maphash.Hash{}
		Hashes[i].SetSeed(maphash.MakeSeed())
	}

	return &BloomFilter{
		M:      m,
		K:      k,
		Bits:   make([]int64, m/64+1),
		Hashes: Hashes,
	}
}

func (bf *BloomFilter) Hash(data []byte) []int {
	hashedIdx := make([]int, bf.K)
	for i, fn := range bf.Hashes {
		fn.Write(data)
		hashedIdx[i] = int(fn.Sum64() % uint64(bf.M))
		fn.Reset()
	}

	return hashedIdx
}

func (bf *BloomFilter) Insert(data []byte) {
	hashedIdx := bf.Hash(data)
	for _, idx := range hashedIdx {
		pos := idx / 64
		bf.Bits[pos] |= int64(1) << (idx % 64)
	}
}

func (bf *BloomFilter) Exist(data []byte) bool {
	hashedIdx := bf.Hash(data)
	for _, idx := range hashedIdx {
		pos := idx / 64
		rem := idx % 64
		if (bf.Bits[pos] >> rem) & 1  == 0 {
			return false
		}
	}
	return true
}
