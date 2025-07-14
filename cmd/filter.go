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
	for i := range hashes {
		hashes[i] = &maphash.Hash{}
		hashes[i].SetSeed(maphash.MakeSeed())
	}

	return &BloomFilter{
		m:      m,
		k:      k,
		bits:   make([]bool, m),
		hashes: hashes,
	}
}

func (bf *BloomFilter) hash(data []byte) []int {
	hashedIdx := make([]int, bf.k)
	for i, fn := range bf.hashes {
		fn.Write(data)
		hashedIdx[i] = int(fn.Sum64() % uint64(bf.m))
		fn.Reset()
	}

	return hashedIdx
}

func (bf *BloomFilter) Insert(data []byte) {
	hashedIdx := bf.hash(data)
	for _, idx := range hashedIdx {
		bf.bits[idx] = true
	}
}

func (bf *BloomFilter) Exist(data []byte) bool {
	hashedIdx := bf.hash(data)
	for _, idx := range hashedIdx {
		if !bf.bits[idx] {
			return false
		}
	}
	return true
}
