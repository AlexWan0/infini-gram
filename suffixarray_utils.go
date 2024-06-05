package main

import (
	"infinigram/suffixarray"
)

func arraySearch(suffixArray []int64, vec []byte, query []byte) (int64, int64) {
	queryLen := int64(len(query))
	saLen := int64(len(suffixArray))
	vecLen := int64(len(vec))

	start := int64(0)
	end := saLen
	for start < end {
		mid := int64((start + end) / 2)
		midSlice := vec[suffixArray[mid]:min(suffixArray[mid]+queryLen, vecLen)]

		cmpValue := compareSlices(midSlice, query)

		if cmpValue < 0 {
			start = mid + 1
		} else {
			end = mid
		}
	}

	if start >= saLen {
		return -1, -1
	}

	startSlice := vec[suffixArray[start]:min(suffixArray[start]+queryLen, vecLen)]
	if (start == saLen) || (compareSlices(startSlice, query) != 0) {
		return -1, -1
	}

	firstOcc := start

	end = saLen
	for start < end {
		mid := int64((start + end) / 2)
		midSlice := vec[suffixArray[mid]:min(suffixArray[mid]+queryLen, vecLen)]

		cmpValue := compareSlices(midSlice, query)

		if (cmpValue == 0) || (cmpValue == -1) {
			start = mid + 1
		} else {
			end = mid
		}
	}

	lastOcc := start - 1

	return firstOcc, lastOcc
}

func retrieve(suffixArray []int64, vec []byte, query []byte) []int64 {
	// use binary search to find matching prefixes
	// return start positions of suffixes

	startIdx, endIdx := arraySearch(suffixArray, vec, query)

	if (startIdx == -1) && (endIdx == -1) {
		return make([]int64, 0)
	}

	suffixStarts := make([]int64, 0, endIdx-startIdx+1)

	for s := startIdx; s <= endIdx; s++ {
		startPos := suffixArray[s]
		suffixStarts = append(suffixStarts, startPos)
	}

	return suffixStarts
}

func retrieveNum(suffixArray []int64, vec []byte, query []byte) int {
	startIdx, endIdx := arraySearch(suffixArray, vec, query)

	if (startIdx == -1) && (endIdx == -1) {
		return 0
	}

	return int(endIdx - startIdx + 1)
}

func retrieveSubstrings(suffixArray []int64, vec []byte, query []byte, extend int64) [][]byte {
	suffixStarts := retrieve(suffixArray, vec, query)

	n_result := len(suffixStarts)
	queryLen := int64(len(query))

	resultSlices := make([][]byte, n_result)
	for i, start := range suffixStarts {
		resultSlices[i] = vec[start : start+queryLen+(extend*2)]
	}

	return resultSlices
}

func encodeSequence(valueBytes []byte, values []uint32, sentinalVal int, sentinalSize int) {
	size := len(values)

	for i := 0; i < size; i++ {
		putByte(valueBytes, uint16(values[i]), i)
	}

	for i := 0; i < sentinalSize; i++ {
		putByte(valueBytes, uint16(sentinalVal), size+i)
	}
}

func createUnalignedSuffixArray(valueBytes []byte) []int64 {
	// not all indices in suffix array align with byte boundaries
	// the non-aligned indices are filtered while writing to disk

	suffixArray := make([]int64, len(valueBytes))
	suffixarray.Text_64(valueBytes, suffixArray)

	return suffixArray
}
