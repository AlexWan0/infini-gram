package main

import (
	"sort"
	"encoding/binary"
	"fmt"
	"strings"
)


func put_byte(vec []byte, val uint16, idx int) {
	binary.LittleEndian.PutUint16(vec[idx * 2:], val)
}

func int_to_byte(vec []uint32) []byte {
	result := make([]byte, len(vec)*2)

	for i, val := range vec {
		put_byte(result, uint16(val), i)
	}

	return result
}

func int_to_uint32(vec []int) []uint32 {
	result := make([]uint32, len(vec))

	for i, val := range vec {
		result[i] = uint32(val)
	}

	return result
}

func uint32_to_int(vec []uint32) []int {
	result := make([]int, len(vec))

	for i, val := range vec {
		result[i] = int(val)
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

func prints_vec(vec []int) string {
	str_vals := make([]string, len(vec))
	for i, num := range vec {
		str_vals[i] = fmt.Sprintf("%d", num)
	}
	return strings.Join(str_vals, " ")
}

type Ordered interface {
	~int | ~int64 | ~byte | ~float64 | ~string
}

func compare_slices[T Ordered](a, b []T) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	} else if len(a) > len(b) {
		return 1
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

func argmax(vec []float32) int {
	max_val := vec[0]
	max_idx := 0
	for i, val := range vec {
		if val > max_val {
			max_val = val
			max_idx = i
		}
	}
	return max_idx
}