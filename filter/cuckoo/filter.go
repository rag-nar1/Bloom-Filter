package cuckoo

import (
	"hash"
	"hash/fnv"
	"math"
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
}

func nextPowerOfTwo(n uint32) uint32 {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return n
}

func NewCuckooFilter(n uint64, fpRate float64, loadFactor float64) *CuckooFilter {
	m := nextPowerOfTwo(uint32(math.Ceil(float64(n) / loadFactor / BucketSize)))

	return &CuckooFilter{
		M:       m,
		Buckets: make([][]byte, m),
		Hasher:  fnv.New64(),
	}
}

// returns the fingerprint and the index of the first bucket
func (cf *CuckooFilter) Hash(data []byte) (uint32, byte) {
	cf.Hasher.Reset()
	cf.Hasher.Write(data)
	hash := cf.Hasher.Sum64()

	h1 := uint32(hash >> 32) % cf.M // most significant 32 bits
	fingerprint := byte(hash%255 + 1) // least significant 8 bits

	return h1, fingerprint
}

func (cf *CuckooFilter) AlternateIndex(h1 uint32, fingerprint byte) uint32 {
	cf.Hasher.Reset()
	cf.Hasher.Write([]byte{fingerprint})
	fphash := uint32(cf.Hasher.Sum64() >> 32) % cf.M

	return (h1 ^ fphash)
}

func (cf *CuckooFilter) InsertFingerprint(fingerprint byte, h uint32, kickingIdx uint32) bool {
	if kickingIdx > MaxKicks {
		return false
	}

	idx := h % cf.M
	if len(cf.Buckets[idx]) < BucketSize {
		cf.Buckets[idx] = append(cf.Buckets[idx], fingerprint)
		return true
	}

	kickedFingerprint := cf.Buckets[idx][0]
	cf.Buckets[idx][0] = fingerprint

	return cf.InsertFingerprint(kickedFingerprint, cf.AlternateIndex(h, kickedFingerprint), kickingIdx+1)
}

func (cf *CuckooFilter) Insert(data []byte) bool {
	h1, fingerprint := cf.Hash(data)
	if len(cf.Buckets[h1%cf.M]) < BucketSize {
		cf.Buckets[h1%cf.M] = append(cf.Buckets[h1%cf.M], fingerprint)
		return true
	}
	h2 := cf.AlternateIndex(h1, fingerprint)
	if len(cf.Buckets[h2%cf.M]) < BucketSize {
		cf.Buckets[h2%cf.M] = append(cf.Buckets[h2%cf.M], fingerprint)
		return true
	}
	return cf.InsertFingerprint(fingerprint, h1, 0)
}

func (cf *CuckooFilter) Lookup(data []byte) bool {
	h1, fingerprint := cf.Hash(data)
	idx := h1 % cf.M
	for _, val := range cf.Buckets[idx] {
		if val == fingerprint {
			return true
		}
	}

	h2 := cf.AlternateIndex(h1, fingerprint)
	idx = h2 % cf.M
	for _, val := range cf.Buckets[idx] {
		if val == fingerprint {
			return true
		}
	}

	return false
}

func (cf *CuckooFilter) Delete(data []byte) bool {
	h1, fingerprint := cf.Hash(data)
	idx := h1 % cf.M
	for i, val := range cf.Buckets[idx] {
		if val == fingerprint {
			cf.Buckets[idx] = slices.Delete(cf.Buckets[idx], i, i+1)
			return true
		}
	}

	h2 := cf.AlternateIndex(h1, fingerprint)
	idx = h2 % cf.M
	for i, val := range cf.Buckets[idx] {
		if val == fingerprint {
			cf.Buckets[idx] = slices.Delete(cf.Buckets[idx], i, i+1)
			return true
		}
	}

	return false
}
