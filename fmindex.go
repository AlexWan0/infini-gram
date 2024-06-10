package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"

	wavelettree "github.com/sekineh/go-watrix"
)

type FMIndex struct {
	tree   wavelettree.WaveletTree
	counts [256]int64
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

func getLongestSuffix(query []byte, counts [256]int64, wt wavelettree.WaveletTree) (int, uint64) {
	countPrefixSum := make([]uint64, 256) // number of symbols before i
	for i := 1; i < 256; i++ {
		countPrefixSum[i] = countPrefixSum[i-1] + uint64(counts[i-1])
	}

	fmt.Println("symbol prefix sum", countPrefixSum)

	index := len(query) - 1
	currChar := query[index]
	if counts[currChar] == 0 {
		return 0, 0
	}

	start, end := countPrefixSum[currChar], countPrefixSum[currChar]+uint64(counts[currChar])
	count := end - start
	pastCount := count
	fmt.Printf("[%d, %d] idx=%d, currChar=%d\n", start, end, index, currChar)

	index--
	for count > 0 && index >= 0 {
		currChar = query[index]

		prevCounts := wt.Rank(start, uint64(currChar))
		allCounts := wt.Rank(end+1, uint64(currChar))
		pastCount = count
		count = allCounts - prevCounts

		start = countPrefixSum[currChar] + prevCounts
		end = start + count
		fmt.Printf("[%d, %d] idx=%d, currChar=%d\n", start, end, index, currChar)

		index--
	}

	return len(query) - index - 1, pastCount
}

func makeFMIndex(sa []int64, vec []byte) *FMIndex {
	bwt, counts := saToBWT(sa, vec)
	wt := makeWaveletTree(bwt)
	return &FMIndex{wt, counts}
}

func (bw *FMIndex) GetLongestSuffix(query []byte) (int, uint64) {
	return getLongestSuffix(query, bw.counts, bw.tree)
}

func (bw *FMIndex) Save(filepath string) error {
	countsPath := filepath + "/counts.bin"
	treePath := filepath + "/bwttree.bin"

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

func loadFMIndex(filepath string) (*FMIndex, error) {
	countsPath := filepath + "/counts.bin"
	treePath := filepath + "/bwttree.bin"

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

	return &FMIndex{wt, counts}, nil
}
