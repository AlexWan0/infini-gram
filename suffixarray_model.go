package main

import (
	"fmt"
	"os"
	"path"
	"strings"
)

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

// Creates the tokenized corpus and suffix array, saves them to outpath, and returns
// the model. If either the tokenized corpus or suffix array already exist, they will
// be loaded from disk. Set sentinalVal and sentinalSize to split documents.
// Tokenization is done in parallel using nWorkers. Set lineSplit to the token that
// separates documents in the input file (filename). Set tokenizerConfig to the path
// of the tokenizer configuration file. vocabSize is the size of the vocabulary.
// Creates a suffix array for each chunk of documents of size chunkSize.
func InitializeSuffixArrayModel(filename, lineSplit, outpath, tokenizerConfig string, sentinalVal, sentinalSize, nWorkers, vocabSize, chunkSize int) (*SuffixArrayModel, error) {
	// check whether tokenized data already exists
	dataPath := path.Join(outpath, "data.bin")
	_, err := os.Stat(dataPath)
	if err != nil {
		// tokenize data: streams documents from text file into binary file
		fmt.Println("Tokenizing data to disk")
		_, err := tokenizeMultiprocess(filename, lineSplit, outpath, tokenizerConfig, sentinalVal, sentinalSize, nWorkers)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("Tokenized data already found")
	}

	dataBytes, err := loadMMappedArray(dataPath)
	if err != nil {
		return nil, err
	}

	// check whether suffix array already exists
	saChunkPathsPath := path.Join(outpath, "suffix_array_paths.txt")

	_, err = os.Stat(saChunkPathsPath)
	if err == nil {
		fmt.Println("Suffix array(s) already found")

		saChunkPathsStr, err := readStringFromFile(saChunkPathsPath)
		if err != nil {
			return nil, err
		}

		suffixArray, err := makeMultiSuffixArray(strings.Split(saChunkPathsStr, "\n"))
		if err != nil {
			return nil, err
		}

		return &SuffixArrayModel{
			suffixArray: suffixArray,
			bytesData:   dataBytes,
			vocabSize:   vocabSize,
		}, nil
	}

	fmt.Println("Creating suffix array(s)")
	offset := int64(0)
	currChunk := 0
	chunkBuffer := make([]byte, chunkSize)
	saCallback := func(chunkLength int) error {
		fmt.Printf("making chunk %d of size %d\n", currChunk, chunkLength)

		readValues := chunkBuffer[:chunkLength]

		unalignedSa := createUnalignedSuffixArray(readValues)

		saChunkPath := path.Join(outpath, fmt.Sprintf("suffix_array_%d.bin", currChunk))
		err = writeIndicesToFile(saChunkPath, unalignedSa, offset)
		if err != nil {
			return err
		}

		currChunk += 1
		offset += int64(chunkLength)

		return nil
	}
	err = documentIter(dataPath, sentinalSize, sentinalVal, chunkBuffer, saCallback)
	if err != nil {
		return nil, err
	}

	// get list of filenames
	numChunks := currChunk
	saChunkPaths := make([]string, numChunks)
	for i := 0; i < numChunks; i++ {
		saChunkPaths[i] = path.Join(outpath, fmt.Sprintf("suffix_array_%d.bin", i))
	}

	// save list of filenames to disk
	err = writeStringToFile(saChunkPathsPath, strings.Join(saChunkPaths, "\n"))
	if err != nil {
		return nil, err
	}

	suffixArray, err := makeMultiSuffixArray(saChunkPaths)
	if err != nil {
		return nil, err
	}

	return &SuffixArrayModel{
		suffixArray: suffixArray,
		bytesData:   dataBytes,
		vocabSize:   vocabSize,
	}, nil
}
