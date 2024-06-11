package main

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
)

// Puts the value into the byte slice at the given index.
func putByte(vec []byte, val uint16, idx int) {
	binary.LittleEndian.PutUint16(vec[idx*2:], val)
}

func intToByte(vec []uint32) []byte {
	result := make([]byte, len(vec)*2)

	for i, val := range vec {
		putByte(result, uint16(val), i)
	}

	return result
}

func intToUint32(vec []int) []uint32 {
	result := make([]uint32, len(vec))

	for i, val := range vec {
		result[i] = uint32(val)
	}

	return result
}

func uint32ToInt(vec []uint32) []int {
	result := make([]int, len(vec))

	for i, val := range vec {
		result[i] = int(val)
	}

	return result
}

func byteToInt(vec []byte) []int {
	n := len(vec) / 2
	result := make([]int, n)

	for i := 0; i < n; i++ {
		result[i] = int(binary.LittleEndian.Uint16(vec[i*2:]))
	}

	return result
}

func byteToUInt16(vec []byte) []uint16 {
	n := len(vec) / 2
	result := make([]uint16, n)

	for i := 0; i < n; i++ {
		result[i] = binary.LittleEndian.Uint16(vec[i*2:])
	}

	return result
}

func printsVec(vec []int) string {
	strVals := make([]string, len(vec))
	for i, num := range vec {
		strVals[i] = fmt.Sprintf("%d", num)
	}
	return strings.Join(strVals, " ")
}

type Ordered interface {
	~int | ~int64 | ~byte | ~float64 | ~string
}

// Compare whether a is before b lexographically.
// Returns -1 if a < b, 0 if a == b, and 1 if a > b.
func compareSlices[T Ordered](a, b []T) int {
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
	maxVal := vec[0]
	maxIdx := 0
	for i, val := range vec {
		if val > maxVal {
			maxVal = val
			maxIdx = i
		}
	}
	return maxIdx
}

func changeEndianness16(vec []byte) {
	for i := 0; i < len(vec); i += 2 {
		vec[i], vec[i+1] = vec[i+1], vec[i]
	}
}
