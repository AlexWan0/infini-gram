package main

import (
	"fmt"
	"infinigram/suffixarray"
)

type SuffixArray interface {
	numArrays() int
	getArray(int) ([]int64, error)
	retrieveNum(TokenArray, []byte) int
	retrieveSubstrings(TokenArray, []byte, int64) [][]byte
}

type MultiSuffixArray struct {
	suffixArrayPaths []string
	loadedArray      []int64
	loadedArrayIdx   int
}

func makeMultiSuffixArray(suffixArrayPaths []string) *MultiSuffixArray {
	return &MultiSuffixArray{
		suffixArrayPaths: suffixArrayPaths,
		loadedArray:      nil,
		loadedArrayIdx:   -1,
	}
}

func (msa *MultiSuffixArray) numArrays() int {
	return len(msa.suffixArrayPaths)
}

func (msa *MultiSuffixArray) getArray(idx int) ([]int64, error) {
	if msa.loadedArrayIdx != -1 && msa.loadedArrayIdx == idx {
		fmt.Printf("suffix array %d is cached\n", idx)
		return msa.loadedArray, nil
	}

	saPath := msa.suffixArrayPaths[idx]

	fmt.Printf("loading suffix array %d from %s\n", idx, saPath)
	suffixArray, err := readInt64FromFile(saPath)
	if err != nil {
		return nil, err
	}

	msa.loadedArray = suffixArray
	msa.loadedArrayIdx = idx

	return suffixArray, nil
}

func (msa *MultiSuffixArray) getLoadOrder() []int {
	defaultOrder := make([]int, msa.numArrays())
	for i := 0; i < msa.numArrays(); i++ {
		defaultOrder[i] = i
	}

	if msa.loadedArrayIdx == -1 {
		return defaultOrder
	}

	// move the loaded array to the front
	loadedIdx := msa.loadedArrayIdx
	defaultOrder[0] = loadedIdx
	defaultOrder[loadedIdx] = 0

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

func binarySearch(suffixArray []int64, vec TokenArray, query []byte, left bool) int64 {
	queryLen := int64(len(query))
	saLen := int64(len(suffixArray))
	vecLen := vec.length()

	start := int64(0)
	end := saLen
	for start < end {
		mid := int64((start + end) / 2)
		midSlice := vec.getSlice(suffixArray[mid], min(suffixArray[mid]+queryLen, vecLen))

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

func arraySearch(suffixArray []int64, vec TokenArray, query []byte) (int64, int64) {
	// bisect left; all values to the left are <, all values to the right are >=
	occStart := binarySearch(suffixArray, vec, query, true)

	// if found, occStart is the first occurrence
	queryLen := int64(len(query))
	vecLen := vec.length()
	firstOcc := vec.getSlice(suffixArray[occStart], min(suffixArray[occStart]+queryLen, vecLen))

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

func retrieve(suffixArray []int64, vec TokenArray, query []byte) []int64 {
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

func retrieveNum(suffixArray []int64, vec TokenArray, query []byte) int {
	startIdx, endIdx := arraySearch(suffixArray, vec, query)

	if (startIdx == -1) && (endIdx == -1) {
		return 0
	}

	return int(endIdx - startIdx + 1)
}

func retrieveSubstrings(suffixArray []int64, vec TokenArray, query []byte, extend int64) [][]byte {
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
