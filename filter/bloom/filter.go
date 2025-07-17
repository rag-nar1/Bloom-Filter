package bloom

import (
	"math"

	"github.com/rag-nar1/Filters/filter"
)

type BloomFilter struct {
	M uint64 // size of bit-array
	K int // number of hash-functions

	Bits   []uint64 // the filter actual storage
	Hasher filter.Hash
}

func NewBloomFilter(n uint64, fpRate float64, hasher filter.Hash) *BloomFilter {
	// m = ceil((n * log(p)) / log(1 / pow(2, log(2))));
	// k = round((m / n) * log(2));
	m := uint64(math.Ceil(float64(n) * math.Log(fpRate) / math.Log(1/math.Pow(2, math.Log(2)))))
	k := int(math.Round(float64(m) / float64(n) * math.Log(2)))
	return &BloomFilter{
		M:      m,
		K:      k,
		Bits:   make([]uint64, m/64+1),
		Hasher: hasher,
	}
}

func (bf *BloomFilter) Hash(data []byte) []int {
	return bf.Hasher(data, bf.M, bf.K)
}

func (bf *BloomFilter) Insert(data []byte) {
	hashedIdx := bf.Hash(data)
	for _, idx := range hashedIdx {
		pos := idx / 64
		bf.Bits[pos] |= uint64(1) << (idx % 64)
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
