package cuckoo

import (
	"hash"
	"hash/fnv"
	"math"
	"math/rand"
	"slices"
)

// fingerprint is considered as a single byte(8 bits)
// number of entries per bucket is 4
const (
	FpSize     = 8
	BucketSize = 4
	MaxKicks   = 500
)

type CuckooFilter struct {
	M       uint32 // number of buckets
	Buckets [][]byte
	Hasher  hash.Hash64
	FPHasher hash.Hash64
	Stash    []byte
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

	return &CuckooFilter{
		M:       m,
		Buckets: make([][]byte, m),
		Hasher:  fnv.New64a(),
		FPHasher: fnv.New64a(),
		Stash: make([]byte, 0),
	}
}

// returns the fingerprint and the index of the first bucket
func (cf *CuckooFilter) Hash(data []byte) (uint32, byte) {
	cf.Hasher.Reset()
	cf.Hasher.Write(data)
	hash := cf.Hasher.Sum64()

	h1 := uint32(hash >> 32) % cf.M // most significant 32 bits
	fingerprint := byte(hash) // least significant 8 bits

	// if h1&1 == 0 {
	// 	return cf.AlternateIndex(h1, fingerprint), fingerprint
	// }
	return h1, fingerprint
}

func (cf *CuckooFilter) AlternateIndex(h1 uint32, fingerprint byte) uint32 {
	cf.FPHasher.Reset()
	cf.FPHasher.Write([]byte{fingerprint})
	fphash := uint32(cf.FPHasher.Sum64() >> 32) % cf.M

	return (h1 ^ fphash)
}

func (cf *CuckooFilter) InsertFingerprint(fingerprint byte, h uint32, kickingIdx uint32) bool {
	if kickingIdx > MaxKicks {
		cf.Stash = append(cf.Stash, fingerprint)
		return false
	}

	if len(cf.Buckets[h]) < BucketSize {
		cf.Buckets[h] = append(cf.Buckets[h], fingerprint)
		return true
	}

	// kick a random bucket to avoid going through the same graph cycle
	randomIndex := rand.Intn(BucketSize)
	kickedFingerprint := cf.Buckets[h][randomIndex]
	cf.Buckets[h][randomIndex] = fingerprint

	return cf.InsertFingerprint(kickedFingerprint, cf.AlternateIndex(h, kickedFingerprint), kickingIdx+1)
}

func (cf *CuckooFilter) Insert(data []byte) bool {
	h1, fingerprint := cf.Hash(data)
	if len(cf.Buckets[h1]) < BucketSize {
		cf.Buckets[h1] = append(cf.Buckets[h1], fingerprint)
		return true
	}
	h2 := cf.AlternateIndex(h1, fingerprint)
	if len(cf.Buckets[h2]) < BucketSize {
		cf.Buckets[h2] = append(cf.Buckets[h2], fingerprint)
		return true
	}
	return cf.InsertFingerprint(fingerprint, h1, 0)
}

func (cf *CuckooFilter) Lookup(data []byte) bool {
	h1, fingerprint := cf.Hash(data)
	if slices.Contains(cf.Stash, fingerprint) {
		return true
	}

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
	if slices.Contains(cf.Stash, fingerprint) {
		idx := slices.Index(cf.Stash, fingerprint)
		cf.Stash = slices.Delete(cf.Stash, idx, idx+1)
		return true
	}

	for i, val := range cf.Buckets[h1] {
		if val == fingerprint {
			cf.Buckets[h1] = slices.Delete(cf.Buckets[h1], i, i+1)
			return true
		}
	}

	h2 := cf.AlternateIndex(h1, fingerprint)
	for i, val := range cf.Buckets[h2] {
		if val == fingerprint {
			cf.Buckets[h2] = slices.Delete(cf.Buckets[h2], i, i+1)
			return true
		}
	}

	return false
}
