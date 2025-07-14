package BloomFilter

import (
	"hash/maphash"
)

type BloomFilter struct {
	m int // size of bit-array
	k int // number of hash-functions

	bits   []bool // the filter actual storage
	hashes []*maphash.Hash
}

func NewBloomFilter(m, k int) *BloomFilter {
	hashes := make([]*maphash.Hash, k)
	for _, fn := range hashes {
		fn = &maphash.Hash{}
		fn.SetSeed(maphash.MakeSeed())
	}

	return &BloomFilter{
		m:      m,
		k:      k,
		bits:   make([]bool, m),
		hashes: hashes,
	}
}

func (bf *BloomFilter) Hash(data []byte) []int {
	hashedIdx := make([]int, bf.k)
	for i, fn := range bf.hashes {
		fn.Write(data)
		hashedIdx[i] = int(fn.Sum64() % uint64(bf.m))
	}

	return hashedIdx
}

func (bf *BloomFilter) Insert(data []byte) {
	hashedIdx := bf.Hash(data)
	for _, idx := range hashedIdx {
		bf.bits[idx] = true
	}
}

func (bf *BloomFilter) Exist(data []byte) bool {
	hashedIdx := bf.Hash(data)
	for _, idx := range hashedIdx {
		if !bf.bits[idx] {
			return false
		}
	}
	return true
}
