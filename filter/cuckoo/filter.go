package cuckoo

import (
	"math"
	"math/rand"
	"slices"

	"github.com/dgryski/go-metro"
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
	Buckets []byte
	Seed    uint64
	FpSeed  uint64
	Stash   []byte
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
		Buckets: make([]byte, m*BucketSize),
		Seed:    rand.Uint64(),
		FpSeed:  rand.Uint64(),
		Stash:   make([]byte, 0),
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
		cf.Stash = append(cf.Stash, fingerprint)
		return false
	}

	if cf.BucketInsert(fingerprint, h) {
		return true
	}

	// kick a random bucket to avoid going through the same graph cycle
	randomIndex := rand.Intn(BucketSize)
	kickedFingerprint := cf.Buckets[h*BucketSize+uint32(randomIndex)]
	cf.Buckets[h*BucketSize+uint32(randomIndex)] = fingerprint

	return cf.InsertFingerprint(kickedFingerprint, cf.AlternateIndex(h, kickedFingerprint), kickingIdx+1)
}

func (cf *CuckooFilter) Lookup(data []byte) bool {
	h1, fingerprint := cf.Hash(data)
	if slices.Contains(cf.Stash, fingerprint) {
		return true
	}

	for i := 0; i < BucketSize; i++ {
		val := cf.Buckets[h1*BucketSize+uint32(i)]
		if val == fingerprint {
			return true
		}
	}

	h2 := cf.AlternateIndex(h1, fingerprint)
	for i := 0; i < BucketSize; i++ {
		val := cf.Buckets[h2*BucketSize+uint32(i)]
		if val == fingerprint {
			return true
		}
	}

	return false
}

func (cf *CuckooFilter) Delete(data []byte) bool {
	h1, fingerprint := cf.Hash(data)
	if slices.Contains(cf.Stash, fingerprint) {
		idx := slices.Index(cf.Stash, fingerprint)
		cf.Stash = slices.Delete(cf.Stash, idx, idx+1)
		return true
	}

	for i := 0; i < BucketSize; i++ {
		val := cf.Buckets[h1*BucketSize+uint32(i)]
		if val == fingerprint {
			cf.Buckets[h1*BucketSize+uint32(i)] = FPNULL
			return true
		}
	}

	h2 := cf.AlternateIndex(h1, fingerprint)
	for i := 0; i < BucketSize; i++ {
		if val := cf.Buckets[h2*BucketSize+uint32(i)]; val == fingerprint {
			cf.Buckets[h2*BucketSize+uint32(i)] = FPNULL
			return true
		}
	}

	return false
}

// returns the fingerprint and the index of the first bucket
func (cf *CuckooFilter) Hash(data []byte) (uint32, byte) {
	hash := metro.Hash64(data, cf.Seed)

	h1 := uint32(hash>>32) & (cf.M - 1) // most significant 32 bits
	fingerprint := byte(hash)     // least significant 8 bits
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
	for i := 0; i < BucketSize; i++ {
		if cf.Buckets[h*BucketSize+uint32(i)] == FPNULL {
			cf.Buckets[h*BucketSize+uint32(i)] = fingerprint
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
