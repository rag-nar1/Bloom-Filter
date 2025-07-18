package cuckoo

import (
	"bytes"
	"math"
	"math/rand"

	"github.com/dgryski/go-metro"
	"github.com/rag-nar1/Filters/filter"
)

// fingerprint is considered as a single byte(8 bits)
// number of entries per bucket is 4
const (
	FpSize     = 8
	BucketSize = 4
	MaxKicks   = 500
	FPNULL     = 0
)

type CuckooFilter struct {
	M       uint32 // number of buckets
	Buckets [][BucketSize]byte
	Seed    uint64
	FpSeed  uint64
}

func nextPowerOfTwo(n uint32) uint32 {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

func NewCuckooFilter(n uint64, loadFactor float64) *CuckooFilter {
	m := nextPowerOfTwo(uint32(math.Ceil(float64(n) / float64(BucketSize) / loadFactor)))
	m = max(m, 1)
	return &CuckooFilter{
		M:       m,
		Buckets: make([][BucketSize]byte, m),
		Seed:    rand.Uint64(),
		FpSeed:  rand.Uint64(),
	}
}

func (cf *CuckooFilter) Insert(data []byte) bool {
	h1, fingerprint := cf.Hash(data)
	if cf.BucketInsert(fingerprint, h1) {
		return true
	}
	h2 := cf.AlternateIndex(h1, fingerprint)
	if cf.BucketInsert(fingerprint, h2) {
		return true
	}
	return cf.InsertFingerprint(fingerprint, RandomChoise(h1, h2), 1)
}

func (cf *CuckooFilter) InsertFingerprint(fingerprint byte, h uint32, kickingIdx uint32) bool {
	if kickingIdx > MaxKicks {
		return false
	}

	if cf.BucketInsert(fingerprint, h) {
		return true
	}

	// kick a random bucket to avoid going through the same graph cycle
	randomIndex := rand.Intn(BucketSize)
	kickedFingerprint := cf.Buckets[h][randomIndex]
	cf.Buckets[h][randomIndex] = fingerprint

	return cf.InsertFingerprint(kickedFingerprint, cf.AlternateIndex(h, kickedFingerprint), kickingIdx+1)
}

func (cf *CuckooFilter) Lookup(data []byte) bool {
	h1, fingerprint := cf.Hash(data)

	for _, val := range cf.Buckets[h1] {
		if val == fingerprint {
			return true
		}
	}

	h2 := cf.AlternateIndex(h1, fingerprint)
	for _, val := range cf.Buckets[h2] {
		if val == fingerprint {
			return true
		}
	}

	return false
}

func (cf *CuckooFilter) Delete(data []byte) bool {
	h1, fingerprint := cf.Hash(data)

	for i, val := range cf.Buckets[h1] {
		if val == fingerprint {
			cf.Buckets[h1][i] = FPNULL
			return true
		}
	}

	h2 := cf.AlternateIndex(h1, fingerprint)
	for i, val := range cf.Buckets[h2] {
		if val == fingerprint {
			cf.Buckets[h2][i] = FPNULL
			return true
		}
	}

	return false
}

// returns the fingerprint and the index of the first bucket
func (cf *CuckooFilter) Hash(data []byte) (uint32, byte) {
	hash := metro.Hash64(data, cf.Seed)

	h1 := uint32(hash>>32) & (cf.M - 1) // most significant 32 bits
	fingerprint := byte(hash)           // least significant 8 bits
	if fingerprint == FPNULL {
		fingerprint = 1
	}
	return h1, fingerprint
}

func (cf *CuckooFilter) AlternateIndex(h1 uint32, fingerprint byte) uint32 {
	fphash := uint32(metro.Hash64([]byte{fingerprint}, cf.FpSeed)>>32) & (cf.M - 1)

	return (h1 ^ fphash)
}

func (cf *CuckooFilter) BucketInsert(fingerprint byte, h uint32) bool {
	for i, _ := range cf.Buckets[h] {
		if cf.Buckets[h][i] == FPNULL {
			cf.Buckets[h][i] = fingerprint
			return true
		}
	}
	return false
}

func RandomChoise[T any](a T, b T) T {
	if rand.Intn(2) == 0 {
		return a
	}
	return b
}

// Serialize the filter to a byte slice in the following format:
// header|buckets
// header format: uint32(M)|uint64(FpSeed)|uint64(Seed) => 4 + 8 + 8 = 20 bytes
func (cf *CuckooFilter) Serialize() []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 12+cf.M*BucketSize))

	filter.SerializeUint(buf, uint64(cf.M), 4)
	filter.SerializeUint(buf, cf.FpSeed, 8)
	filter.SerializeUint(buf, cf.Seed, 8)

	for _, bucket := range cf.Buckets {
		buf.Write(bucket[:])
	}

	return buf.Bytes()
}

func Deserialize(data []byte) *CuckooFilter {
	buf := bytes.NewBuffer(data)

	m := filter.DeserializeUint[uint32](buf, 4)
	fpSeed := filter.DeserializeUint[uint64](buf, 8)
	seed := filter.DeserializeUint[uint64](buf, 8)

	cf := &CuckooFilter{
		M:       m,
		FpSeed:  fpSeed,
		Seed:    seed,
		Buckets: make([][BucketSize]byte, m),
	}

	for i := range cf.Buckets {
		buf.Read(cf.Buckets[i][:])
	}

	return cf
}