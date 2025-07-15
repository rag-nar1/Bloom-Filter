package filter_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/rag-nar1/Bloom-Filter/filter"
)

// BenchmarkPerformance measures Insert and Exist operation performance
func BenchmarkPerformance(b *testing.B) {
	bf := filter.NewBloomFilter(100000, 5)
	testData := []byte("performance test data")

	// Pre-insert the data for Exist testing
	bf.Insert(testData)

	b.Run("Insert", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			bf.Insert(testData)
		}
	})

	b.Run("Exist", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			bf.Exist(testData)
		}
	})
}

// BenchmarkAccuracy measures false positive rate and filter accuracy
func BenchmarkAccuracy(b *testing.B) {
	bf := filter.NewBloomFilter(50000, 5)

	// Insert known items (simulating real usage)
	knownItems := make([][]byte, 5000)
	for i := range knownItems {
		knownItems[i] = []byte(fmt.Sprintf("known_item_%d", i))
		bf.Insert(knownItems[i])
	}

	// Test items that should NOT be in the filter
	testItems := make([][]byte, 10000)
	for i := range testItems {
		testItems[i] = []byte(fmt.Sprintf("unknown_item_%d", i))
	}

	// Measure false positive rate
	falsePositives := 0
	for _, item := range testItems {
		if bf.Exist(item) {
			falsePositives++
		}
	}

	// Calculate metrics
	falsePositiveRate := float64(falsePositives) / float64(len(testItems))

	// Calculate theoretical false positive rate: (1 - e^(-k*n/m))^k
	k := 5.0                      // hash functions
	n := float64(len(knownItems)) // items inserted
	m := 50000.0                  // filter size
	theoreticalFPR := 1.0
	for i := 0; i < 5; i++ {
		theoreticalFPR *= (1.0 - (1.0 / (1.0 + (k*n)/m)))
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark Exist operations on unknown items
	for i := 0; i < b.N; i++ {
		bf.Exist(testItems[i%len(testItems)])
	}

	// Report accuracy metrics
	b.ReportMetric(falsePositiveRate*100, "actual_fpr_%")
	b.ReportMetric(theoreticalFPR*100, "theoretical_fpr_%")
}

// BenchmarkMemoryEfficiency measures memory usage and efficiency
func BenchmarkMemoryEfficiency(b *testing.B) {
	var m1, m2 runtime.MemStats

	// Get baseline memory
	runtime.GC()
	runtime.ReadMemStats(&m1)

	filterSize := 100000
	bf := filter.NewBloomFilter(filterSize, 5)

	// Insert a significant number of items
	itemCount := 10000
	for i := 0; i < itemCount; i++ {
		bf.Insert([]byte(fmt.Sprintf("memory_test_item_%d", i)))
	}

	// Measure memory after operations
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Calculate memory metrics
	actualMemoryBytes := m2.Alloc - m1.Alloc
	theoreticalMemoryBytes := float64(filterSize) / 8 // bits to bytes
	memoryEfficiency := (theoreticalMemoryBytes / float64(actualMemoryBytes)) * 100
	bitsPerItem := float64(filterSize) / float64(itemCount)

	b.ResetTimer()
	b.ReportAllocs()

	testData := []byte("memory benchmark data")
	for i := 0; i < b.N; i++ {
		bf.Insert(testData)
	}

	// Report memory metrics
	b.ReportMetric(float64(actualMemoryBytes)/1024, "actual_memory_KB")
	b.ReportMetric(theoreticalMemoryBytes/1024, "theoretical_memory_KB")
	b.ReportMetric(memoryEfficiency, "memory_efficiency_%")
	b.ReportMetric(bitsPerItem, "bits_per_item")
}
