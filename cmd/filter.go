package BloomFilter

type BloomFilter struct {
	m int // size of bit-array
	k int // number of hash-functions

	bits []bool // the filter actual storage
}

func NewBloomFilter(m, k int) *BloomFilter {
	return &BloomFilter{
		m:    m,
		k:    k,
		bits: make([]bool, m),
	}
}


