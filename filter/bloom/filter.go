package bloom

import (
	"bytes"
	"math"
	"math/rand"

	"github.com/dgryski/go-metro"
	"github.com/rag-nar1/Filters/filter"
)

type BloomFilter struct {
	M    uint32 // size of bit-array
	K    uint32 // number of hash-functions
	Seed uint64

	Bits []uint64 // the filter actual storage
}

func NewBloomFilter(n uint64, fpRate float64) *BloomFilter {
	// m = ceil((n * log(p)) / log(1 / pow(2, log(2))));
	// k = round((m / n) * log(2));
	m := uint32(math.Ceil(float64(n) * math.Log(fpRate) / math.Log(1/math.Pow(2, math.Log(2)))))
	k := uint32(math.Round(float64(m) / float64(n) * math.Log(2)))
	m = filter.NextPowerOfTwo(m)
	return &BloomFilter{
		M:    m,
		K:    k,
		Bits: make([]uint64, m/64+1),
		Seed: rand.Uint64(),
	}
}

func (bf *BloomFilter) Hash(data []byte) []int {
	return filter.DoubleHash(metro.Hash64(data, bf.Seed), bf.M, bf.K)
}

func (bf *BloomFilter) Insert(data []byte) {
	hash := metro.Hash64(data, bf.Seed)
	h1 := uint32(hash)
	h2 := uint32(hash >> 32)
	for i := uint32(0); i < bf.K; i++ {
		idx := (h1 + i*h2) & (bf.M - 1)
		pos := idx / 64
		bf.Bits[pos] |= uint64(1) << (idx & 63)
	}
}

func (bf *BloomFilter) Exist(data []byte) bool {
	hash := metro.Hash64(data, bf.Seed)
	h1 := uint32(hash)
	h2 := uint32(hash >> 32)
	for i := uint32(0); i < bf.K; i++ {
		idx := (h1 + i*h2) & (bf.M - 1)
		pos := idx / 64
		if (bf.Bits[pos]>>(idx&63))&1 == 0 {
			return false
		}
	}
	return true
}

// Serialize the filter to a byte slice in the following format:
// header|bits
// header format: uint32(M)|uint32(K)|uint64(seed) => 4 + 4 + 8 = 16 bytes
func (bf *BloomFilter) Serialize() []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 16+len(bf.Bits)*8))
	filter.SerializeUint(buf, uint64(bf.M), 4)
	filter.SerializeUint(buf, uint64(bf.K), 4)
	filter.SerializeUint(buf, bf.Seed, 8)
	for _, bit := range bf.Bits {
		filter.SerializeUint(buf, bit, 8)
	}
	return buf.Bytes()
}

func Deserialize(data []byte) *BloomFilter {
	buf := bytes.NewBuffer(data)
	m := filter.DeserializeUint[uint32](buf, 4)
	k := filter.DeserializeUint[uint32](buf, 4)
	seed := filter.DeserializeUint[uint64](buf, 8)
	bits := make([]uint64, m/64+1)
	for i := range bits {
		bits[i] = filter.DeserializeUint[uint64](buf, 8)
	}
	return &BloomFilter{
		M:    m,
		K:    k,
		Seed: seed,
		Bits: bits,
	}
}
