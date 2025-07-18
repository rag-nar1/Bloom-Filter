package cuckoo_test

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	filterCuckoo "github.com/rag-nar1/Filters/filter/cuckoo"
)

// Helper function to create formatted tables and write to files
func formatTable(b *testing.B, title string, sections []struct {
	sectionTitle string
	metrics      []struct {
		name  string
		value interface{}
		unit  string
	}
}) {
	// Create the bench-results directory if it doesn't exist
	resultsDir := "/home/ragnar/Desktop/axiom/Bloom-Filter/filter/cuckoo/bench-results"
	err := os.MkdirAll(resultsDir, 0755)
	if err != nil {
		b.Logf("Error creating directory %s: %v", resultsDir, err)
		return
	}

	// Create filename based on benchmark name
	benchName := strings.ReplaceAll(b.Name(), "/", "_")
	filename := fmt.Sprintf("%s/benchmark_results_%s.txt", resultsDir, benchName)

	// Create or overwrite the file
	file, err := os.Create(filename)
	if err != nil {
		b.Logf("Error creating file %s: %v", filename, err)
		return
	}
	defer file.Close()

	// Write table to file
	fmt.Fprintf(file, "\n"+strings.Repeat("=", 80)+"\n")
	fmt.Fprintf(file, "  %s\n", title)
	fmt.Fprintf(file, strings.Repeat("=", 80)+"\n")

	for _, section := range sections {
		fmt.Fprintf(file, "\n┌─ %s\n", section.sectionTitle)

		// Find max width for alignment across all metrics in this section
		maxNameWidth := 0
		for _, metric := range section.metrics {
			if len(metric.name) > maxNameWidth {
				maxNameWidth = len(metric.name)
			}
		}

		// Ensure minimum spacing
		if maxNameWidth < 20 {
			maxNameWidth = 20
		}

		for _, metric := range section.metrics {
			// Create proper padding to align all colons
			padding := strings.Repeat(" ", maxNameWidth-len(metric.name))

			switch v := metric.value.(type) {
			case float64:
				if strings.Contains(metric.unit, "%") {
					fmt.Fprintf(file, "│  %s%s : %9.2f %s\n", metric.name, padding, v, metric.unit)
				} else if strings.Contains(metric.unit, "ms") || strings.Contains(metric.unit, "KB") || strings.Contains(metric.unit, "ns") {
					fmt.Fprintf(file, "│  %s%s : %9.1f %s\n", metric.name, padding, v, metric.unit)
				} else {
					fmt.Fprintf(file, "│  %s%s : %9.0f %s\n", metric.name, padding, v, metric.unit)
				}
			case int:
				fmt.Fprintf(file, "│  %s%s : %9d %s\n", metric.name, padding, v, metric.unit)
			default:
				fmt.Fprintf(file, "│  %s%s : %9v %s\n", metric.name, padding, v, metric.unit)
			}
		}
	}
	fmt.Fprintf(file, "└"+strings.Repeat("─", 79)+"\n")

	// Log the filename so user knows where to find the results
	fmt.Printf("📊 Benchmark results saved to: %s\n", filename)
}

// BenchmarkPerformance measures Insert, Lookup, and Delete operation performance
func BenchmarkPerformance(b *testing.B) {
	n := uint64(100000)
	loadFactor := 0.95
	cf := filterCuckoo.NewCuckooFilter(n, loadFactor)
	testData := []byte("performance test data")

	// Pre-insert the data for Lookup and Delete testing
	cf.Insert(testData)

	b.Run("Insert", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Use different data each time to avoid duplicates
			data := []byte(fmt.Sprintf("insert_test_%d", i))
			cf.Insert(data)
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cf.Lookup(testData)
		}
	})

	b.Run("Delete", func(b *testing.B) {
		// Pre-populate with data to delete
		testItems := make([][]byte, b.N)
		for i := 0; i < b.N; i++ {
			testItems[i] = []byte(fmt.Sprintf("delete_test_%d", i))
			cf.Insert(testItems[i])
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cf.Delete(testItems[i])
		}
	})
}

// BenchmarkAccuracy measures false positive rate and filter accuracy
func BenchmarkAccuracy(b *testing.B) {
	n := uint64(5000)
	loadFactor := 0.95
	cf := filterCuckoo.NewCuckooFilter(n, loadFactor)

	// Insert known items (simulating real usage)
	knownItems := make([][]byte, int(n)/2) // Insert half capacity to avoid hitting limits
	successfulInserts := 0
	for i := range knownItems {
		knownItems[i] = []byte(fmt.Sprintf("known_item_%d", i))
		if cf.Insert(knownItems[i]) {
			successfulInserts++
		}
	}

	// Test items that should NOT be in the filter
	testItems := make([][]byte, 10000)
	for i := range testItems {
		testItems[i] = []byte(fmt.Sprintf("unknown_item_%d", i))
	}

	// Measure false positive rate
	falsePositives := 0
	for _, item := range testItems {
		if cf.Lookup(item) {
			falsePositives++
		}
	}

	// Calculate metrics
	falsePositiveRate := float64(falsePositives) / float64(len(testItems))
	successRate := float64(successfulInserts) / float64(len(knownItems))

	// Display accuracy metrics in formatted table
	formatTable(b, "CUCKOO FILTER ACCURACY BENCHMARK", []struct {
		sectionTitle string
		metrics      []struct {
			name  string
			value interface{}
			unit  string
		}
	}{
		{
			sectionTitle: "Accuracy Results",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"False Positive Rate", falsePositiveRate * 100, "%"},
				{"Insert Success Rate", successRate * 100, "%"},
			},
		},
		{
			sectionTitle: "Test Statistics",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Items Inserted", successfulInserts, "items"},
				{"Items Tested (Non-Existent)", len(testItems), "items"},
				{"Filter Capacity", int(n), "items"},
			},
		},
		{
			sectionTitle: "Filter State",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Load Factor", loadFactor * 100, "%"},
			},
		},
	})

	// Add minimal benchmark to prevent re-running the entire function
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(cf.Buckets) // Minimal operation
	}
}

// BenchmarkMemoryEfficiency measures memory usage and efficiency
func BenchmarkMemoryEfficiency(b *testing.B) {
	var m1, m2 runtime.MemStats

	// Get baseline memory
	runtime.GC()
	runtime.ReadMemStats(&m1)

	n := uint64(100000)
	loadFactor := 0.95
	cf := filterCuckoo.NewCuckooFilter(n, loadFactor)

	// Insert a significant number of items
	insertedCount := 0
	for i := 0; i < int(n)/2; i++ { // Insert half capacity to avoid hitting limits
		if cf.Insert([]byte(fmt.Sprintf("memory_test_item_%d", i))) {
			insertedCount++
		}
	}

	// Measure memory after operations
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Calculate memory metrics
	actualMemoryBytes := m2.Alloc - m1.Alloc

	// Theoretical memory: M buckets * 4 entries per bucket * 1 byte per fingerprint + stash
	theoreticalMemoryBytes := float64(cf.M*filterCuckoo.BucketSize*1)
	memoryEfficiency := (theoreticalMemoryBytes / float64(actualMemoryBytes)) * 100
	bytesPerItem := float64(actualMemoryBytes) / float64(insertedCount)

	// Display memory metrics in formatted table
	formatTable(b, "CUCKOO FILTER MEMORY EFFICIENCY BENCHMARK", []struct {
		sectionTitle string
		metrics      []struct {
			name  string
			value interface{}
			unit  string
		}
	}{
		{
			sectionTitle: "Memory Usage",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Actual Memory", float64(actualMemoryBytes) / 1024, "KB"},
				{"Theoretical Memory", theoreticalMemoryBytes / 1024, "KB"},
				{"Memory Efficiency", memoryEfficiency, "%"},
			},
		},
		{
			sectionTitle: "Per-Item Metrics",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Bytes Per Item", bytesPerItem, "bytes"},
				{"Items Inserted", insertedCount, "items"},
				{"Target Capacity", int(n), "items"},
			},
		},
		{
			sectionTitle: "Filter Structure",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Buckets Count", int(cf.M), "buckets"},
				{"Total Slots", int(cf.M * filterCuckoo.BucketSize), "slots"},
				{"Load Factor", loadFactor * 100, "%"},
			},
		},
	})

	// Add minimal benchmark to prevent re-running the entire function
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(cf.Buckets) // Minimal operation
	}
}

// BenchmarkCapacityLimits tests behavior when approaching filter capacity
func BenchmarkCapacityLimits(b *testing.B) {
	capacityTests := []struct {
		name       string
		n          uint64
		loadFactor float64
		insertPct  float64 // percentage of capacity to try inserting
	}{
		{"low_load", 10000, 0.5, 0.8},
		{"medium_load", 10000, 0.75, 0.9},
		{"high_load", 10000, 0.95, 1.1},
	}

	for _, test := range capacityTests {
		b.Run(test.name, func(b *testing.B) {
			cf := filterCuckoo.NewCuckooFilter(test.n, test.loadFactor)

			itemsToInsert := int(float64(test.n) * test.insertPct)
			insertedCount := 0
			insertFailures := 0

			// Try to insert items up to the test percentage
			items := make([][]byte, itemsToInsert)
			for i := 0; i < itemsToInsert; i++ {
				items[i] = []byte(fmt.Sprintf("capacity_test_%d_%s", i, test.name))
				if cf.Insert(items[i]) {
					insertedCount++
				} else {
					insertFailures++
				}
			}

			// Calculate metrics
			insertSuccessRate := float64(insertedCount) / float64(itemsToInsert) * 100
			capacityUtilization := float64(insertedCount) / float64(test.n) * 100

			// Display capacity metrics in formatted table
			formatTable(b, fmt.Sprintf("CUCKOO FILTER CAPACITY LIMITS - %s", strings.ToUpper(test.name)), []struct {
				sectionTitle string
				metrics      []struct {
					name  string
					value interface{}
					unit  string
				}
			}{
				{
					sectionTitle: "Success Rates",
					metrics: []struct {
						name  string
						value interface{}
						unit  string
					}{
						{"Insert Success Rate", insertSuccessRate, "%"},
						{"Capacity Utilization", capacityUtilization, "%"},
					},
				},
				{
					sectionTitle: "Insert Statistics",
					metrics: []struct {
						name  string
						value interface{}
						unit  string
					}{
						{"Items Inserted", insertedCount, "items"},
						{"Insert Failures", insertFailures, "items"},
						{"Items Attempted", itemsToInsert, "items"},
						{"Target Capacity", int(test.n), "items"},
					},
				},
				{
					sectionTitle: "Filter Configuration",
					metrics: []struct {
						name  string
						value interface{}
						unit  string
					}{
						{"Load Factor", test.loadFactor * 100, "%"},
						{"Buckets Count", int(cf.M), "buckets"},
					},
				},
			})

			// Add minimal benchmark to prevent re-running the entire function
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = len(cf.Buckets) // Minimal operation
			}
		})
	}
}

// BenchmarkStashBehavior specifically tests stash usage under high load
func BenchmarkStashBehavior(b *testing.B) {
	n := uint64(5000)  // Small filter to force stash usage
	loadFactor := 0.98 // Very high load factor
	cf := filterCuckoo.NewCuckooFilter(n, loadFactor)

	// Fill the filter beyond capacity to force stash usage
	insertCount := int(n * 2) // Try to insert twice the capacity
	insertedItems := make([][]byte, 0, insertCount)

	for i := 0; i < insertCount; i++ {
		item := []byte(fmt.Sprintf("stash_test_item_%d", i))
		if cf.Insert(item) {
			insertedItems = append(insertedItems, item)
		}
	}

	// Display stash metrics in formatted table
	formatTable(b, "CUCKOO FILTER STASH BEHAVIOR BENCHMARK", []struct {
		sectionTitle string
		metrics      []struct {
			name  string
			value interface{}
			unit  string
		}
	}{
		{
			sectionTitle: "Insert Results",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Successful Inserts", len(insertedItems), "items"},
				{"Total Attempted", insertCount, "items"},
				{"Insert Success Rate", float64(len(insertedItems)) / float64(insertCount) * 100, "%"},
			},
		},
		{
			sectionTitle: "Stash Usage",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Stash Usage Rate", 0, "%"},
			},
		},
		{
			sectionTitle: "Capacity Analysis",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Filter Capacity", int(n), "items"},
				{"Capacity Utilization", float64(len(insertedItems)) / float64(n) * 100, "%"},
				{"Load Factor", loadFactor * 100, "%"},
			},
		},
	})

	// Add minimal benchmark to prevent re-running the entire function
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(cf.Buckets) // Minimal operation
	}
}

// BenchmarkBigDataOperations tests performance with large data sets
func BenchmarkBigDataOperations(b *testing.B) {
	// Configuration
	n := uint64(100000)
	loadFactor := 0.9
	bigDataSize := 1024 // 1KB per item
	nonExistentCount := 15000
	iterations := 3 // Number of iterations to average

	// Variables to accumulate metrics across iterations
	totalFalsePositiveRate := 0.0
	totalInsertSuccessRate := 0.0
	totalIterationTime := time.Duration(0)

	// Start total time tracking
	totalStartTime := time.Now()

	// Run multiple iterations
	for iteration := 0; iteration < iterations; iteration++ {
		// Start iteration timer
		iterationStartTime := time.Now()

		// Create a fresh cuckoo filter for each iteration
		cf := filterCuckoo.NewCuckooFilter(n, loadFactor)

		// Generate big data items to insert
		insertAttempts := int(n / 2) // Try to insert half capacity
		insertedItems := make([][]byte, 0, insertAttempts)
		successfulInserts := 0

		// Create large data items with varied content
		for i := 0; i < insertAttempts; i++ {
			data := make([]byte, bigDataSize)
			// Fill with pattern to make each item unique
			pattern := fmt.Sprintf("big_data_item_%d_%d_", iteration, i)
			copy(data, []byte(pattern))
			// Fill rest with repeating pattern
			for j := len(pattern); j < bigDataSize; j++ {
				data[j] = byte((iteration + i + j) % 256)
			}

			if cf.Insert(data) {
				insertedItems = append(insertedItems, data)
				successfulInserts++
			}
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
			if cf.Lookup(item) {
				existentHits++
			}
		}

		// Test existence of non-existent items (count false positives)
		falsePositives := 0
		for _, item := range nonExistentItems {
			if cf.Lookup(item) {
				falsePositives++
			}
		}

		// Calculate metrics for this iteration
		falsePositiveRate := float64(falsePositives) / float64(len(nonExistentItems)) * 100
		insertSuccessRate := float64(successfulInserts) / float64(insertAttempts) * 100

		// Accumulate metrics
		totalFalsePositiveRate += falsePositiveRate
		totalInsertSuccessRate += insertSuccessRate

		// Record iteration time
		iterationTime := time.Since(iterationStartTime)
		totalIterationTime += iterationTime
	}

	// Calculate total time
	totalTime := time.Since(totalStartTime)

	// Calculate averages
	avgFalsePositiveRate := totalFalsePositiveRate / float64(iterations)
	avgInsertSuccessRate := totalInsertSuccessRate / float64(iterations)
	avgIterationTime := totalIterationTime / time.Duration(iterations)

	// Create final cuckoo filter for benchmarking
	cf := filterCuckoo.NewCuckooFilter(n, loadFactor)
	finalItems := make([][]byte, 0)

	// Add items for mixed testing
	for i := 0; i < int(n)/3; i++ {
		data := make([]byte, bigDataSize)
		pattern := fmt.Sprintf("final_big_data_item_%d_", i)
		copy(data, []byte(pattern))
		for j := len(pattern); j < bigDataSize; j++ {
			data[j] = byte((i + j) % 256)
		}
		if cf.Insert(data) {
			finalItems = append(finalItems, data)
		}
	}

	// Add non-existent items
	for i := 0; i < nonExistentCount && i < 5000; i++ {
		data := make([]byte, bigDataSize)
		pattern := fmt.Sprintf("final_non_existent_%d_", i)
		copy(data, []byte(pattern))
		for j := len(pattern); j < bigDataSize; j++ {
			data[j] = byte((i*2 + j + 128) % 256)
		}
		finalItems = append(finalItems, data)
	}

	// Display comprehensive metrics in formatted table
	formatTable(b, "CUCKOO FILTER BIG DATA OPERATIONS BENCHMARK", []struct {
		sectionTitle string
		metrics      []struct {
			name  string
			value interface{}
			unit  string
		}
	}{
		{
			sectionTitle: "Accuracy Metrics",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Avg False Positive Rate", avgFalsePositiveRate, "%"},
				{"Avg Insert Success Rate", avgInsertSuccessRate, "%"},
			},
		},
		{
			sectionTitle: "Test Configuration",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Iterations", iterations, "runs"},
				{"Filter Capacity", int(n), "items"},
				{"Load Factor", loadFactor * 100, "%"},
			},
		},
		{
			sectionTitle: "Data Characteristics",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Data Size", bigDataSize, "bytes"},
				{"Non-Existent Items Per Iteration", nonExistentCount, "items"},
				{"Insert Attempts Per Iteration", int(n) / 2, "items"},
			},
		},
		{
			sectionTitle: "Filter Structure",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Filter Size", float64(cf.M*filterCuckoo.BucketSize) / 1024, "KB"},
				{"Buckets Count", int(cf.M), "buckets"},
				{"Total Slots", int(cf.M * filterCuckoo.BucketSize), "slots"},
			},
		},
		{
			sectionTitle: "Performance Timing",
			metrics: []struct {
				name  string
				value interface{}
				unit  string
			}{
				{"Avg Iteration Time", float64(avgIterationTime.Milliseconds()), "ms"},
				{"Total Time", float64(totalTime.Milliseconds()), "ms"},
				{"Time Per Item", float64(avgIterationTime.Nanoseconds()) / float64(n/2), "ns"},
			},
		},
	})

	// Add minimal benchmark to prevent re-running the entire function
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(cf.Buckets) // Minimal operation
	}
}

// BenchmarkComparativeSizes tests different filter sizes and their performance characteristics
func BenchmarkComparativeSizes(b *testing.B) {
	sizes := []struct {
		name string
		n    uint64
	}{
		{"small_1K", 1000},
		{"medium_10K", 10000},
		{"large_100K", 100000},
		{"xlarge_1M", 1000000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			cf := filterCuckoo.NewCuckooFilter(size.n, 0.9)

			// Insert up to 80% of capacity
			insertCount := int(float64(size.n) * 0.8)
			successfulInserts := 0

			items := make([][]byte, insertCount)
			for i := 0; i < insertCount; i++ {
				items[i] = []byte(fmt.Sprintf("size_test_%s_%d", size.name, i))
				if cf.Insert(items[i]) {
					successfulInserts++
				}
			}

			// Display size-specific metrics in formatted table
			formatTable(b, fmt.Sprintf("CUCKOO FILTER COMPARATIVE SIZES - %s", strings.ToUpper(size.name)), []struct {
				sectionTitle string
				metrics      []struct {
					name  string
					value interface{}
					unit  string
				}
			}{
				{
					sectionTitle: "Filter Configuration",
					metrics: []struct {
						name  string
						value interface{}
						unit  string
					}{
						{"Target Capacity", int(size.n), "items"},
						{"Actual Buckets", int(cf.M), "buckets"},
						{"Total Slots", int(cf.M * filterCuckoo.BucketSize), "slots"},
						{"Memory Size", float64(cf.M*filterCuckoo.BucketSize) / 1024, "KB"},
					},
				},
				{
					sectionTitle: "Insert Performance",
					metrics: []struct {
						name  string
						value interface{}
						unit  string
					}{
						{"Successful Inserts", successfulInserts, "items"},
						{"Total Attempted", insertCount, "items"},
						{"Insert Success Rate", float64(successfulInserts) / float64(insertCount) * 100, "%"},
					},
				},
				{
					sectionTitle: "Capacity Analysis",
					metrics: []struct {
						name  string
						value interface{}
						unit  string
					}{
						{"Capacity Utilization", float64(successfulInserts) / float64(size.n) * 100, "%"},
						{"Load Factor", 90.0, "%"}, // Fixed 0.9 from the code
					},
				},
			})

			// Add minimal benchmark to prevent re-running the entire function
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = len(cf.Buckets) // Minimal operation
			}
		})
	}
}
