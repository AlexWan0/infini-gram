package main

import (
	"sort"
)

func compare_slices(a []byte, b []byte) int {
	if len(a) != len(b) {
		panic("lengths must be equal")
	}

	// assumes the two arrays are the same size
	n := len(a)
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func compare_slices_int64(a []int64, b []int64) int {
	if len(a) != len(b) {
		panic("lengths must be equal")
	}

	// assumes the two arrays are the same size
	n := len(a)
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func argsort(vec []float32, descending bool) []int {
	indices := make([]int, len(vec))
	for i := 0; i < len(vec); i++ {
		indices[i] = i
	}

	sort.Slice(indices, func(i, j int) bool {
		if descending {
			return vec[indices[j]] < vec[indices[i]]
		}
		return vec[indices[i]] < vec[indices[j]]
	})
	
	return indices
}