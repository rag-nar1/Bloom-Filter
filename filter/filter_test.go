package filter_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
	"github.com/rag-nar1/Bloom-Filter/filter"
)

func TestNewBloomFilter(t *testing.T) {
	tests := []struct {
		M    int
		K    int
		name string
	}{
		{100, 3, "test1"},
		{1000, 5, "test2"},
		{10000, 7, "test3"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bf := filter.NewBloomFilter(test.M, test.K)

			// checking any intilization problems
			if bf.M != test.M {
				t.Errorf("expected m=%d, got %d", test.M, bf.M)
			}
			if bf.K != test.K {
				t.Errorf("expected k=%d, got %d", test.K, bf.K)
			}
			if len(bf.Hashes) != test.K {
				t.Errorf("expected %d hash functions, got %d", test.K, len(bf.Hashes))
			}

			// check filter size
			expectedSizeForFilter := test.M/64 + 1
			if len(bf.Bits) != expectedSizeForFilter {
				t.Errorf("expected bit array size %d, got %d", expectedSizeForFilter, len(bf.Bits))
			}
		})
	}
}

func TestHash(t *testing.T) {
	m, k := 1000, 3
	bf := filter.NewBloomFilter(m, k)
	testData := [][]byte{
		[]byte("RAGNAR"),
		[]byte("New value 1"),
		[]byte("New value 2 but very new"),
		[]byte("New value 3 but this one has some money"),
	}

	for _, data := range testData {
		h1 := bf.Hash(data)
		h2 := bf.Hash(data)

		if len(h1) != k {
			t.Errorf("expected length %d, got %d", k, len(h1))
		}
		if len(h2) != k {
			t.Errorf("expected length %d, got %d", k, len(h2))
		}

		for i := range k {
			if h1[i] != h2[i] {
				t.Errorf("expected equlity between hashes got h1: %d, h2: %d", h1[i], h2[i])
			}
		}
	}
}

func TestInsert(t *testing.T) {
	m, k := 1000, 3
	bf := filter.NewBloomFilter(m, k)

	testData := [][]byte{
		[]byte("RAGNAR"),
		[]byte("New value 1"),
		[]byte("New value 2 but very new"),
		[]byte("New value 3 but this one has some money"),
	}

	// Insert test data
	for _, data := range testData {
		bf.Insert(data)
	}

	// check that the bits with index contained in bf.hash(data) is set to true
	for _, data := range testData {
		h := bf.Hash(data)
		if len(h) != k {
			t.Errorf("expected length %d, got %d", k, len(h))
		}

		for i := range k {
			pos := h[i] / 64
			rem := h[i] % 64
			if (bf.Bits[pos] >> rem) & 1  == 0 {
				t.Errorf("unexpected false negative for data %s", string(data))
			}
		}

	}
}
func TestExist(t *testing.T) {
	bf := filter.NewBloomFilter(1000, 3)
	
	testData := [][]byte{
		[]byte("RAGNAR"),
		[]byte("New value 1"),
		[]byte("New value 2 but very new"),
		[]byte("New value 3 but this one has some money"),
		[]byte("apple"),
		[]byte("banana"),
		[]byte("cherry"),
	}
	
	// Insert test data
	for _, data := range testData {
		bf.Insert(data)
	}
	
	// checks if inserted items exist
	for _, data := range testData {
		if !bf.Exist(data) {
			t.Errorf("expected %s to exist in filter", string(data))
		}
	}
}

func TestNoFalseNegatives(t *testing.T) {
	bf := filter.NewBloomFilter(10000, 5)
	
	testItems := make([][]byte, 100)
	for i := range testItems {
		testItems[i] = []byte(fmt.Sprintf("item_%d", i))
		bf.Insert(testItems[i])
	}
	
	// All inserted items must exist
	for _, item := range testItems {
		if !bf.Exist(item) {
			t.Errorf("False negative: %s should exist but doesn't", string(item))
		}
	}
}


func TestFalsePositiveRate(t *testing.T) {
	m := 10000 
	k := 5      
	n := 1000   
	e := 0.015 // at most
	bf := filter.NewBloomFilter(m, k)
	
	// Insert n items
	insertedItems := make(map[string]bool)
	for i := 0; i < n; i++ {
		item := fmt.Sprintf("inserted_%d", i)
		bf.Insert([]byte(item))
		insertedItems[item] = true
	}
	
	// Test false positive rate with non-inserted items
	falsePositives := 0
	testCount := 10000
	
	rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < testCount; i++ {
		item := fmt.Sprintf("test_%d_%d", i, rand.Intn(100000))
		if !insertedItems[item] && bf.Exist([]byte(item)) {
			falsePositives++
		}
	}
	
	falsePositiveRate := float64(falsePositives) / float64(testCount)
	
	if falsePositiveRate > e {
		t.Errorf("False positive rate too high: %f (expected <= %f)", falsePositiveRate, e)
	}
	
	t.Logf("False positive rate: %f", falsePositiveRate)
}