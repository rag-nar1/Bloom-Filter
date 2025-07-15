package bloom_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	filter "github.com/rag-nar1/Bloom-Filter/filter/bloom"
)

// BenchmarkPerformance measures Insert and Exist operation performance
func BenchmarkPerformance(b *testing.B) {
	n := 100000
	fpRate := 0.01
	bf := filter.NewBloomFilter(uint64(n), fpRate)
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
	n := 50000
	fpRate := 0.01
	bf := filter.NewBloomFilter(uint64(n), fpRate)

	// Insert known items (simulating real usage)
	knownItems := make([][]byte, n)
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

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark Exist operations on unknown items
	for i := 0; i < b.N; i++ {
		bf.Exist(testItems[i%len(testItems)])
	}

	// Report accuracy metrics
	b.ReportMetric(falsePositiveRate*100, "actual_fpr_%")
	b.ReportMetric(fpRate*100, "theoretical_fpr_%")
}

// BenchmarkMemoryEfficiency measures memory usage and efficiency
func BenchmarkMemoryEfficiency(b *testing.B) {
	var m1, m2 runtime.MemStats

	// Get baseline memory
	runtime.GC()
	runtime.ReadMemStats(&m1)

	n := 100000
	fpRate := 0.01
	bf := filter.NewBloomFilter(uint64(n), fpRate)

	// Insert a significant number of items
	for i := 0; i < n; i++ {
		bf.Insert([]byte(fmt.Sprintf("memory_test_item_%d", i)))
	}

	// Measure memory after operations
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Calculate memory metrics
	actualMemoryBytes := m2.Alloc - m1.Alloc
	theoreticalMemoryBytes := float64(bf.M) / 8 // bits to bytes
	memoryEfficiency := (theoreticalMemoryBytes / float64(actualMemoryBytes)) * 100
	bitsPerItem := float64(bf.M) / float64(n)

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

// BenchmarkBigDataOperations tests performance with large data sets over multiple iterations and reports average metrics
func BenchmarkBigDataOperations(b *testing.B) {
	// Configuration
	n := 500000
	fpRate := 0.01
	bigDataSize := 1024 // 1KB per item
	nonExistentCount := 15000
	iterations := 5 // Number of iterations to average

	// Variables to accumulate metrics across iterations
	totalFalsePositiveRate := 0.0
	totalIterationTime := time.Duration(0)

	// Start total time tracking
	totalStartTime := time.Now()

	// Run multiple iterations
	for iteration := 0; iteration < iterations; iteration++ {
		// Start iteration timer
		iterationStartTime := time.Now()

		// Create a fresh bloom filter for each iteration
		bf := filter.NewBloomFilter(uint64(n), fpRate)

		// Generate big data items to insert
		insertedItems := make([][]byte, n)

		// Create large data items with varied content
		for i := 0; i < n; i++ {
			data := make([]byte, bigDataSize)
			// Fill with pattern to make each item unique (include iteration to ensure variety)
			pattern := fmt.Sprintf("big_data_item_%d_%d_", iteration, i)
			copy(data, []byte(pattern))
			// Fill rest with repeating pattern
			for j := len(pattern); j < bigDataSize; j++ {
				data[j] = byte((iteration + i + j) % 256)
			}
			insertedItems[i] = data
			bf.Insert(data)
		}

		// Generate non-existent big data items for false positive testing
		nonExistentItems := make([][]byte, nonExistentCount)
		for i := 0; i < nonExistentCount; i++ {
			data := make([]byte, bigDataSize)
			pattern := fmt.Sprintf("non_existent_big_data_%d_%d_", iteration, i)
			copy(data, []byte(pattern))
			// Fill rest with different pattern
			for j := len(pattern); j < bigDataSize; j++ {
				data[j] = byte((iteration*3 + i*2 + j + 128) % 256)
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

		// Calculate metrics for this iteration
		falsePositiveRate := float64(falsePositives) / float64(len(nonExistentItems)) * 100

		// Accumulate metrics
		totalFalsePositiveRate += falsePositiveRate

		// Record iteration time
		iterationTime := time.Since(iterationStartTime)
		totalIterationTime += iterationTime
	}

	// Calculate total time
	totalTime := time.Since(totalStartTime)

	// Calculate averages
	avgFalsePositiveRate := totalFalsePositiveRate / float64(iterations)
	avgIterationTime := totalIterationTime / time.Duration(iterations)

	// Create final bloom filter for benchmarking
	bf := filter.NewBloomFilter(uint64(n), fpRate)
	finalItems := make([][]byte, n+nonExistentCount)

	// Add both inserted and non-existent items for mixed testing
	for i := 0; i < n; i++ {
		data := make([]byte, bigDataSize)
		pattern := fmt.Sprintf("final_big_data_item_%d_", i)
		copy(data, []byte(pattern))
		for j := len(pattern); j < bigDataSize; j++ {
			data[j] = byte((i + j) % 256)
		}
		finalItems[i] = data
		bf.Insert(data)
	}

	for i := 0; i < nonExistentCount; i++ {
		data := make([]byte, bigDataSize)
		pattern := fmt.Sprintf("final_non_existent_%d_", i)
		copy(data, []byte(pattern))
		for j := len(pattern); j < bigDataSize; j++ {
			data[j] = byte((i*2 + j + 128) % 256)
		}
		finalItems[n+i] = data
	}

	b.ResetTimer()

	// Benchmark mixed operations on big data
	for i := 0; i < b.N; i++ {
		bf.Exist(finalItems[i%len(finalItems)])
	}

	// Report comprehensive averaged metrics
	b.ReportMetric(avgFalsePositiveRate, "avg_actual_fpr_%")
	b.ReportMetric(fpRate*100, "theoretical_fpr_%")
	b.ReportMetric(float64(iterations), "iterations#")
	b.ReportMetric(float64(n), "items_inserted_per_iteration")
	b.ReportMetric(float64(nonExistentCount), "items_tested_non_existent_per_iteration")
	b.ReportMetric(float64(bigDataSize), "avg_data_size_bytes")
	b.ReportMetric(float64(bf.M)/8/1024, "filter_size_KB")
	b.ReportMetric(float64(avgIterationTime.Milliseconds()), "avg_iteration_time_ms")
	b.ReportMetric(float64(totalTime.Milliseconds()), "total_time_ms")
}
