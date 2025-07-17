package filter_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	filter "github.com/rag-nar1/Filters/filter"
	filterBloom "github.com/rag-nar1/Filters/filter/bloom"
	filterCuckoo "github.com/rag-nar1/Filters/filter/cuckoo"
)

// Data structures to hold test results
type TestConfig struct {
	N                      uint64
	FpRate                 float64
	LoadFactor             float64
	InsertCount            int
	FalsePositiveTestCount int
	DeletionTestCount      int
}

type InsertResults struct {
	BloomInsertCount      int
	CuckooInsertCount     int
	BloomSuccessRate      float64
	CuckooSuccessRate     float64
	BloomTotalInsertTime  time.Duration
	CuckooTotalInsertTime time.Duration
	BloomAvgInsertTime    time.Duration
	CuckooAvgInsertTime   time.Duration
	BloomInsertOpsPerSec  float64
	CuckooInsertOpsPerSec float64
}

type AccuracyResults struct {
	BloomFalseNegatives   int
	CuckooFalseNegatives  int
	BloomFalsePositives   int
	CuckooFalsePositives  int
	BloomFPR              float64
	CuckooFPR             float64
	BloomFNStatus         string
	CuckooFNStatus        string
	BloomTotalLookupTime  time.Duration
	CuckooTotalLookupTime time.Duration
	BloomAvgLookupTime    time.Duration
	CuckooAvgLookupTime   time.Duration
	BloomLookupOpsPerSec  float64
	CuckooLookupOpsPerSec float64
	TotalLookupOperations int
}

type DeletionResults struct {
	DeletedCount    int
	NotFoundCount   int
	DeletionSuccess float64
	TotalDeleteTime time.Duration
	AvgDeleteTime   time.Duration
	DeleteOpsPerSec float64
}

type MemoryResults struct {
	BloomMemoryMB      float64
	CuckooMemoryMB     float64
	BloomBitArraySize  uint64
	BloomHashFunctions int
	CuckooBuckets      uint32
	CuckooStashItems   int
}

type FilterFeatures struct {
	BloomSupportsDeletion  bool
	CuckooSupportsDeletion bool
	BloomExactMembership   bool
	CuckooExactMembership  bool
	BloomNoFalseNegatives  bool
	CuckooNoFalseNegatives bool
	BloomConfigurableFPR   bool
	CuckooConfigurableFPR  bool
	BloomHardCapacity      bool
	CuckooHardCapacity     bool
}

// Logic functions - pure data processing
func runInsertTest(bf *filterBloom.BloomFilter, cf *filterCuckoo.CuckooFilter, testItems [][]byte) InsertResults {
	bloomInsertCount := 0
	cuckooInsertCount := 0

	// Measure Bloom filter insert time
	bloomStart := time.Now()
	for _, item := range testItems {
		bf.Insert(item)
		bloomInsertCount++
	}
	bloomTotalTime := time.Since(bloomStart)

	// Measure Cuckoo filter insert time
	cuckooStart := time.Now()
	for _, item := range testItems {
		if cf.Insert(item) {
			cuckooInsertCount++
		}
	}
	cuckooTotalTime := time.Since(cuckooStart)

	// Calculate averages and ops per second
	bloomAvgTime := time.Duration(0)
	bloomOpsPerSec := 0.0
	if bloomInsertCount > 0 {
		bloomAvgTime = bloomTotalTime / time.Duration(bloomInsertCount)
		bloomOpsPerSec = float64(bloomInsertCount) / bloomTotalTime.Seconds()
	}

	cuckooAvgTime := time.Duration(0)
	cuckooOpsPerSec := 0.0
	if cuckooInsertCount > 0 {
		cuckooAvgTime = cuckooTotalTime / time.Duration(cuckooInsertCount)
		cuckooOpsPerSec = float64(cuckooInsertCount) / cuckooTotalTime.Seconds()
	}

	return InsertResults{
		BloomInsertCount:      bloomInsertCount,
		CuckooInsertCount:     cuckooInsertCount,
		BloomSuccessRate:      100.0,
		CuckooSuccessRate:     float64(cuckooInsertCount) / float64(len(testItems)) * 100,
		BloomTotalInsertTime:  bloomTotalTime,
		CuckooTotalInsertTime: cuckooTotalTime,
		BloomAvgInsertTime:    bloomAvgTime,
		CuckooAvgInsertTime:   cuckooAvgTime,
		BloomInsertOpsPerSec:  bloomOpsPerSec,
		CuckooInsertOpsPerSec: cuckooOpsPerSec,
	}
}

func runAccuracyTest(bf *filterBloom.BloomFilter, cf *filterCuckoo.CuckooFilter,
	insertedItems [][]byte, nonInsertedItems [][]byte, cuckooInsertCount int) AccuracyResults {

	// False negative test with timing
	bloomFalseNegatives := 0
	cuckooFalseNegatives := 0

	bloomLookupOps := 0
	cuckooLookupOps := 0

	// Measure Bloom filter lookup time for false negatives
	bloomStart := time.Now()
	for _, item := range insertedItems {
		if !bf.Exist(item) {
			bloomFalseNegatives++
		}
		bloomLookupOps++
	}
	bloomFNTime := time.Since(bloomStart)

	// Measure Cuckoo filter lookup time for false negatives
	cuckooStart := time.Now()
	for i, item := range insertedItems {
		if i < cuckooInsertCount && !cf.Lookup(item) {
			cuckooFalseNegatives++
		}
		if i < cuckooInsertCount {
			cuckooLookupOps++
		}
	}
	cuckooFNTime := time.Since(cuckooStart)

	// False positive test with timing
	bloomFalsePositives := 0
	cuckooFalsePositives := 0

	// Measure Bloom filter lookup time for false positives
	bloomFPStart := time.Now()
	for _, item := range nonInsertedItems {
		if bf.Exist(item) {
			bloomFalsePositives++
		}
		bloomLookupOps++
	}
	bloomFPTime := time.Since(bloomFPStart)

	// Measure Cuckoo filter lookup time for false positives
	cuckooFPStart := time.Now()
	for _, item := range nonInsertedItems {
		if cf.Lookup(item) {
			cuckooFalsePositives++
		}
		cuckooLookupOps++
	}
	cuckooFPTime := time.Since(cuckooFPStart)

	// Calculate total lookup times and averages
	bloomTotalLookupTime := bloomFNTime + bloomFPTime
	cuckooTotalLookupTime := cuckooFNTime + cuckooFPTime

	bloomAvgLookupTime := time.Duration(0)
	bloomLookupOpsPerSec := 0.0
	if bloomLookupOps > 0 {
		bloomAvgLookupTime = bloomTotalLookupTime / time.Duration(bloomLookupOps)
		bloomLookupOpsPerSec = float64(bloomLookupOps) / bloomTotalLookupTime.Seconds()
	}

	cuckooAvgLookupTime := time.Duration(0)
	cuckooLookupOpsPerSec := 0.0
	if cuckooLookupOps > 0 {
		cuckooAvgLookupTime = cuckooTotalLookupTime / time.Duration(cuckooLookupOps)
		cuckooLookupOpsPerSec = float64(cuckooLookupOps) / cuckooTotalLookupTime.Seconds()
	}

	bloomFPR := float64(bloomFalsePositives) / float64(len(nonInsertedItems)) * 100
	cuckooFPR := float64(cuckooFalsePositives) / float64(len(nonInsertedItems)) * 100

	bloomFNStatus := "✅ PASS"
	cuckooFNStatus := "✅ PASS"
	if bloomFalseNegatives > 0 {
		bloomFNStatus = "❌ FAIL"
	}
	if cuckooFalseNegatives > 0 {
		cuckooFNStatus = "❌ FAIL"
	}

	return AccuracyResults{
		BloomFalseNegatives:   bloomFalseNegatives,
		CuckooFalseNegatives:  cuckooFalseNegatives,
		BloomFalsePositives:   bloomFalsePositives,
		CuckooFalsePositives:  cuckooFalsePositives,
		BloomFPR:              bloomFPR,
		CuckooFPR:             cuckooFPR,
		BloomFNStatus:         bloomFNStatus,
		CuckooFNStatus:        cuckooFNStatus,
		BloomTotalLookupTime:  bloomTotalLookupTime,
		CuckooTotalLookupTime: cuckooTotalLookupTime,
		BloomAvgLookupTime:    bloomAvgLookupTime,
		CuckooAvgLookupTime:   cuckooAvgLookupTime,
		BloomLookupOpsPerSec:  bloomLookupOpsPerSec,
		CuckooLookupOpsPerSec: cuckooLookupOpsPerSec,
		TotalLookupOperations: bloomLookupOps + cuckooLookupOps,
	}
}

func runDeletionTest(cf *filterCuckoo.CuckooFilter, deleteItems [][]byte) DeletionResults {
	deletedCount := 0

	// Measure deletion time
	deleteStart := time.Now()
	for _, item := range deleteItems {
		if cf.Delete(item) {
			deletedCount++
		}
	}
	totalDeleteTime := time.Since(deleteStart)

	// Verify deleted items are no longer found
	notFoundCount := 0
	for _, item := range deleteItems {
		if !cf.Lookup(item) {
			notFoundCount++
		}
	}

	deletionSuccess := 0.0
	if deletedCount > 0 {
		deletionSuccess = float64(notFoundCount) / float64(deletedCount) * 100
	}

	avgDeleteTime := time.Duration(0)
	deleteOpsPerSec := 0.0
	if deletedCount > 0 {
		avgDeleteTime = totalDeleteTime / time.Duration(deletedCount)
		deleteOpsPerSec = float64(deletedCount) / totalDeleteTime.Seconds()
	}

	return DeletionResults{
		DeletedCount:    deletedCount,
		NotFoundCount:   notFoundCount,
		DeletionSuccess: deletionSuccess,
		TotalDeleteTime: totalDeleteTime,
		AvgDeleteTime:   avgDeleteTime,
		DeleteOpsPerSec: deleteOpsPerSec,
	}
}

func analyzeMemoryAndStructure(bf *filterBloom.BloomFilter, cf *filterCuckoo.CuckooFilter) MemoryResults {
	bloomMemoryBits := bf.M
	bloomMemoryMB := float64(bloomMemoryBits) / 8 / 1024 / 1024 // Convert to MB
	cuckooMemoryMB := float64(cf.M*4) / 1024 / 1024             // 4 bytes per bucket, convert to MB

	return MemoryResults{
		BloomMemoryMB:      bloomMemoryMB,
		CuckooMemoryMB:     cuckooMemoryMB,
		BloomBitArraySize:  bf.M,
		BloomHashFunctions: bf.K,
		CuckooBuckets:      cf.M,
		CuckooStashItems:   len(cf.Stash),
	}
}

func getFilterFeatures() FilterFeatures {
	return FilterFeatures{
		BloomSupportsDeletion:  false,
		CuckooSupportsDeletion: true,
		BloomExactMembership:   false,
		CuckooExactMembership:  true,
		BloomNoFalseNegatives:  true,
		CuckooNoFalseNegatives: true,
		BloomConfigurableFPR:   true,
		CuckooConfigurableFPR:  false,
		BloomHardCapacity:      false,
		CuckooHardCapacity:     true,
	}
}

// Helper function to format duration for display
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0ns"
	}

	// Convert to appropriate unit
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000.0)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// Helper function to center text within a fixed width
func centerText(text string, width int) string {
	if len(text) >= width {
		if len(text) > width {
			return text[:width-3] + "..."
		}
		return text
	}

	padding := width - len(text)
	leftPad := padding / 2
	rightPad := padding - leftPad

	return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
}

// Helper function to create table headers with fixed widths
func createTableHeader(col1, col2, col3, col4 string, col1Width, col2Width, col3Width, col4Width int) string {
	return fmt.Sprintf("│  %s │ %s │ %s │ %s",
		centerText(col1, col1Width),
		centerText(col2, col2Width),
		centerText(col3, col3Width),
		centerText(col4, col4Width))
}

// Helper function to create table separator with fixed widths
func createTableSeparator(col1Width, col2Width, col3Width, col4Width int) string {
	return fmt.Sprintf("│  %s─┼─%s─┼─%s─┼─%s",
		strings.Repeat("─", col1Width),
		strings.Repeat("─", col2Width),
		strings.Repeat("─", col3Width),
		strings.Repeat("─", col4Width))
}

// Helper function to create table row with fixed widths and centered text
func createTableRow(col1, col2, col3, col4 string, col1Width, col2Width, col3Width, col4Width int) string {
	return fmt.Sprintf("│  %s │ %s │ %s │ %s",
		centerText(col1, col1Width),
		centerText(col2, col2Width),
		centerText(col3, col3Width),
		centerText(col4, col4Width))
}

// Printing functions - pure display logic
func printTestHeader(config TestConfig) {
	fmt.Printf("\n" + strings.Repeat("=", 100) + "\n")
	fmt.Printf("  🔬 BLOOM vs CUCKOO FILTER COMPARISON TEST (MILLIONS SCALE)\n")
	fmt.Printf(strings.Repeat("=", 100) + "\n")

	fmt.Printf("\n┌─ Test Configuration\n")
	fmt.Printf("│  Target Capacity    : %10d items (%.1fM)\n", config.N, float64(config.N)/1000000)
	fmt.Printf("│  Bloom Target FPR   : %10.2f%%\n", config.FpRate*100)
	fmt.Printf("│  Cuckoo Load Factor : %10.0f%%\n", config.LoadFactor*100)
}

func printInsertResults(config TestConfig, results InsertResults) {
	const col1Width, col2Width, col3Width, col4Width = 30, 22, 22, 12

	fmt.Printf("\n┌─ Phase 1: Insert Operations (Large Scale)\n")
	fmt.Printf("│  Preparing %d test items...\n", config.InsertCount)

	fmt.Printf("%s\n", createTableHeader("Metric", "Bloom Filter", "Cuckoo Filter", "Unit", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableSeparator(col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Items Attempted",
		fmt.Sprintf("%.1fM", float64(config.InsertCount)/1000000),
		fmt.Sprintf("%.1fM", float64(config.InsertCount)/1000000),
		"items", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Items Inserted",
		fmt.Sprintf("%.1fM", float64(results.BloomInsertCount)/1000000),
		fmt.Sprintf("%.1fM", float64(results.CuckooInsertCount)/1000000),
		"items", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Success Rate",
		fmt.Sprintf("%.1f", results.BloomSuccessRate),
		fmt.Sprintf("%.1f", results.CuckooSuccessRate),
		"%", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Total Insert Time",
		formatDuration(results.BloomTotalInsertTime),
		formatDuration(results.CuckooTotalInsertTime),
		"time", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Avg Insert Time",
		formatDuration(results.BloomAvgInsertTime),
		formatDuration(results.CuckooAvgInsertTime),
		"per op", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Insert Ops/Sec",
		fmt.Sprintf("%.1fK", results.BloomInsertOpsPerSec/1000),
		fmt.Sprintf("%.1fK", results.CuckooInsertOpsPerSec/1000),
		"K ops/s", col1Width, col2Width, col3Width, col4Width))
}

func printAccuracyResults(config TestConfig, results AccuracyResults) {
	const col1Width, col2Width, col3Width, col4Width = 30, 22, 22, 12

	fmt.Printf("\n┌─ Phase 2: False Negative Analysis (Million Scale)\n")
	fmt.Printf("│  Testing %d inserted items for false negatives...\n", config.InsertCount)

	fmt.Printf("%s\n", createTableHeader("Metric", "Bloom Filter", "Cuckoo Filter", "Unit", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableSeparator(col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Items Tested",
		fmt.Sprintf("%.1fM", float64(config.InsertCount)/1000000),
		fmt.Sprintf("%.1fM", float64(config.InsertCount)/1000000),
		"items", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("False Negatives",
		fmt.Sprintf("%d", results.BloomFalseNegatives),
		fmt.Sprintf("%d", results.CuckooFalseNegatives),
		"items", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Status",
		results.BloomFNStatus,
		results.CuckooFNStatus,
		"result", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("\n┌─ Phase 3: False Positive Analysis (Million Scale)\n")
	fmt.Printf("│  Generating %d non-inserted items for false positive testing...\n", config.FalsePositiveTestCount)
	fmt.Printf("│  Testing false positives on %d items...\n", config.FalsePositiveTestCount)

	fmt.Printf("%s\n", createTableHeader("Metric", "Bloom Filter", "Cuckoo Filter", "Unit", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableSeparator(col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Items Tested",
		fmt.Sprintf("%.1fM", float64(config.FalsePositiveTestCount)/1000000),
		fmt.Sprintf("%.1fM", float64(config.FalsePositiveTestCount)/1000000),
		"items", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("False Positives",
		fmt.Sprintf("%d", results.BloomFalsePositives),
		fmt.Sprintf("%d", results.CuckooFalsePositives),
		"items", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("False Positive Rate",
		fmt.Sprintf("%.2f", results.BloomFPR),
		fmt.Sprintf("%.2f", results.CuckooFPR),
		"%", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Target FP Rate",
		fmt.Sprintf("%.2f", config.FpRate*100),
		"Dynamic",
		"%", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Total Lookup Time",
		formatDuration(results.BloomTotalLookupTime),
		formatDuration(results.CuckooTotalLookupTime),
		"time", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Avg Lookup Time",
		formatDuration(results.BloomAvgLookupTime),
		formatDuration(results.CuckooAvgLookupTime),
		"per op", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Lookup Ops/Sec",
		fmt.Sprintf("%.1fK", results.BloomLookupOpsPerSec/1000),
		fmt.Sprintf("%.1fK", results.CuckooLookupOpsPerSec/1000),
		"K ops/s", col1Width, col2Width, col3Width, col4Width))
}

func printDeletionResults(config TestConfig, results DeletionResults) {
	const col1Width, col2Width, col3Width, col4Width = 30, 22, 22, 12

	fmt.Printf("\n┌─ Phase 4: Deletion Capability Test (Large Scale)\n")
	fmt.Printf("│  Testing deletion of %d items...\n", config.DeletionTestCount)
	fmt.Printf("│  Verifying %d deletions...\n", config.DeletionTestCount)

	fmt.Printf("%s\n", createTableHeader("Metric", "Bloom Filter", "Cuckoo Filter", "Unit", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableSeparator(col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Supports Deletion",
		"✗ No",
		fmt.Sprintf("%d", config.DeletionTestCount),
		"items", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Items Deleted",
		"N/A",
		fmt.Sprintf("%d", results.DeletedCount),
		"items", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Items Not Found",
		"N/A",
		fmt.Sprintf("%d", results.NotFoundCount),
		"items", col1Width, col2Width, col3Width, col4Width))

	if results.DeletedCount > 0 {
		fmt.Printf("%s\n", createTableRow("Deletion Success",
			"N/A",
			fmt.Sprintf("%.1f", results.DeletionSuccess),
			"%", col1Width, col2Width, col3Width, col4Width))

		fmt.Printf("%s\n", createTableRow("Total Delete Time",
			"N/A",
			formatDuration(results.TotalDeleteTime),
			"time", col1Width, col2Width, col3Width, col4Width))

		fmt.Printf("%s\n", createTableRow("Avg Delete Time",
			"N/A",
			formatDuration(results.AvgDeleteTime),
			"per op", col1Width, col2Width, col3Width, col4Width))

		fmt.Printf("%s\n", createTableRow("Delete Ops/Sec",
			"N/A",
			fmt.Sprintf("%.1fK", results.DeleteOpsPerSec/1000),
			"K ops/s", col1Width, col2Width, col3Width, col4Width))
	}
}

func printMemoryResults(memory MemoryResults) {
	const col1Width, col2Width, col3Width, col4Width = 30, 22, 22, 12

	fmt.Printf("\n┌─ Phase 5: Memory & Structure Analysis (Million Scale)\n")

	fmt.Printf("%s\n", createTableHeader("Metric", "Bloom Filter", "Cuckoo Filter", "Unit", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableSeparator(col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Memory Usage",
		fmt.Sprintf("%.1f", memory.BloomMemoryMB),
		fmt.Sprintf("%.1f", memory.CuckooMemoryMB),
		"MB", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Bit Array Size",
		fmt.Sprintf("%.1fM", float64(memory.BloomBitArraySize)/1000000),
		"N/A",
		"bits", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Hash Functions",
		fmt.Sprintf("%d", memory.BloomHashFunctions),
		"2",
		"count", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Buckets",
		"N/A",
		fmt.Sprintf("%.1fM", float64(memory.CuckooBuckets)/1000000),
		"count", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("Stash Items",
		"N/A",
		fmt.Sprintf("%d", memory.CuckooStashItems),
		"items", col1Width, col2Width, col3Width, col4Width))
}

func printFeatureComparison(features FilterFeatures) {
	const col1Width, col2Width, col3Width, col4Width = 30, 22, 22, 12

	fmt.Printf("\n┌─ Phase 6: Feature Comparison\n")

	fmt.Printf("%s\n", createTableHeader("Feature", "Bloom Filter", "Cuckoo Filter", "Support", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableSeparator(col1Width, col2Width, col3Width, col4Width))

	bloomDeletion := "✗"
	if features.BloomSupportsDeletion {
		bloomDeletion = "✓"
	}
	cuckooDeletion := "✗"
	if features.CuckooSupportsDeletion {
		cuckooDeletion = "✓"
	}

	bloomExact := "✗"
	if features.BloomExactMembership {
		bloomExact = "✓"
	}
	cuckooExact := "✗"
	if features.CuckooExactMembership {
		cuckooExact = "✓"
	}

	bloomNoFN := "✗"
	if features.BloomNoFalseNegatives {
		bloomNoFN = "✓"
	}
	cuckooNoFN := "✗"
	if features.CuckooNoFalseNegatives {
		cuckooNoFN = "✓"
	}

	bloomConfigFPR := "✗"
	if features.BloomConfigurableFPR {
		bloomConfigFPR = "✓"
	}
	cuckooConfigFPR := "✗"
	if features.CuckooConfigurableFPR {
		cuckooConfigFPR = "✓"
	}

	bloomHardCap := "✗"
	if features.BloomHardCapacity {
		bloomHardCap = "✓"
	}
	cuckooHardCap := "✗"
	if features.CuckooHardCapacity {
		cuckooHardCap = "✓"
	}

	fmt.Printf("%s\n", createTableRow("Deletion", bloomDeletion, cuckooDeletion, "bool", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Exact Membership", bloomExact, cuckooExact, "bool", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("No False Negatives", bloomNoFN, cuckooNoFN, "bool", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Configurable FPR", bloomConfigFPR, cuckooConfigFPR, "bool", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Hard Capacity", bloomHardCap, cuckooHardCap, "bool", col1Width, col2Width, col3Width, col4Width))
}

func printTestSummary(config TestConfig, insert InsertResults, accuracy AccuracyResults,
	deletion DeletionResults, memory MemoryResults) {

	const col1Width, col2Width, col3Width, col4Width = 30, 22, 22, 12

	fmt.Printf("\n┌─ Test Summary (Million Scale Results)\n")

	fmt.Printf("%s\n", createTableHeader("Result", "Bloom Filter", "Cuckoo Filter", "Status", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableSeparator(col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("%s\n", createTableRow("False Negatives", accuracy.BloomFNStatus, accuracy.CuckooFNStatus, "test", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("FP Rate", fmt.Sprintf("%.2f", accuracy.BloomFPR), fmt.Sprintf("%.2f", accuracy.CuckooFPR), "%", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Deletions", "N/A", fmt.Sprintf("%d", deletion.DeletedCount), "items", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Memory", fmt.Sprintf("%.1f", memory.BloomMemoryMB), fmt.Sprintf("%.1f", memory.CuckooMemoryMB), "MB", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Items Processed", fmt.Sprintf("%.1fM", float64(config.InsertCount)/1000000), fmt.Sprintf("%.1fM", float64(insert.CuckooInsertCount)/1000000), "items", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Avg Insert Time", formatDuration(insert.BloomAvgInsertTime), formatDuration(insert.CuckooAvgInsertTime), "per op", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Avg Lookup Time", formatDuration(accuracy.BloomAvgLookupTime), formatDuration(accuracy.CuckooAvgLookupTime), "per op", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Insert Throughput", fmt.Sprintf("%.1fK", insert.BloomInsertOpsPerSec/1000), fmt.Sprintf("%.1fK", insert.CuckooInsertOpsPerSec/1000), "K ops/s", col1Width, col2Width, col3Width, col4Width))
	fmt.Printf("%s\n", createTableRow("Lookup Throughput", fmt.Sprintf("%.1fK", accuracy.BloomLookupOpsPerSec/1000), fmt.Sprintf("%.1fK", accuracy.CuckooLookupOpsPerSec/1000), "K ops/s", col1Width, col2Width, col3Width, col4Width))

	overallStatus := "🎉 ALL TESTS PASSED"
	if accuracy.BloomFalseNegatives > 0 || accuracy.CuckooFalseNegatives > 0 {
		overallStatus = "❌ SOME TESTS FAILED"
	}
	fmt.Printf("%s\n", createTableRow("Overall Status", overallStatus, "", "result", col1Width, col2Width, col3Width, col4Width))

	fmt.Printf("└" + strings.Repeat("─", col1Width+col2Width+col3Width+col4Width+12) + "\n\n")
}

func printProgress(phase string, current, total int) {
	if current > 0 && current%500000 == 0 {
		fmt.Printf("│  %s: %.1fM items processed...\n", phase, float64(current)/1000000)
	}
}

// Helper function to create formatted comparison tables and write to files for tests
func formatTestComparisonTable(t *testing.T, title string, sections []struct {
	sectionTitle string
	metrics      []struct {
		name        string
		bloomValue  interface{}
		cuckooValue interface{}
		unit        string
	}
}) {
	// Create the bench-results directory if it doesn't exist
	resultsDir := "/home/ragnar/Desktop/axiom/Bloom-Filter/filter/bench-results"
	err := os.MkdirAll(resultsDir, 0755)
	if err != nil {
		t.Logf("Error creating directory %s: %v", resultsDir, err)
		return
	}

	// Create filename based on test name
	testName := strings.ReplaceAll(t.Name(), "/", "_")
	filename := fmt.Sprintf("%s/test_results_%s.txt", resultsDir, testName)

	// Create or overwrite the file
	file, err := os.Create(filename)
	if err != nil {
		t.Logf("Error creating file %s: %v", filename, err)
		return
	}
	defer file.Close()

	// Write table to file
	fmt.Fprintf(file, "\n"+strings.Repeat("=", 90)+"\n")
	fmt.Fprintf(file, "  %s\n", title)
	fmt.Fprintf(file, strings.Repeat("=", 90)+"\n")

	for _, section := range sections {
		fmt.Fprintf(file, "\n┌─ %s\n", section.sectionTitle)
		fmt.Fprintf(file, "│  %-30s │ %15s │ %15s │ %10s\n", "Metric", "Bloom Filter", "Cuckoo Filter", "Unit")
		fmt.Fprintf(file, "│  "+strings.Repeat("─", 30)+"─┼─"+strings.Repeat("─", 15)+"─┼─"+strings.Repeat("─", 15)+"─┼─"+strings.Repeat("─", 10)+"\n")

		for _, metric := range section.metrics {
			var bloomStr, cuckooStr string

			switch v := metric.bloomValue.(type) {
			case float64:
				if strings.Contains(metric.unit, "%") {
					bloomStr = fmt.Sprintf("%.2f", v)
				} else if strings.Contains(metric.unit, "ms") || strings.Contains(metric.unit, "KB") || strings.Contains(metric.unit, "ns") {
					bloomStr = fmt.Sprintf("%.1f", v)
				} else {
					bloomStr = fmt.Sprintf("%.0f", v)
				}
			case int:
				bloomStr = fmt.Sprintf("%d", v)
			case bool:
				if v {
					bloomStr = "✓"
				} else {
					bloomStr = "✗"
				}
			default:
				bloomStr = fmt.Sprintf("%v", v)
			}

			switch v := metric.cuckooValue.(type) {
			case float64:
				if strings.Contains(metric.unit, "%") {
					cuckooStr = fmt.Sprintf("%.2f", v)
				} else if strings.Contains(metric.unit, "ms") || strings.Contains(metric.unit, "KB") || strings.Contains(metric.unit, "ns") {
					cuckooStr = fmt.Sprintf("%.1f", v)
				} else {
					cuckooStr = fmt.Sprintf("%.0f", v)
				}
			case int:
				cuckooStr = fmt.Sprintf("%d", v)
			case bool:
				if v {
					cuckooStr = "✓"
				} else {
					cuckooStr = "✗"
				}
			default:
				cuckooStr = fmt.Sprintf("%v", v)
			}

			fmt.Fprintf(file, "│  %-30s │ %15s │ %15s │ %10s\n",
				metric.name, bloomStr, cuckooStr, metric.unit)
		}
	}
	fmt.Fprintf(file, "└"+strings.Repeat("─", 94)+"\n")

	// Log the filename so user knows where to find the results
	fmt.Printf("📝 Test comparison results saved to: %s\n", filename)
}

// TestBloomVsCuckooComparison - Main orchestrator function
func TestBloomVsCuckooComparison(t *testing.T) {
	// Test configuration
	config := TestConfig{
		N:                      5000000, // 5 million capacity
		FpRate:                 0.01,
		LoadFactor:             0.95,
		InsertCount:            5000000, // 5 million items
		FalsePositiveTestCount: 1000000, // 1 million items
		DeletionTestCount:      500000,  // 500K items
	}

	// Create filters
	bf := filterBloom.NewBloomFilter(config.N, config.FpRate, filter.DoubleHash)
	cf := filterCuckoo.NewCuckooFilter(config.N, config.LoadFactor)

	// Print test header
	printTestHeader(config)

	// Phase 1: Prepare test data and run insert test
	testItems := make([][]byte, config.InsertCount)
	for i := range testItems {
		testItems[i] = []byte(fmt.Sprintf("test_item_%d", i))
	}

	insertResults := runInsertTest(bf, cf, testItems)
	printInsertResults(config, insertResults)

	// Phase 2 & 3: Run accuracy tests
	nonInsertedItems := make([][]byte, config.FalsePositiveTestCount)
	for i := range nonInsertedItems {
		nonInsertedItems[i] = []byte(fmt.Sprintf("non_inserted_item_%d", i))
	}

	accuracyResults := runAccuracyTest(bf, cf, testItems, nonInsertedItems, insertResults.CuckooInsertCount)

	// Report errors to testing framework
	for i := 0; i < accuracyResults.BloomFalseNegatives && i < 10; i++ {
		t.Errorf("BLOOM FALSE NEGATIVE: item should exist but doesn't")
	}
	for i := 0; i < accuracyResults.CuckooFalseNegatives && i < 10; i++ {
		t.Errorf("CUCKOO FALSE NEGATIVE: item should exist but doesn't")
	}

	// Validate Bloom filter FP rate
	if accuracyResults.BloomFPR > config.FpRate*100*3 {
		t.Errorf("Bloom filter false positive rate %.2f%% exceeds expected ~%.2f%%",
			accuracyResults.BloomFPR, config.FpRate*100)
	}

	printAccuracyResults(config, accuracyResults)

	// Phase 4: Run deletion test
	deleteCount := min(config.DeletionTestCount, insertResults.CuckooInsertCount)
	deleteTestItems := testItems[:deleteCount]
	deletionResults := runDeletionTest(cf, deleteTestItems)
	printDeletionResults(config, deletionResults)

	// Phase 5: Analyze memory and structure
	memoryResults := analyzeMemoryAndStructure(bf, cf)
	printMemoryResults(memoryResults)

	// Phase 6: Feature comparison
	features := getFilterFeatures()
	printFeatureComparison(features)

	// Summary
	printTestSummary(config, insertResults, accuracyResults, deletionResults, memoryResults)

	// Generate file report
	formatTestComparisonTable(t, "BLOOM vs CUCKOO FILTER - MILLION SCALE COMPARISON TEST", []struct {
		sectionTitle string
		metrics      []struct {
			name        string
			bloomValue  interface{}
			cuckooValue interface{}
			unit        string
		}
	}{
		{
			sectionTitle: "Insert Performance (Million Scale)",
			metrics: []struct {
				name        string
				bloomValue  interface{}
				cuckooValue interface{}
				unit        string
			}{
				{"Items Attempted", config.InsertCount, config.InsertCount, "items"},
				{"Items Successfully Inserted", insertResults.BloomInsertCount, insertResults.CuckooInsertCount, "items"},
				{"Insert Success Rate", insertResults.BloomSuccessRate, insertResults.CuckooSuccessRate, "%"},
				{"Items in Millions", float64(config.InsertCount) / 1000000, float64(insertResults.CuckooInsertCount) / 1000000, "M items"},
			},
		},
		{
			sectionTitle: "Insert Timing Performance",
			metrics: []struct {
				name        string
				bloomValue  interface{}
				cuckooValue interface{}
				unit        string
			}{
				{"Total Insert Time", formatDuration(insertResults.BloomTotalInsertTime), formatDuration(insertResults.CuckooTotalInsertTime), "time"},
				{"Average Insert Time", formatDuration(insertResults.BloomAvgInsertTime), formatDuration(insertResults.CuckooAvgInsertTime), "per op"},
				{"Insert Operations/Sec", int(insertResults.BloomInsertOpsPerSec), int(insertResults.CuckooInsertOpsPerSec), "ops/s"},
				{"Insert Throughput", fmt.Sprintf("%.1fK", insertResults.BloomInsertOpsPerSec/1000), fmt.Sprintf("%.1fK", insertResults.CuckooInsertOpsPerSec/1000), "K ops/s"},
			},
		},
		{
			sectionTitle: "Lookup Timing Performance",
			metrics: []struct {
				name        string
				bloomValue  interface{}
				cuckooValue interface{}
				unit        string
			}{
				{"Total Lookup Time", formatDuration(accuracyResults.BloomTotalLookupTime), formatDuration(accuracyResults.CuckooTotalLookupTime), "time"},
				{"Average Lookup Time", formatDuration(accuracyResults.BloomAvgLookupTime), formatDuration(accuracyResults.CuckooAvgLookupTime), "per op"},
				{"Lookup Operations/Sec", int(accuracyResults.BloomLookupOpsPerSec), int(accuracyResults.CuckooLookupOpsPerSec), "ops/s"},
				{"Lookup Throughput", fmt.Sprintf("%.1fK", accuracyResults.BloomLookupOpsPerSec/1000), fmt.Sprintf("%.1fK", accuracyResults.CuckooLookupOpsPerSec/1000), "K ops/s"},
			},
		},
		{
			sectionTitle: "Accuracy Analysis (Million Scale)",
			metrics: []struct {
				name        string
				bloomValue  interface{}
				cuckooValue interface{}
				unit        string
			}{
				{"False Negatives Found", accuracyResults.BloomFalseNegatives, accuracyResults.CuckooFalseNegatives, "items"},
				{"False Positives Found", accuracyResults.BloomFalsePositives, accuracyResults.CuckooFalsePositives, "items"},
				{"False Positive Rate", accuracyResults.BloomFPR, accuracyResults.CuckooFPR, "%"},
				{"FP Test Items", config.FalsePositiveTestCount, config.FalsePositiveTestCount, "items"},
				{"Target FP Rate", config.FpRate * 100, "Dynamic", "%"},
			},
		},
		{
			sectionTitle: "Deletion Performance",
			metrics: []struct {
				name        string
				bloomValue  interface{}
				cuckooValue interface{}
				unit        string
			}{
				{"Supports Deletion", features.BloomSupportsDeletion, features.CuckooSupportsDeletion, "bool"},
				{"Items Deleted", "N/A", deletionResults.DeletedCount, "items"},
				{"Deletion Success Rate", "N/A", deletionResults.DeletionSuccess, "%"},
				{"Total Delete Time", "N/A", formatDuration(deletionResults.TotalDeleteTime), "time"},
				{"Average Delete Time", "N/A", formatDuration(deletionResults.AvgDeleteTime), "per op"},
				{"Delete Operations/Sec", "N/A", fmt.Sprintf("%.1fK", deletionResults.DeleteOpsPerSec/1000), "K ops/s"},
			},
		},
		{
			sectionTitle: "Filter Capabilities",
			metrics: []struct {
				name        string
				bloomValue  interface{}
				cuckooValue interface{}
				unit        string
			}{
				{"Supports Deletion", features.BloomSupportsDeletion, features.CuckooSupportsDeletion, "bool"},
				{"Exact Membership Test", features.BloomExactMembership, features.CuckooExactMembership, "bool"},
				{"Guaranteed No False Negatives", features.BloomNoFalseNegatives, features.CuckooNoFalseNegatives, "bool"},
				{"Configurable FP Rate", features.BloomConfigurableFPR, features.CuckooConfigurableFPR, "bool"},
				{"Hard Capacity Limit", features.BloomHardCapacity, features.CuckooHardCapacity, "bool"},
			},
		},
		{
			sectionTitle: "Memory & Structure (Million Scale)",
			metrics: []struct {
				name        string
				bloomValue  interface{}
				cuckooValue interface{}
				unit        string
			}{
				{"Memory Usage", memoryResults.BloomMemoryMB, memoryResults.CuckooMemoryMB, "MB"},
				{"Bit Array Size", int(memoryResults.BloomBitArraySize), "N/A", "bits"},
				{"Hash Functions Used", memoryResults.BloomHashFunctions, 2, "count"},
				{"Buckets", "N/A", int(memoryResults.CuckooBuckets), "count"},
				{"Stash Items", "N/A", memoryResults.CuckooStashItems, "items"},
			},
		},
		{
			sectionTitle: "Test Configuration (Million Scale)",
			metrics: []struct {
				name        string
				bloomValue  interface{}
				cuckooValue interface{}
				unit        string
			}{
				{"Target Capacity", int(config.N), int(config.N), "items"},
				{"Target Capacity (Millions)", float64(config.N) / 1000000, float64(config.N) / 1000000, "M items"},
				{"Load Factor", "N/A", config.LoadFactor * 100, "%"},
				{"Items Tested for FP", config.FalsePositiveTestCount, config.FalsePositiveTestCount, "items"},
				{"Items Tested for Deletion", "N/A", config.DeletionTestCount, "items"},
			},
		},
	})
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
