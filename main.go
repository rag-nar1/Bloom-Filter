package main

import (
	// "fmt"
	"math/rand"
	"time"

	BloomFilter "github.com/rag-nar1/Bloom-Filter/filter/bloom"
)

const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" + "0123456789"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandomString(length int, charset string) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return b
}

func main() {
	n := 1_000_000
	fpRate := 0.01
	bf := BloomFilter.NewBloomFilter(uint64(n), fpRate)
	// n := 1_000_000
	// startTime := time.Now()
	randData := make(map[string]bool, n)
	ln := 0
	for _ = range n {
		s := RandomString(seededRand.Intn(1000), charset)
		bf.Insert(s)
		ln += len(s)
		randData[string(s)] = true
	}

	fn, fp := 0, 0
	for s, _ := range randData {
		if !bf.Exist([]byte(s)) {
			fn++	
		}
	}

	for _ = range n {
		s := RandomString(seededRand.Intn(len(charset)), charset)
		if _, ok := randData[string(s)]; !ok && bf.Exist(s) {
			fp++
		} else if ok && !bf.Exist(s) {
			fn++
		}
	}

	// fmt.Println("string avg length: ", ln/n, "    filter Size:", m/8/1024/1024, "mb", "  number of hash functions:", k)
	// fmt.Println("Number of insert op: ", n)
	// fmt.Println("Number of lookups: ", 2*n,"\n")
	// fmt.Println("Time taken: " , time.Since(startTime).Seconds() , "sec")
	// fmt.Println("false positives: ", fp, "false negative: ", fn, "\nError: ", float32(fp)/float32(n)*100.0,"%")
}
