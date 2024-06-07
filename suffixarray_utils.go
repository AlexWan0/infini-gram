package main

import (
	"fmt"
	"infinigram/suffixarray"
)

type SuffixArray interface {
	numArrays() int
	getArray(int) ([]int64, error)
	retrieveNum([]byte, []byte) int
	retrieveSubstrings([]byte, []byte, int64) [][]byte
}

type MultiSuffixArray struct {
	suffixArrayPaths []string
}

func (msa *MultiSuffixArray) numArrays() int {
	return len(msa.suffixArrayPaths)
}

func (msa *MultiSuffixArray) getArray(idx int) ([]int64, error) {
	saPath := msa.suffixArrayPaths[idx]

	suffixArray, err := readInt64FromFile(saPath)
	if err != nil {
		return nil, err
	}

	return suffixArray, nil
}

func (msa *MultiSuffixArray) retrieveNum(vec []byte, query []byte) int {
	// // debug; just use the first one
	// arr, err := msa.getArray(0)
	// if err != nil {
	// 	return 0 // TODO: handle error here
	// }

	numArrays := msa.numArrays()
	numResults := 0
	for i := 0; i < numArrays; i++ {
		arr, err := msa.getArray(i)
		if err != nil {
			return 0 // TODO: handle error here
		}

		// fmt.Println("---- arr", i, len(arr), "-----")
		// for s := 0; s < len(arr); s++ {
		// 	startPos := arr[s]
		// 	fmt.Println(vec[startPos : startPos+2])
		// }
		// fmt.Println("--------------")

		numResults += retrieveNum(arr, vec, query)
		fmt.Printf("%d: numResults=%d\n", i, numResults)
	}
	return numResults
}

func (msa *MultiSuffixArray) retrieveSubstrings(vec []byte, query []byte, extend int64) [][]byte {
	// debug; just use the first one
	// arr, err := msa.getArray(0)
	// if err != nil {
	// 	return nil // TODO: handle error here
	// }
	// return retrieveSubstrings(arr, vec, query, extend)

	numArrays := msa.numArrays()
	results := make([][]byte, 0)
	for i := 0; i < numArrays; i++ {
		arr, err := msa.getArray(i)
		if err != nil {
			return nil // TODO: handle error here
		}
		substrings := retrieveSubstrings(arr, vec, query, extend)
		results = append(results, substrings...)
	}
	return results
}

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
		// fmt.Println("bs", mid, midSlice, query, cmpValue)

		// -1 -1 -1 0 0 0 1 1 1
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

	// fmt.Println("occStart", occStart)
	// fmt.Println("query", query)

	// if found, occStart is the first occurrence
	queryLen := int64(len(query))
	vecLen := int64(len(vec))
	firstOcc := vec[suffixArray[occStart]:min(suffixArray[occStart]+queryLen, vecLen)]
	// fmt.Println("firstOcc", firstOcc, query)
	if compareSlices(firstOcc, query) != 0 {
		return -1, -1
	}

	// fmt.Println("first", firstOcc)

	// bisect right; all values to the left are <=, all values to the right are >
	occEnd := binarySearch(suffixArray, vec, query, false)

	// fmt.Println("occEnd", occEnd)

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

	// fmt.Println("suffixes")
	for s := startIdx; s <= endIdx; s++ {
		startPos := suffixArray[s]
		suffixStarts = append(suffixStarts, startPos)
		// fmt.Println(vec[startPos : startPos+8])
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
