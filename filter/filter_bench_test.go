package filter_test

import (
	"fmt"
	"math"
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

// BenchmarkBigDataOperations tests performance with large data sets and reports comprehensive metrics
func BenchmarkBigDataOperations(b *testing.B) {
	// Create a larger bloom filter for big data testing
	filterSize := 500000
	hashFunctions := 7
	bf := filter.NewBloomFilter(filterSize, hashFunctions)

	// Generate big data items to insert
	bigDataCount := 25000
	bigDataSize := 1024 // 1KB per item
	insertedItems := make([][]byte, bigDataCount)

	// Create large data items with varied content
	for i := 0; i < bigDataCount; i++ {
		data := make([]byte, bigDataSize)
		// Fill with pattern to make each item unique
		pattern := fmt.Sprintf("big_data_item_%d_", i)
		copy(data, []byte(pattern))
		// Fill rest with repeating pattern
		for j := len(pattern); j < bigDataSize; j++ {
			data[j] = byte((i + j) % 256)
		}
		insertedItems[i] = data
		bf.Insert(data)
	}

	// Generate non-existent big data items for false positive testing
	nonExistentCount := 15000
	nonExistentItems := make([][]byte, nonExistentCount)
	for i := 0; i < nonExistentCount; i++ {
		data := make([]byte, bigDataSize)
		pattern := fmt.Sprintf("non_existent_big_data_%d_", i)
		copy(data, []byte(pattern))
		// Fill rest with different pattern
		for j := len(pattern); j < bigDataSize; j++ {
			data[j] = byte((i*2 + j + 128) % 256)
		}
		nonExistentItems[i] = data
	}

	// Test existence of inserted items (should all be true)
	existentHits := 0
	for _, item := range insertedItems {
		if bf.Exist(item) {
			existentHits++
		}
	}

	// Test existence of non-existent items (count false positives)
	falsePositives := 0
	for _, item := range nonExistentItems {
		if bf.Exist(item) {
			falsePositives++
		}
	}

	// Calculate metrics
	hitRate := float64(existentHits) / float64(len(insertedItems)) * 100
	falsePositiveRate := float64(falsePositives) / float64(len(nonExistentItems)) * 100
	totalDataSizeMB := float64(bigDataCount*bigDataSize) / (1024 * 1024)

	// Calculate theoretical false positive rate
	// p = pow(1 - exp(-k / (m / n)), k)
	k := float64(hashFunctions)
	n := float64(bigDataCount)
	m := float64(filterSize)
	theoreticalFPR := math.Pow(1.0-math.Exp(-k/(m/n)), k)

	b.ResetTimer()

	// Benchmark mixed operations on big data
	allItems := append(insertedItems, nonExistentItems...)
	for i := 0; i < b.N; i++ {
		bf.Exist(allItems[i%len(allItems)])
	}

	// Report comprehensive metrics
	b.ReportMetric(hitRate, "hit_rate_%")
	b.ReportMetric(falsePositiveRate, "actual_fpr_%")
	b.ReportMetric(theoreticalFPR*100, "theoretical_fpr_%")
	b.ReportMetric(float64(bigDataCount), "items_inserted")
	b.ReportMetric(float64(nonExistentCount), "items_tested_non_existent")
	b.ReportMetric(float64(bigDataSize), "avg_data_size_bytes")
	b.ReportMetric(totalDataSizeMB, "total_data_size_MB")
	b.ReportMetric(float64(filterSize)/8/1024, "filter_size_KB")
}
