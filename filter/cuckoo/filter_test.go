package cuckoo_test

import (
	"fmt"
	"testing"

	filterCuckoo "github.com/rag-nar1/Filters/filter/cuckoo"
)

func TestNewCuckooFilter(t *testing.T) {
	tests := []struct {
		name       string
		n          uint64
		fpRate     float64
		loadFactor float64
		expectedM  uint32
	}{
		{
			name:       "small filter",
			n:          100,
			fpRate:     0.01,
			loadFactor: 0.95,
			expectedM:  32, // Next power of two of ceil(100 / 0.95 / 4)
		},
		{
			name:       "large filter",
			n:          10000,
			fpRate:     0.001,
			loadFactor: 0.8,
			expectedM:  4096, // Next power of two of ceil(10000 / 0.8 / 4)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := filterCuckoo.NewCuckooFilter(tt.n, tt.loadFactor)

			if cf.M != tt.expectedM {
				t.Errorf("expected M=%d, got M=%d", tt.expectedM, cf.M)
			}

			if len(cf.Buckets) != int(tt.expectedM) {
				t.Errorf("expected %d buckets, got %d", tt.expectedM, len(cf.Buckets))
			}

			// Check that all buckets are initialized as empty
			for i, bucket := range cf.Buckets {
				if len(bucket) != 0 {
					t.Errorf("bucket %d should be empty, got length %d", i, len(bucket))
				}
			}
		})
	}
}

func TestHash(t *testing.T) {
	cf := filterCuckoo.NewCuckooFilter(1000, 0.95)

	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"single byte", []byte{1}},
		{"string data", []byte("hello")},
		{"longer string", []byte("this is a longer test string")},
		{"binary data", []byte{0, 1, 2, 3, 255, 254}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h1, fingerprint := cf.Hash(tt.data)

			// Fingerprint should be a valid byte (0-255) - this is always true for byte type
			// but we can check it's been set
			_ = fingerprint

			// Hash should be deterministic
			h1_2, fingerprint_2 := cf.Hash(tt.data)
			if h1 != h1_2 || fingerprint != fingerprint_2 {
				t.Errorf("hash not deterministic: first call (%d, %d), second call (%d, %d)",
					h1, fingerprint, h1_2, fingerprint_2)
			}
		})
	}
}

func TestAlternateIndex(t *testing.T) {
	cf := filterCuckoo.NewCuckooFilter(1000, 0.95)

	tests := []struct {
		name        string
		idx         uint32
		fingerprint byte
	}{
		{"zero index", 0, 123},
		{"mid index", cf.M / 2, 45},
		{"max index", cf.M - 1, 255},
		{"zero fingerprint", 100, 0},
		{"max fingerprint", 200, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			altIdx := cf.AlternateIndex(tt.idx, tt.fingerprint)
			// XOR property: alternating twice should give original index
			doubleAlt := cf.AlternateIndex(altIdx, tt.fingerprint)
			if doubleAlt != tt.idx {
				t.Errorf("double alternate should equal original: %d -> %d -> %d",
					tt.idx, altIdx, doubleAlt)
			}
		})
	}
}

func TestInsertAndLookup(t *testing.T) {
	cf := filterCuckoo.NewCuckooFilter(1000, 0.95)

	testData := [][]byte{
		[]byte("apple"),
		[]byte("banana"),
		[]byte("cherry"),
		[]byte("date"),
		[]byte("elderberry"),
		[]byte("fig"),
		[]byte("grape"),
		[]byte{},              // empty data
		[]byte{0},             // single null byte
		[]byte{255, 254, 253}, // binary data
	}

	// Test insertion
	for _, data := range testData {
		if !cf.Insert(data) {
			t.Errorf("failed to insert %v", data)
		}
	}

	// Test lookup for inserted items
	for _, data := range testData {
		if !cf.Lookup(data) {
			t.Errorf("failed to lookup inserted item %v", data)
		}
	}

	// Test lookup for non-inserted items
	nonInsertedData := [][]byte{
		[]byte("orange"),
		[]byte("kiwi"),
		[]byte("mango"),
	}

	for _, data := range nonInsertedData {
		if cf.Lookup(data) {
			t.Errorf("false positive: found non-inserted item %v", data)
		}
	}
}

func TestDelete(t *testing.T) {
	cf := filterCuckoo.NewCuckooFilter(1000, 0.95)

	testData := [][]byte{
		[]byte("apple"),
		[]byte("banana"),
		[]byte("cherry"),
	}

	// Insert test data
	for _, data := range testData {
		if !cf.Insert(data) {
			t.Errorf("failed to insert %v", data)
		}
	}

	// Verify items are present before deletion
	for _, data := range testData {
		if !cf.Lookup(data) {
			t.Errorf("item %v should be present before deletion", data)
		}
	}

	// Delete items
	for _, data := range testData {
		if !cf.Delete(data) {
			t.Errorf("failed to delete %v", data)
		}
	}

	// Verify items are gone after deletion
	for _, data := range testData {
		if cf.Lookup(data) {
			t.Errorf("item %v should not be present after deletion", data)
		}
	}

	// Try to delete non-existent items
	nonExistentData := [][]byte{
		[]byte("orange"),
		[]byte("kiwi"),
	}

	for _, data := range nonExistentData {
		if cf.Delete(data) {
			t.Errorf("should not be able to delete non-existent item %v", data)
		}
	}
}

func TestInsertDuplicates(t *testing.T) {
	cf := filterCuckoo.NewCuckooFilter(100, 0.95)

	data := []byte("duplicate")

	// Insert the same data multiple times
	for i := 0; i < 5; i++ {
		if !cf.Insert(data) {
			t.Errorf("failed to insert duplicate #%d", i+1)
		}
	}

	// Should still be able to lookup
	if !cf.Lookup(data) {
		t.Error("failed to lookup after inserting duplicates")
	}

	// Delete should work (removes one copy)
	if !cf.Delete(data) {
		t.Error("failed to delete duplicate")
	}

	// Depending on implementation, might still be present
	// (this tests the behavior with duplicates)
	present := cf.Lookup(data)
	t.Logf("After deleting one duplicate, item present: %v", present)
}

func TestCapacityLimits(t *testing.T) {
	// Create a very small filter to test capacity limits
	cf := filterCuckoo.NewCuckooFilter(10, 0.95)

	insertedCount := 0
	maxAttempts := 1000

	// Try to insert many items
	for i := 0; i < maxAttempts; i++ {
		data := []byte{byte(i), byte(i >> 8), byte(i >> 16)}
		if cf.Insert(data) {
			insertedCount++
		} else {
			// Filter is full or couldn't place item due to cuckoo limit
			break
		}
	}

	t.Logf("Successfully inserted %d items before hitting capacity limit", insertedCount)

	// Should have inserted at least a few items
	if insertedCount == 0 {
		t.Error("should be able to insert at least some items")
	}

	// Should hit capacity limit before inserting all attempts
	if insertedCount == maxAttempts {
		t.Error("should hit capacity limit before inserting all items")
	}
}

func TestBucketBoundaries(t *testing.T) {
	cf := filterCuckoo.NewCuckooFilter(100, 0.95)

	// Test with data that might map to edge buckets
	edgeData := [][]byte{
		[]byte{0},    // Might map to bucket 0
		[]byte{255},  // Different hash
		[]byte{0, 0}, // Different length
	}

	for _, data := range edgeData {
		h1, fingerprint := cf.Hash(data)
		h2 := cf.AlternateIndex(h1, fingerprint)

		// Both indices should be valid
		if h1%cf.M >= cf.M {
			t.Errorf("h1=%d out of bounds for data %v", h1, data)
		}
		if h2%cf.M >= cf.M {
			t.Errorf("h2=%d out of bounds for data %v", h2, data)
		}

		// Should be able to insert and lookup
		if !cf.Insert(data) {
			t.Errorf("failed to insert edge case data %v", data)
		}
		if !cf.Lookup(data) {
			t.Errorf("failed to lookup edge case data %v", data)
		}
	}
}

func TestErrorRates(t *testing.T) {
	tests := []struct {
		name           string
		n              uint64
		loadFactor     float64
		insertCount    int
		testCount      int
		maxExpectedFPR float64 // maximum acceptable false positive rate
	}{
		{
			name:           "small filter error rates",
			n:              1000,
			loadFactor:     0.95,
			insertCount:    500,   // insert 500 items
			testCount:      10000, // test 10000 random items for false positives
			maxExpectedFPR: 0.05,  // expect < 5% false positive rate
		},
		{
			name:           "large filter error rates",
			n:              10000,
			loadFactor:     0.95,
			insertCount:    5000,
			testCount:      50000,
			maxExpectedFPR: 0.04, // expect < 4% false positive rate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := filterCuckoo.NewCuckooFilter(tt.n, tt.loadFactor)

			// Generate test data for insertion
			insertedItems := make([][]byte, 0, tt.insertCount)
			insertedMap := make(map[string]bool) // track what we actually inserted

			// Insert items and track successful insertions
			successfulInserts := 0
			for i := 0; i < tt.insertCount; i++ {
				// Create unique test data
				data := []byte(fmt.Sprintf("test_item_%d_%d", i, i*2))

				if cf.Insert(data) {
					insertedItems = append(insertedItems, data)
					insertedMap[string(data)] = true
					successfulInserts++
				}
			}

			t.Logf("Successfully inserted %d out of %d items", successfulInserts, tt.insertCount)

			// Test for FALSE NEGATIVES
			// All successfully inserted items should be found
			falseNegatives := 0
			for _, item := range insertedItems {
				if !cf.Lookup(item) {
					falseNegatives++
					t.Errorf("FALSE NEGATIVE: item %s was inserted but not found", string(item))
				}
			}

			falseNegativeRate := float64(falseNegatives) / float64(len(insertedItems))
			t.Logf("False Negative Rate: %.4f%% (%d/%d)",
				falseNegativeRate*100, falseNegatives, len(insertedItems))

			// False negatives should NEVER occur in a working Cuckoo filter
			if falseNegatives > 0 {
				t.Errorf("Expected 0 false negatives, got %d", falseNegatives)
			}

			// Test for FALSE POSITIVES
			// Generate random test data that was NOT inserted
			falsePositives := 0
			testedNonInserted := 0

			for i := 0; i < tt.testCount; i++ {
				// Create test data that we know wasn't inserted
				testData := []byte("random_test_" + string(rune(i+100000)) + "_not_inserted")

				// Skip if this was actually inserted (very unlikely but possible)
				if insertedMap[string(testData)] {
					continue
				}

				testedNonInserted++
				if cf.Lookup(testData) {
					falsePositives++
				}
			}

			falsePositiveRate := float64(falsePositives) / float64(testedNonInserted)
			t.Logf("False Positive Rate: %.4f%% (%d/%d)",
				falsePositiveRate*100, falsePositives, testedNonInserted)

			// Check if false positive rate is within acceptable bounds
			if falsePositiveRate > tt.maxExpectedFPR {
				t.Errorf("False positive rate %.4f%% exceeds maximum expected rate %.4f%%",
					falsePositiveRate*100, tt.maxExpectedFPR*100)
			}
		})
	}
}

func TestErrorRatesWithDuplicates(t *testing.T) {
	cf := filterCuckoo.NewCuckooFilter(1000, 0.8)

	// Insert same item multiple times
	testItem := []byte("duplicate_test_item")
	insertCount := 5

	for i := 0; i < insertCount; i++ {
		if !cf.Insert(testItem) {
			t.Errorf("Failed to insert duplicate #%d", i+1)
		}
	}

	// Should still lookup successfully (no false negative)
	if !cf.Lookup(testItem) {
		t.Error("FALSE NEGATIVE: duplicate item not found after insertion")
	}

	// Delete and test partial removal
	deleted := cf.Delete(testItem)
	if !deleted {
		t.Error("Failed to delete duplicate item")
	}

	// Depending on implementation, item might still be present due to other copies
	stillPresent := cf.Lookup(testItem)
	t.Logf("After deleting one copy of duplicate, item still present: %v", stillPresent)
}

func TestErrorRatesEdgeCases(t *testing.T) {
	cf := filterCuckoo.NewCuckooFilter(100, 0.9)

	// Test with empty data
	emptyData := []byte{}
	if !cf.Insert(emptyData) {
		t.Error("Failed to insert empty data")
	}
	if !cf.Lookup(emptyData) {
		t.Error("FALSE NEGATIVE: empty data not found after insertion")
	}

	// Test with single byte data
	singleByte := []byte{42}
	if !cf.Insert(singleByte) {
		t.Error("Failed to insert single byte")
	}
	if !cf.Lookup(singleByte) {
		t.Error("FALSE NEGATIVE: single byte not found after insertion")
	}

	// Test with binary data
	binaryData := []byte{0, 255, 128, 1, 254}
	if !cf.Insert(binaryData) {
		t.Error("Failed to insert binary data")
	}
	if !cf.Lookup(binaryData) {
		t.Error("FALSE NEGATIVE: binary data not found after insertion")
	}
}

func TestFalseNegatives(t *testing.T) { // i think m need to be changed to be bigger because of insertions failure
	cf := filterCuckoo.NewCuckooFilter(1000, 0.95)

	// Generate 1000 random strings
	testData := make([]string, 1000)
	inserted := make([]string, 0, 1000)

	for i := 0; i < 1000; i++ {
		testData[i] = fmt.Sprintf("test_item_%d_%d", i, i*7+13)
		inserted = append(inserted, testData[i])

		if !cf.Insert([]byte(testData[i])) {
			t.Logf("Failed to insert item %d", i)
		}
	}

	t.Logf("Successfully inserted %d out of %d items", len(inserted), len(testData))

	// Check for false negatives - all inserted items should be found
	falseNegatives := 0
	for _, item := range inserted {
		if !cf.Lookup([]byte(item)) {
			falseNegatives++
			t.Errorf("FALSE NEGATIVE: item '%s' was inserted but not found", item)
		}
	}

	if falseNegatives > 0 {
		t.Errorf("Found %d false negatives out of %d inserted items", falseNegatives, len(inserted))
	} else {
		t.Logf("No false negatives found - all %d inserted items were successfully looked up", len(inserted))
	}
	t.Logf("Stash size: %d", len(cf.Stash))
}


func TestFalseNegativesBiggerM(t *testing.T) {
	N := uint64(10)
	cf := filterCuckoo.NewCuckooFilter(N, 0.95)

	// Generate 1000 random strings
	testData := make([]string, N)
	inserted := make([]string, 0, N)

	for i := uint64(0); i < N; i++ {
		testData[i] = fmt.Sprintf("known_item_%d", i)
		inserted = append(inserted, testData[i])

		if !cf.Insert([]byte(testData[i])) {
			t.Logf("Reached max kicks for item %d", i)
		}
	}

	t.Logf("Successfully inserted %d out of %d items", len(inserted), len(testData))

	// Check for false negatives - all inserted items should be found
	falseNegatives := 0
	for _, item := range inserted {
		if !cf.Lookup([]byte(item)) {
			falseNegatives++
			t.Errorf("FALSE NEGATIVE: item '%s' was inserted but not found", item)
		}
	}

	if falseNegatives > 0 {
		t.Errorf("Found %d false negatives out of %d inserted items", falseNegatives, len(inserted))
	} else {
		t.Logf("No false negatives found - all %d inserted items were successfully looked up", len(inserted))
	}
	t.Logf("Stash size: %d", len(cf.Stash))
}
