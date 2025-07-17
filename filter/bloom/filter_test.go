package bloom_test

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	filter "github.com/rag-nar1/Filters/filter"
	filterBloom "github.com/rag-nar1/Filters/filter/bloom"
)

func TestNewBloomFilter(t *testing.T) {
	tests := []struct {
		N    uint64
		fpRate float64
		name string
	}{
		{100, 0.01, "test1"},
		{1000, 0.05, "test2"},
		{10000, 0.1, "test3"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bf := filterBloom.NewBloomFilter(test.N, test.fpRate, filter.DoubleHash)

			// checking any intilization problems
			if bf.M <= 0 {
				t.Errorf("expected m>0, got %d", bf.M)
			}
			if bf.K <= 0 {
				t.Errorf("expected k>0, got %d", bf.K)
			}
			if len(bf.Bits) <= 0 {
				t.Errorf("expected bit array size >0, got %d", len(bf.Bits))
			}

			if bf.Hasher == nil {
				t.Errorf("expected hash function, got nil")
			}
		})
	}
}

func TestHash(t *testing.T) {
	n := 1000
	fpRate := 0.01
	bf := filterBloom.NewBloomFilter(uint64(n), fpRate, filter.DoubleHash)
	testData := [][]byte{
		[]byte("RAGNAR"),
		[]byte("New value 1"),
		[]byte("New value 2 but very new"),
		[]byte("New value 3 but this one has some money"),
	}

	for _, data := range testData {
		h1 := bf.Hash(data)
		h2 := bf.Hash(data)

		if len(h1) != bf.K {
			t.Errorf("expected length %d, got %d", bf.K, len(h1))
		}
		if len(h2) != bf.K {
			t.Errorf("expected length %d, got %d", bf.K, len(h2))
		}

		for i := range bf.K {
			if h1[i] != h2[i] {
				t.Errorf("expected equlity between hashes got h1: %d, h2: %d", h1[i], h2[i])
			}
		}
	}
}

func TestInsert(t *testing.T) {
	n := 1000
	fpRate := 0.01
	bf := filterBloom.NewBloomFilter(uint64(n), fpRate, filter.DoubleHash)

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
		if len(h) != bf.K {
			t.Errorf("expected length %d, got %d", bf.K, len(h))
		}

		for i := range bf.K {
			pos := h[i] / 64
			rem := h[i] % 64
			if (bf.Bits[pos] >> rem) & 1  == 0 {
				t.Errorf("unexpected false negative for data %s", string(data))
			}
		}

	}
}
func TestExist(t *testing.T) {
	n := 1000
	fpRate := 0.01
	bf := filterBloom.NewBloomFilter(uint64(n), fpRate, filter.DoubleHash)
	
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
	n := 100
	fpRate := 0.05
	bf := filterBloom.NewBloomFilter(uint64(n), fpRate, filter.DoubleHash)
	
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
	n := 1000   
	e := 0.01 // at most
	bf := filterBloom.NewBloomFilter(uint64(n), e, filter.DoubleHash)
	
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
	
	if math.Abs(falsePositiveRate - e) > 0.01 {
		t.Errorf("False positive rate too high: %f (expected <= %f)", falsePositiveRate, e)
	}
	
	t.Logf("False positive rate: %f", falsePositiveRate)
}