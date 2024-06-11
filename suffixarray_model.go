package main

import "fmt"

// Wrapper around infini-gram model data.
type SuffixArrayModel struct {
	suffixArray SuffixArray
	bytesData   TokenArray
	vocabSize   int
}

// Will return the prediction of the next token distribution corresponding to the
// longest suffix in queryIds. For a suffix to be considered valid, there must be
// at least minMatches occurrences of it in the data. The retrieved suffixes will
// include numExtend extra tokens (set to 1 to just get the next token).
func (m *SuffixArrayModel) NextTokenDistribution(queryIds []uint32, numExtend int, minMatches int) *Prediction {
	vocabSize := m.vocabSize
	suffixArray := m.suffixArray
	dataBytes := m.bytesData

	var bestQueryEnc []byte

	if len(queryIds) == 0 {
		bestQueryEnc = make([]byte, 0)
	} else {
		// perform binary search to find longest suffix
		left := 0
		right := len(queryIds) + 1

		for left < right {
			// the current candidate for the longest suffix length
			mid := (left + right) / 2
			queryIdsSuffix := queryIds[len(queryIds)-mid:]

			querySuffixEnc := intToByte(queryIdsSuffix)

			// check if, at this length, we get any matches
			numMatches := suffixArray.retrieveNum(dataBytes, querySuffixEnc)

			if numMatches >= minMatches {
				left = mid + 1
			} else {
				right = mid
			}
		}

		if left == 0 {
			// TODO: i don't think this should happen
			fmt.Println("none found")
			return &Prediction{nil, -1, 0, numExtend, make([][]int, 0)}
		}
		best_n := left - 1

		bestQueryEnc = intToByte(queryIds[len(queryIds)-best_n:])
	}

	substrings := suffixArray.retrieveSubstrings(dataBytes, bestQueryEnc, int64(numExtend))

	rawSuffixes := make([][]int, len(substrings))
	distr := make([]float32, vocabSize)
	total := 0
	for i, s := range substrings {
		retrievedSuffix := byteToInt(s)

		newIds := retrievedSuffix[len(retrievedSuffix)-numExtend:]
		newIds = append([]int{}, newIds...)

		// add to raw retrievals
		rawSuffixes[i] = newIds

		// populate distribution
		distr[newIds[0]] += 1
		total += 1
	}

	for i := range distr {
		distr[i] /= float32(total)
	}

	return &Prediction{distr, len(bestQueryEnc) / 2, total, numExtend, rawSuffixes}
}
