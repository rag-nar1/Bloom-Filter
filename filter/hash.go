package filter

type Hash func(data []byte, m uint64, k uint32) []int

func DoubleHash(hash uint64, m uint32, k uint32) []int {
	hashedIdx := make([]int, k)
	h1 := uint32(hash)
	h2 := uint32(hash >> 32)

	for i := uint32(0); i < k; i++ {
		hashedIdx[i] = int((h1 + uint32(i)*h2) & (m - 1))
	}
	return hashedIdx
}
