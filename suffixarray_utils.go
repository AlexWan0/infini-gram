package main

import (
	"infinigram/suffixarray"
)

func binarySearch(suffixArray []int64, vec []byte, query []byte, left bool) int64 {
	queryLen := int64(len(query))
	saLen := int64(len(suffixArray))
	vecLen := int64(len(vec))

	start := int64(0)
	end := saLen
	for start < end {
		mid := int64((start + end) / 2)
		midSlice := vec[suffixArray[mid]:min(suffixArray[mid]+queryLen, vecLen)]

		cmpValue := compareSlices(midSlice, query)

		cmpResult := cmpValue < 0
		if !left {
			cmpResult = cmpValue <= 0
		}

		if cmpResult {
			start = mid + 1
		} else {
			end = mid
		}
	}

	return start
}

func arraySearch(suffixArray []int64, vec []byte, query []byte) (int64, int64) {
	// bisect left; all values to the left are <, all values to the right are >=
	occStart := binarySearch(suffixArray, vec, query, true)

	// bisect right; all values to the left are <=, all values to the right are >
	occEnd := binarySearch(suffixArray, vec, query, false)

	// if the two indices are the same, the query is not present
	if occStart == occEnd {
		return -1, -1
	}

	return occStart, occEnd - 1
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
