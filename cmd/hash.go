package BloomFilter

type HashingFun struct {
	p     int
	m     int
	p_pow []int
}

func NewHashingfunc(p, m int) *HashingFun {
	hashfunc := &HashingFun{
		p:     p,
		m:     m,
		p_pow: make([]int, m),
	}

	hashfunc.p_pow[0] = 1
	for i := 1; i < m; i++ {
		hashfunc.p_pow[i] = hashfunc.p_pow[i-1] * p % m
	}

	return hashfunc
}

// it works by calculating this formula
// h(x) = (x[0]*p^0 + x[1]*p^1 + ... + x[n]*p^n) % m
// where x is iteratable and n = len(x)

func (h *HashingFun) HashStr(x string) int {
	hash := 0
	for i, c := range x {
		hash += h.p_pow[i] * int(c) % h.m
		hash %= h.m
	}

	return hash
}

func (h *HashingFun) HashInt(x int) int {

	return x % h.m
}
