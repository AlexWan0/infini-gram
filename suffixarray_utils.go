package main

import (
	"fmt"
	"infinigram/suffixarray"
)

type SuffixArray interface {
	retrieveNum(corpusVec TokenArray, query []byte) int                              // retrieve number of continuations
	retrieveSubstrings(corpusVec TokenArray, query []byte, numExtend int64) [][]byte // retrieve all continuations
}

// Wrapper around suffix arrays corresponding to multiple chunks
// of data. Will sum over the results of each chunk.
type MultiSuffixArray struct {
	suffixArrays []SuffixArrayData // suffix array for each chunk of documents
}

// Create a multi-suffix array from a list of suffix array paths.
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

// Retrieve the number of suffix arrays.
func (msa *MultiSuffixArray) numArrays() int {
	return len(msa.suffixArrays)
}

// Retrieve the suffix array for a given index.
func (msa *MultiSuffixArray) getArray(idx int) (SuffixArrayData, error) {
	return msa.suffixArrays[idx], nil
}

// Retrieve the number of continuations. Sums over results from each chunk.
func (msa *MultiSuffixArray) retrieveNum(vec TokenArray, query []byte) int {
	numResults := 0
	for i := 0; i < msa.numArrays(); i++ {
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
	for i := 0; i < msa.numArrays(); i++ {
		arr, err := msa.getArray(i)
		if err != nil {
			return nil // TODO: handle error here
		}
		substrings := retrieveSubstrings(arr, vec, query, extend)
		results = append(results, substrings...)
	}
	return results
}

// Perform left or right binary search on the suffix array.
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

// Search for the occurrences of a query in the suffix array.
// Returns the starting and ending positions of the occurrences.
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

// Uses binary search to find the occurrences of a query in the
// suffix array. Returns the starting position of the occurences.
func retrieve(suffixArray SuffixArrayData, vec TokenArray, query []byte) []int64 {
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

// Retrieve the number of occurrences of a query in the suffix array.
func retrieveNum(suffixArray SuffixArrayData, vec TokenArray, query []byte) int {
	startIdx, endIdx := arraySearch(suffixArray, vec, query)

	if (startIdx == -1) && (endIdx == -1) {
		return 0
	}

	return int(endIdx - startIdx + 1)
}

// Retrieve all occurrences of a query in the suffix array. The returned occurrences
// are extended by extend tokens.
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

// Encode a sequence of integers into a byte array ending in the sentinal.
// The sentinalVal is repeated sentinalSize times.
func encodeSequence(valueBytes []byte, values []uint32, sentinalVal int, sentinalSize int) {
	size := len(values)

	for i := 0; i < size; i++ {
		putByte(valueBytes, uint16(values[i]), i)
	}

	for i := 0; i < sentinalSize; i++ {
		putByte(valueBytes, uint16(sentinalVal), size+i)
	}
}

// Create a suffix array for a given byte array.
// Each token is a uint16 value, so will take up two bytes. This means that
// only even indices in the suffix array are valid. The unaligned suffix
// array returned will contain odd values.
func createUnalignedSuffixArray(valueBytes []byte) []int64 {
	suffixArray := make([]int64, len(valueBytes))
	suffixarray.Text_64(valueBytes, suffixArray)

	return suffixArray
}
