package main

import (
	"math/rand"
	"testing"
)

func makeRandUint16Array(n int, minVal int, maxVal int) []uint16 {
	result := make([]uint16, n)
	for i := 0; i < n; i++ {
		result[i] = uint16(rand.Intn(maxVal-minVal) + minVal)
	}
	return result
}

func TestBasic(t *testing.T) {
	data := makeRandUint16Array(1000, 0, 5000)

	cache := NewBitCache()

	for i := 0; i < 1000; i += 2 {
		cache.AddValue(data[i], data[i+1])
	}

	// test false positives
	for i := 0; i < 1000; i += 2 {
		if !cache.HasValue(data[i], data[i+1]) {
			t.Errorf("Expected to find value at index %d", i)
		}
	}

	// test false negatives
	dataNeg := makeRandUint16Array(1000, 5001, 10000)
	for i := 0; i < 1000; i += 2 {
		if cache.HasValue(dataNeg[i], dataNeg[i+1]) {
			t.Errorf("Expected to not find value at index %d", i)
		}
	}
}
