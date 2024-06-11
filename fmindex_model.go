package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"

	wavelettree "github.com/sekineh/go-watrix"
)

type FMIndexModel struct {
	tree      wavelettree.WaveletTree
	counts    [256]int64
	vocabSize int
}

func saToBWT(sa []int64, vec []byte) ([]byte, [256]int64) {
	bwtBack := make([]byte, len(sa))
	symbCount := [256]int64{}

	for i, suffixIdx := range sa {
		prevIdx := suffixIdx - 1
		if prevIdx < 0 {
			prevIdx = int64(len(vec) - 1)
		}
		backSymbol := vec[prevIdx]

		bwtBack[i] = backSymbol
	}

	for _, symbol := range vec {
		symbCount[symbol]++
	}

	return bwtBack, symbCount
}

func makeWaveletTree(vec []byte) wavelettree.WaveletTree {
	builder := wavelettree.NewBuilder()
	for _, v := range vec {
		builder.PushBack(uint64(v))
	}
	wt := builder.Build()

	return wt
}

func saveWaveletTree(wt wavelettree.WaveletTree, filename string) error {
	wtBytes, err := wt.MarshalBinary()
	if err != nil {
		return err
	}

	err = writeBytesToFile(filename, wtBytes)
	if err != nil {
		return err
	}

	return nil
}

func loadWaveletTree(filename string) (wavelettree.WaveletTree, error) {
	wtBytes, err := readBytesFromFile(filename)
	if err != nil {
		return nil, err
	}

	wt := wavelettree.New()
	err = wt.UnmarshalBinary(wtBytes)
	if err != nil {
		return nil, err
	}

	return wt, nil
}

func getLongestSuffix(query []byte, counts [256]int64, wt wavelettree.WaveletTree, minMatches int) (int, uint64) {
	countPrefixSum := make([]uint64, 256) // number of symbols before i
	for i := 1; i < 256; i++ {
		countPrefixSum[i] = countPrefixSum[i-1] + uint64(counts[i-1])
	}

	// fmt.Println("symbol prefix sum", countPrefixSum)

	// fmt.Println("query encoded:", query)

	index := len(query) - 1
	currChar := query[index]
	if counts[currChar] == 0 {
		return 0, 0
	}

	start, end := countPrefixSum[currChar], countPrefixSum[currChar]+uint64(counts[currChar])
	count := end - start
	pastCount := count
	// fmt.Printf("[%d, %d] idx=%d, currChar=%d\n", start, end, index, currChar)
	longestSuffix := 0

	index--
	for count >= uint64(minMatches) && index >= 0 {
		// character we should look for in the prev position
		currChar = query[index]

		// number of this character that we found
		prevCounts := wt.Rank(start, uint64(currChar))
		allCounts := wt.Rank(end, uint64(currChar))
		count = allCounts - prevCounts

		// if count > uint64(counts[currChar]) {
		// 	panic("count is greater than number of symbols")
		// }

		// we found a valid suffix and we're aligned with byte boundaries: update
		if count >= uint64(minMatches) && index%2 == 0 {
			longestSuffix += 2
			pastCount = count
			// fmt.Println("found longer", longestSuffix, index, pastCount)
		}

		start = countPrefixSum[currChar] + prevCounts
		end = start + count
		// fmt.Printf("[%d, %d] idx=%d, currChar=%d\n", start, end, index, currChar)

		index--
	}

	return longestSuffix, pastCount
}

func makeFMIndex(sa []int64, vec []byte, vocabSize int) *FMIndexModel {
	// for rowIdx, idx := range sa {
	// 	fmt.Print(rowIdx, vec[idx:])
	// 	if idx > 0 {
	// 		fmt.Println("", vec[idx-1])
	// 	} else {
	// 		fmt.Println("", vec[len(vec)-1])
	// 	}
	// }

	bwt, counts := saToBWT(sa, vec)
	wt := makeWaveletTree(bwt)
	return &FMIndexModel{wt, counts, vocabSize}
}

func (bw *FMIndexModel) GetLongestSuffix(query []byte) (int, uint64) {
	return getLongestSuffix(query, bw.counts, bw.tree, 1)
}

func (bw *FMIndexModel) Save(filepath string) error {
	countsPath := path.Join(filepath, "counts.bin")
	treePath := path.Join(filepath, "bwttree.bin")

	err := saveWaveletTree(bw.tree, treePath)
	if err != nil {
		return err
	}

	countBytes := make([]byte, 256*8)
	for i, count := range bw.counts {
		binary.LittleEndian.PutUint64(countBytes[i*8:], uint64(count))
	}

	err = writeBytesToFile(countsPath, countBytes)
	if err != nil {
		return err
	}

	return nil
}

func loadFMIndex(filepath string, vocabSize int) (*FMIndexModel, error) {
	countsPath := path.Join(filepath, "counts.bin")
	treePath := path.Join(filepath, "bwttree.bin")

	wt, err := loadWaveletTree(treePath)
	if err != nil {
		return nil, err
	}

	countsFile, err := os.Open(countsPath)
	if err != nil {
		return nil, err
	}
	defer countsFile.Close()

	reader := bufio.NewReader(countsFile)
	counts := [256]int64{}
	for i := 0; i < 256; i++ {
		err = binary.Read(reader, binary.LittleEndian, &counts[i])
		if err != nil {
			return nil, err
		}
	}

	return &FMIndexModel{wt, counts, vocabSize}, nil
}

// Will return the prediction of the next token distribution corresponding to the
// longest suffix in queryIds. For a suffix to be considered valid, there must be
// at least minMatches occurrences of it in the data. The retrieved suffixes will
// include numExtend extra tokens (set to 1 to just get the next token).
// TODO: Only numExtend = 1 is implemented for now.
func (m *FMIndexModel) NextTokenDistribution(queryIds []uint32, numExtend int, minMatches int) *Prediction {
	// copy and append
	newQueryIds := make([]uint32, len(queryIds)+1)
	copy(newQueryIds, queryIds)
	replIdx := len(queryIds)

	effectiveNBytes, longestCount := m.GetLongestSuffix(intToByte(newQueryIds[:replIdx]))
	fmt.Printf("longest suffix size=%d, count=%d\n", effectiveNBytes/2, longestCount)

	rawSuffixes := make([][]int, 0, longestCount)
	distr := make([]float32, m.vocabSize)
	runningTotal := uint64(0)
	for nextToken := uint32(0); nextToken < uint32(m.vocabSize); nextToken++ {
		// fmt.Println("testing query:", newQueryIds)
		newQueryIds[replIdx] = nextToken

		querySuffixEnc := intToByte(newQueryIds)

		suffixSize, count := getLongestSuffix(querySuffixEnc, m.counts, m.tree, minMatches)

		if suffixSize == (effectiveNBytes + 2) {
			// fmt.Println("found suffix:", querySuffixEnc, "count:", count, "size:", suffixSize)
			distr[nextToken] = float32(count) / float32(longestCount)
			runningTotal += count
			rawSuffixes = append(rawSuffixes, []int{int(nextToken)})
		}

		if runningTotal == longestCount {
			break
		}
	}

	return &Prediction{distr, effectiveNBytes / 2, int(longestCount), 1, rawSuffixes}
}

func InitializeFMIndexModel(filename, lineSplit, outpath, tokenizerConfig string, sentinalVal, sentinalSize, nWorkers, vocabSize, chunkSize int) (*FMIndexModel, error) {
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

	fmt.Println("Creating suffix array")
	fmt.Println("WARNING: will only use the first chunk; multiple chunks not implemented yet")
	currChunk := 0
	chunkBuffer := make([]byte, chunkSize)
	var fmIndex *FMIndexModel
	saCallback := func(chunkLength int) error {
		if currChunk > 0 {
			return errors.New("multiple chunks not implemented yet")
		}
		fmt.Printf("making chunk %d of size %d\n", currChunk, chunkLength)

		readValues := chunkBuffer[:chunkLength]

		unalignedSa := createUnalignedSuffixArray(readValues)

		fmIndex = makeFMIndex(unalignedSa, readValues, vocabSize)

		currChunk += 1

		return nil
	}
	err = documentIter(dataPath, sentinalSize, sentinalVal, chunkBuffer, saCallback)
	if err != nil {
		return nil, err
	}

	return fmIndex, nil
}
