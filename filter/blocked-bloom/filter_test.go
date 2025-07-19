package blockedbloom_test

import (
	"fmt"
	"testing"
	"time"

	blockedbloom "github.com/rag-nar1/Filters/filter/blocked-bloom"
)

func TestBenchmarkMetrics(t *testing.T) {
	const N = 1000000
	const fpRate = 0.003142
	const iterations = 10 // Number of iterations for averaging

	var totalFpCount int
	var totalInsertTime time.Duration
	var totalCheckTime time.Duration

	fmt.Println("--------------------------------")
	fmt.Println("Blocked Bloom Filter Benchmark Metrics")
	fmt.Println("--------------------------------")
	fmt.Println("N:", N)
	fmt.Println("fpRate:", fpRate)
	fmt.Println("Iterations:", iterations)
	fmt.Println("--------------------------------")

	for i := 0; i < iterations; i++ {
		bf := blockedbloom.NewBlockedBloomFilter(uint64(N), fpRate)

		itemsToInsert := make([][]byte, N)
		for j := 0; j < N; j++ {
			itemsToInsert[j] = []byte(fmt.Sprintf("inserted_%d_%d", i, j))
		}

		startInsert := time.Now()
		for _, item := range itemsToInsert {
			bf.Insert(item)
		}
		totalInsertTime += time.Since(startInsert)

		itemsToCheck := make([][]byte, N)
		for j := 0; j < N; j++ {
			itemsToCheck[j] = []byte(fmt.Sprintf("not_inserted_%d_%d", i, j))
		}

		fpCount := 0
		startCheck := time.Now()
		for _, item := range itemsToCheck {
			if bf.Exist(item) {
				fpCount++
			}
		}
		totalCheckTime += time.Since(startCheck)
		totalFpCount += fpCount
	}

	avgFpCount := float64(totalFpCount) / float64(iterations)
	avgFprPercentage := (avgFpCount / float64(N)) * 100
	avgInsertNsOp := (float64(totalInsertTime.Nanoseconds()) / float64(N)) / float64(iterations)
	avgCheckNsOp := (float64(totalCheckTime.Nanoseconds()) / float64(N)) / float64(iterations)

	fmt.Printf("Avg FPR (%%)\tAvg FP Count\tAvg Insert (ns/op)\tAvg Check (ns/op)\n")
	fmt.Printf("%.4f \t\t %.2f \t\t %.2f \t\t\t %.2f \n\n",
		avgFprPercentage,
		avgFpCount,
		avgInsertNsOp,
		avgCheckNsOp,
	)
}
