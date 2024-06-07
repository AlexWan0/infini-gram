package main

import (
	"fmt"
	"infinigram/suffixarray"
)

type SuffixArray interface {
	numArrays() int
	getArray(int) (SuffixArrayData, error)
	retrieveNum(TokenArray, []byte) int
	retrieveSubstrings(TokenArray, []byte, int64) [][]byte
}

type MultiSuffixArray struct {
	suffixArrays []SuffixArrayData
}

func makeMultiSuffixArray(suffixArrayPaths []string) (*MultiSuffixArray, error) {
	suffixArrays := make([]SuffixArrayData, len(suffixArrayPaths))
	for i, path := range suffixArrayPaths {
		newSA, err := makeMMappedSA(path)
		if err != nil {
			return nil, err
		}
		suffixArrays[i] = newSA
	}

	return &MultiSuffixArray{suffixArrays: suffixArrays}, nil
}

func (msa *MultiSuffixArray) numArrays() int {
	return len(msa.suffixArrays)
}

func (msa *MultiSuffixArray) getArray(idx int) (SuffixArrayData, error) {
	return msa.suffixArrays[idx], nil
}

func (msa *MultiSuffixArray) getLoadOrder() []int {
	defaultOrder := make([]int, msa.numArrays())
	for i := 0; i < msa.numArrays(); i++ {
		defaultOrder[i] = i
	}
	return defaultOrder
}

func (msa *MultiSuffixArray) retrieveNum(vec TokenArray, query []byte) int {
	numResults := 0
	for _, i := range msa.getLoadOrder() {
		arr, err := msa.getArray(i)
		if err != nil {
			return 0 // TODO: handle error here
		}

		numResults += retrieveNum(arr, vec, query)
		fmt.Printf("retrieved from chunk #%d: suffix_size=%d, occurrences=%d\n", i, len(query)/2, numResults)
	}

	fmt.Printf("suffix of size %d has %d total occurrences\n", len(query)/2, numResults)

	return numResults
}

func (msa *MultiSuffixArray) retrieveSubstrings(vec TokenArray, query []byte, extend int64) [][]byte {
	results := make([][]byte, 0)
	for _, i := range msa.getLoadOrder() {
		arr, err := msa.getArray(i)
		if err != nil {
			return nil // TODO: handle error here
		}
		substrings := retrieveSubstrings(arr, vec, query, extend)
		results = append(results, substrings...)
	}
	return results
}

func binarySearch(suffixArray SuffixArrayData, vec TokenArray, query []byte, left bool) int64 {
	queryLen := int64(len(query))
	saLen := int64(suffixArray.length())
	vecLen := vec.length()

	start := int64(0)
	end := saLen
	for start < end {
		mid := int64((start + end) / 2)
		midIdx := suffixArray.get(mid)
		midSlice := vec.getSlice(midIdx, min(midIdx+queryLen, vecLen))

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

func arraySearch(suffixArray SuffixArrayData, vec TokenArray, query []byte) (int64, int64) {
	// bisect left; all values to the left are <, all values to the right are >=
	occStart := binarySearch(suffixArray, vec, query, true)

	// if found, occStart is the first occurrence
	queryLen := int64(len(query))
	vecLen := vec.length()
	startIdx := suffixArray.get(occStart)
	firstOcc := vec.getSlice(startIdx, min(startIdx+queryLen, vecLen))

	if compareSlices(firstOcc, query) != 0 {
		return -1, -1
	}

	// bisect right; all values to the left are <=, all values to the right are >
	occEnd := binarySearch(suffixArray, vec, query, false)

	// if the two indices are the same, the query is not present
	if occStart == occEnd {
		return -1, -1
	}

	return occStart, occEnd - 1
}

func retrieve(suffixArray SuffixArrayData, vec TokenArray, query []byte) []int64 {
	// use binary search to find matching prefixes
	// return start positions of suffixes

	startIdx, endIdx := arraySearch(suffixArray, vec, query)

	if (startIdx == -1) && (endIdx == -1) {
		return make([]int64, 0)
	}

	suffixStarts := make([]int64, 0, endIdx-startIdx+1)

	for s := startIdx; s <= endIdx; s++ {
		startPos := suffixArray.get(s)
		suffixStarts = append(suffixStarts, startPos)
	}

	return suffixStarts
}

func retrieveNum(suffixArray SuffixArrayData, vec TokenArray, query []byte) int {
	startIdx, endIdx := arraySearch(suffixArray, vec, query)

	if (startIdx == -1) && (endIdx == -1) {
		return 0
	}

	return int(endIdx - startIdx + 1)
}

func retrieveSubstrings(suffixArray SuffixArrayData, vec TokenArray, query []byte, extend int64) [][]byte {
	suffixStarts := retrieve(suffixArray, vec, query)

	n_result := len(suffixStarts)
	queryLen := int64(len(query))

	resultSlices := make([][]byte, n_result)
	for i, start := range suffixStarts {
		resultSlices[i] = vec.getSlice(start, start+queryLen+(extend*2))
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
