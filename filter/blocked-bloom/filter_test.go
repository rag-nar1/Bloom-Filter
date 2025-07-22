package blockedbloom_test

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
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

	var bf *blockedbloom.BlockedBloomFilter
	for i := 0; i < iterations; i++ {
		bf = blockedbloom.NewBlockedBloomFilter(uint64(N), fpRate)

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

		// check for false negatives
		for _, item := range itemsToInsert {
			if !bf.Exist(item) {
				t.Errorf("Item %s should exist in the filter", item)
			}
		}
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

	// test the same benchmark with aginst bloom filter from github.com/bits-and-blooms/bloom/v3
	totalInsertTime = 0
	totalCheckTime = 0
	totalFpCount = 0
	for i := 0; i < iterations; i++ {
		bloomFilter := bloom.NewWithEstimates(uint(N), fpRate)

		itemsToInsert := make([][]byte, N)
		for j := 0; j < N; j++ {
			itemsToInsert[j] = []byte(fmt.Sprintf("inserted_%d_%d", i, j))
		}


		startInsert := time.Now()
		for _, item := range itemsToInsert {
			bloomFilter.Add(item)
		}
		totalInsertTime += time.Since(startInsert)

		itemsToCheck := make([][]byte, N)
		for j := 0; j < N; j++ {
			itemsToCheck[j] = []byte(fmt.Sprintf("not_inserted_%d_%d", i, j))
		}

		fpCount := 0
		startCheck := time.Now()
		for _, item := range itemsToCheck {
			if bloomFilter.Test(item) {
				fpCount++
			}
		}
		totalCheckTime += time.Since(startCheck)
		totalFpCount += fpCount

		// check for false negatives
		for _, item := range itemsToInsert {
			if !bloomFilter.Test(item) {
				t.Errorf("Item %s should exist in the filter", item)
			}
		}
	}
	fmt.Println("--------------------------------")
	avgFpCount = float64(totalFpCount) / float64(iterations)
	avgFprPercentage = (avgFpCount / float64(N)) * 100
	avgInsertNsOp = (float64(totalInsertTime.Nanoseconds()) / float64(N)) / float64(iterations)
	avgCheckNsOp = (float64(totalCheckTime.Nanoseconds()) / float64(N)) / float64(iterations)
	fmt.Printf("Avg FPR (%%)\tAvg FP Count\tAvg Insert (ns/op)\tAvg Check (ns/op)\n")
	fmt.Printf("%.4f \t\t %.2f \t\t %.2f \t\t\t %.2f \n\n",
		avgFprPercentage,
		avgFpCount,
		avgInsertNsOp,
		avgCheckNsOp,
	)
}

func BenchmarkSeparateTestAndAdd(b *testing.B) {
	f := blockedbloom.NewBlockedBloomFilter(uint64(b.N), 0.0001)
	key := make([]byte, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint32(key, uint32(i))
		f.Insert(key)
		f.Exist(key)
	}
}

func BenchmarkSeparateTestAndAddBloom(b *testing.B) {
	f := bloom.NewWithEstimates(uint(b.N), 0.0001)
	key := make([]byte, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint32(key, uint32(i))
		f.Test(key)
		f.Add(key)
	}
}