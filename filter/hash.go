package filter

import (
	"hash/fnv"
)

type Hash func(data []byte, m uint64, k int) []int

func DoubleHash(data []byte, m uint64, k int) []int {
	hashedIdx := make([]int, k)
	hasher := fnv.New64()
	hasher.Write(data)
	hash := hasher.Sum64()
	h1 := uint32(hash)
	h2 := uint32(hash >> 32)

	for i := 0; i < k; i++ {
		hashedIdx[i] = int((h1 + uint32(i)*h2) % uint32(m))
	}
	return hashedIdx
}