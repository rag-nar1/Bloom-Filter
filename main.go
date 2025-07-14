package main

const (
	N = 100_000_000
	primesLen = 6_820_000 // approxmatino using 1.25506*N/ln(N)
)

var ( // used for linear sieve
	spf    [N]int
	prime []int
)

func init() {
	// implementation of linear sieve to get some primes
	prime = make([]int, primesLen)
	for i := range N {
		spf[i] = 0
	}
	for i := range N {
		if spf[i] == 0 {
			spf[i] = i
			prime = append(prime, i)
		}
		for j := 0; prime[j] * i < N; j ++ {
			spf[i*prime[j]] = prime[j]
			if i%prime[j] == 0 {
				break
			}
		}
	}
	
	prime = append([]int(nil), prime...)
}

func GetRandPrimeRange(l, r int) int {
	
}
