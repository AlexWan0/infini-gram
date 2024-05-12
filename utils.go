package main

import (
	"sort"
	"encoding/binary"
	"math/rand"
	"fmt"
	"strings"
)


func rand_vector(size int, min int, max int) []int {
	vec := make([]int, size)
	for i := 0; i < size; i++ {
		vec[i] = rand.Intn(max-min) + min
	}
	return vec
}

func put_byte(vec []byte, val uint16, idx int) {
	binary.LittleEndian.PutUint16(vec[idx * 2:], val)
}

func int_to_byte(vec []int) []byte {
	result := make([]byte, len(vec)*2)

	for i, val := range vec {
		put_byte(result, uint16(val), i)
	}

	return result
}

func byte_to_int(vec []byte) []int {
	n := len(vec) / 2
	result := make([]int, n)

	for i := 0; i < n; i++ {
		result[i] = int(binary.LittleEndian.Uint16(vec[i * 2:]))
	}

	return result
}

func rand_vector_byte(size int, min int, max int) []byte {
	return int_to_byte(rand_vector(size, min, max))
}

func prints_vec(vec []int) string {
	str_vals := make([]string, len(vec))
	for i, num := range vec {
		str_vals[i] = fmt.Sprintf("%d", num)
	}
	return strings.Join(str_vals, " ")
}

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
