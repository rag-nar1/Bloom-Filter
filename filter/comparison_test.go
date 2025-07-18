package filter_test

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/rag-nar1/Filters/filter/bloom"
	"github.com/rag-nar1/Filters/filter/cuckoo"
)

const (
	N                = 5000000
	BloomFPRate      = 0.01
	CuckooLoadFactor = 0.95
)

func generateData(n int) [][]byte {
	data := make([][]byte, n)
	for i := 0; i < n; i++ {
		data[i] = []byte(fmt.Sprintf("testdata%d", i))
	}
	return data
}

func TestBloomVsCuckooComparison(t *testing.T) {
	// Generate test data
	insertedData := generateData(N)
	nonInsertedData := generateData(2 * N)[N:]

	// Create filters
	bloomFilter := bloom.NewBloomFilter(N, BloomFPRate)
	cuckooFilter := cuckoo.NewCuckooFilter(N, CuckooLoadFactor)

	// --- Benchmarking Bloom Filter ---
	// Memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	initialMem := memStats.Alloc
	bloomFilter = bloom.NewBloomFilter(N, BloomFPRate)
	runtime.ReadMemStats(&memStats)
	bloomMemUsage := memStats.Alloc - initialMem

	// Insert time
	startTime := time.Now()
	for _, item := range insertedData {
		bloomFilter.Insert(item)
	}
	bloomInsertTime := time.Since(startTime)
	bloomAvgInsertTime := bloomInsertTime / N

	// Lookup time (existing items)
	startTime = time.Now()
	for _, item := range insertedData {
		bloomFilter.Exist(item)
	}
	bloomLookupTimeExisting := time.Since(startTime)

	// Lookup time (non-existing items)
	startTime = time.Now()
	for _, item := range nonInsertedData {
		bloomFilter.Exist(item)
	}
	bloomLookupTimeNonExisting := time.Since(startTime)
	bloomAvgLookupTime := (bloomLookupTimeExisting + bloomLookupTimeNonExisting) / (2 * N)

	// False Positive Rate
	fpCount := 0
	for _, item := range nonInsertedData {
		if bloomFilter.Exist(item) {
			fpCount++
		}
	}
	bloomFPR := float64(fpCount) / float64(len(nonInsertedData))

	// Throughput
	bloomThroughput := float64(N) / bloomInsertTime.Seconds()

	// --- Benchmarking Cuckoo Filter ---
	// Memory usage
	runtime.ReadMemStats(&memStats)
	initialMem = memStats.Alloc
	cuckooFilter = cuckoo.NewCuckooFilter(N, CuckooLoadFactor)
	runtime.ReadMemStats(&memStats)
	cuckooMemUsage := memStats.Alloc - initialMem

	// Insert time
	startTime = time.Now()
	for _, item := range insertedData {
		cuckooFilter.Insert(item)
	}
	cuckooInsertTime := time.Since(startTime)
	cuckooAvgInsertTime := cuckooInsertTime / N

	// Lookup time (existing items)
	startTime = time.Now()
	for _, item := range insertedData {
		cuckooFilter.Lookup(item)
	}
	cuckooLookupTimeExisting := time.Since(startTime)

	// Lookup time (non-existing items)
	startTime = time.Now()
	for _, item := range nonInsertedData {
		cuckooFilter.Lookup(item)
	}
	cuckooLookupTimeNonExisting := time.Since(startTime)
	cuckooAvgLookupTime := (cuckooLookupTimeExisting + cuckooLookupTimeNonExisting) / (2 * N)

	// False Positive Rate
	fpCount = 0
	for _, item := range nonInsertedData {
		if cuckooFilter.Lookup(item) {
			fpCount++
		}
	}
	cuckooFPR := float64(fpCount) / float64(len(nonInsertedData))

	// Throughput
	cuckooThroughput := float64(N) / cuckooInsertTime.Seconds()

	// --- Print Results ---
	// Convert to desired units
	bloomMemUsageMB := float64(bloomMemUsage) / (1024 * 1024)
	cuckooMemUsageMB := float64(cuckooMemUsage) / (1024 * 1024)
	bloomThroughputKops := bloomThroughput / 1000
	cuckooThroughputKops := cuckooThroughput / 1000

	fmt.Println("\n--- Filter Comparison Results ---")
	fmt.Printf("Number of items (N): %d\n\n", N)
	fmt.Println("| Metric                | Bloom Filter      | Cuckoo Filter     |")
	fmt.Println("|-----------------------|-------------------|-------------------|")
	fmt.Printf("| Memory Usage (MB)     | %-17.2f | %-17.2f |\n", bloomMemUsageMB, cuckooMemUsageMB)
	fmt.Printf("| Avg. Insert Time (ns) | %-17d | %-17d |\n", bloomAvgInsertTime.Nanoseconds(), cuckooAvgInsertTime.Nanoseconds())
	fmt.Printf("| Avg. Lookup Time (ns) | %-17d | %-17d |\n", bloomAvgLookupTime.Nanoseconds(), cuckooAvgLookupTime.Nanoseconds())
	fmt.Printf("| FPR (%%)               | %-17.4f | %-17.4f |\n", bloomFPR*100, cuckooFPR*100)
	fmt.Printf("| Throughput (kops)     | %-17.2f | %-17.2f |\n", bloomThroughputKops, cuckooThroughputKops)
}

func init() {
	// Seed random number generator for deterministic results
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
