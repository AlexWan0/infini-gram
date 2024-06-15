package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	wavelettree "github.com/AlexWan0/go-watrix"
	"github.com/schollz/progressbar/v3"
)

const (
	NUM_SYMBOLS     = 256 * 256
	TREE_FILENAME   = "bwttree.bin"
	COUNTS_FILENAME = "counts.bin"
	CACHE_FILENAME  = "cache.bin"
)

type FMIndexModel struct {
	tree      wavelettree.WaveletTree
	counts    [NUM_SYMBOLS]int64
	vocabSize int
	cache     TwoGramCache
}

// vec is expected to be little endian & the suffix array is expected to sort
// based on the little endian-encoded values. But, converting back to uin16
// would disrupt this ordering. So, we instead pretend that our data is big
// Endian and convert it back to little endian when we need to.
// In the current implementation, we don't even need to convert it back because
// we only query for the longest suffix size & its count.
// TODO: stream the result to disk
func saToBWT(sa SuffixArrayData, vec TokenArray) (wavelettree.WaveletTree, [NUM_SYMBOLS]int64, TwoGramCache) {
	builder := wavelettree.NewBuilder()

	symbCount := [NUM_SYMBOLS]int64{}

	cache := NewBitCache()

	fmt.Println("Constructing BWT and cache")
	bar := progressbar.Default(-1)
	for i := int64(0); i < sa.length(); i++ {
		suffixIdx := sa.get(i)
		if suffixIdx%2 == 1 {
			continue
		}

		// fmt.Println(counter, vec[suffixIdx:suffixIdx+2], binary.BigEndian.Uint16(vec[suffixIdx:suffixIdx+2]))

		// we have two ranges: suffixIdx - 2 to suffixIdx and suffixIdx to suffixIdx + 2
		// if this range is continuous, we can make a single call to getSlice
		// otherwise, we need to make two calls
		var backData []byte
		var frontData []byte
		backIdx := suffixIdx - 2
		if backIdx < 0 {
			backIdx += int64(vec.length())

			backData = vec.getSlice(backIdx, backIdx+2)
			frontData = vec.getSlice(suffixIdx, suffixIdx+2)
		} else {
			bothData := vec.getSlice(backIdx, backIdx+4)
			backData = bothData[:2]
			frontData = bothData[2:]
		}

		// vec[backIdx : backIdx+2]
		backSymbol16 := binary.BigEndian.Uint16(backData)
		builder.PushBack(uint64(backSymbol16))

		// fmt.Println(backSymbol16)

		frontSymbol16 := binary.BigEndian.Uint16(frontData)
		symbCount[frontSymbol16]++

		cache.AddValue(backSymbol16, frontSymbol16)

		bar.Add(1)
	}

	wt := builder.Build()

	return wt, symbCount, cache
}

func saveWaveletTree(wt wavelettree.WaveletTree, filename string) error {
	err := wt.MarshalBinaryFile(filename)
	if err != nil {
		return err
	}

	return nil
}

func loadWaveletTree(filename string) (wavelettree.WaveletTree, error) {
	wt := wavelettree.New()
	err := wt.UnmarshalBinaryFile(filename)
	if err != nil {
		return nil, err
	}

	return wt, nil
}

func getLongestSuffix(query16 []uint16, counts [NUM_SYMBOLS]int64, wt wavelettree.WaveletTree, minMatches int, cache TwoGramCache) (int, uint64) {
	countPrefixSum := make([]uint64, NUM_SYMBOLS) // number of symbols before i
	for i := 1; i < NUM_SYMBOLS; i++ {
		countPrefixSum[i] = countPrefixSum[i-1] + uint64(counts[i-1])
	}

	// fmt.Println("symbol prefix sum", countPrefixSum)

	// query16 := byteToInt(query)
	// fmt.Println("query encoded:", query16)

	query16FlipEnd := changeEndUint16Vec(query16)

	index := len(query16FlipEnd) - 1
	currChar := query16FlipEnd[index]
	if counts[currChar] == 0 {
		return 0, 0
	}

	start, end := countPrefixSum[currChar], countPrefixSum[currChar]+uint64(counts[currChar])
	count := end - start
	pastCount := count
	// fmt.Printf("[%d, %d] idx=%d, currChar=%d\n", start, end, index, currChar)
	longestSuffix := 1

	var followingChar uint16
	index--
	for count >= uint64(minMatches) && index >= 0 {
		followingChar = currChar

		// character we should look for in the prev position
		currChar = query16FlipEnd[index]

		// check cache
		if !cache.HasValue(currChar, followingChar) {
			break
		}

		// lookup counts
		allCounts := wt.Rank(end, uint64(currChar))
		if allCounts == 0 {
			break
		}
		prevCounts := wt.Rank(start, uint64(currChar))
		count = allCounts - prevCounts

		// if count > uint64(counts[currChar]) {
		// 	panic("count is greater than number of symbols")
		// }

		// we found a valid suffix, update
		if count >= uint64(minMatches) {
			longestSuffix++
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

func makeFMIndex(sa SuffixArrayData, vec TokenArray, vocabSize int) *FMIndexModel {
	// for rowIdx, idx := range sa {
	// 	fmt.Print(rowIdx, vec[idx:])
	// 	if idx > 0 {
	// 		fmt.Println("", vec[idx-1])
	// 	} else {
	// 		fmt.Println("", vec[len(vec)-1])
	// 	}
	// }

	wt, counts, cache := saToBWT(sa, vec)
	return &FMIndexModel{wt, counts, vocabSize, cache}
}

func (bw *FMIndexModel) GetLongestSuffix(query []uint16) (int, uint64) {
	return getLongestSuffix(query, bw.counts, bw.tree, 1, bw.cache)
}

func (bw *FMIndexModel) Save(filepath string) error {
	fmt.Println("Saving FMIndex to", filepath)

	countsPath := path.Join(filepath, COUNTS_FILENAME)
	treePath := path.Join(filepath, TREE_FILENAME)
	cachePath := path.Join(filepath, CACHE_FILENAME)

	err := saveWaveletTree(bw.tree, treePath)
	if err != nil {
		return err
	}

	countBytes := make([]byte, NUM_SYMBOLS*8)
	for i, count := range bw.counts {
		binary.LittleEndian.PutUint64(countBytes[i*8:], uint64(count))
	}

	err = writeBytesToFile(countsPath, countBytes)
	if err != nil {
		return err
	}

	err = bw.cache.Save(cachePath)
	if err != nil {
		return err
	}

	return nil
}

func FMExists(filepath string) bool {
	countsPath := path.Join(filepath, COUNTS_FILENAME)
	treePath := path.Join(filepath, TREE_FILENAME)
	cachePath := path.Join(filepath, CACHE_FILENAME)

	if _, err := os.Stat(countsPath); err != nil {
		return false
	}

	if _, err := os.Stat(treePath); err != nil {
		return false
	}

	if _, err := os.Stat(cachePath); err != nil {
		return false
	}

	return true
}

func loadFMIndex(filepath string, vocabSize int) (*FMIndexModel, error) {
	countsPath := path.Join(filepath, COUNTS_FILENAME)
	treePath := path.Join(filepath, TREE_FILENAME)
	cachePath := path.Join(filepath, CACHE_FILENAME)

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
	counts := [NUM_SYMBOLS]int64{}
	for i := 0; i < NUM_SYMBOLS; i++ {
		err = binary.Read(reader, binary.LittleEndian, &counts[i])
		if err != nil {
			return nil, err
		}
	}

	cache, err := LoadBitCache(cachePath)
	if err != nil {
		return nil, err
	}

	return &FMIndexModel{wt, counts, vocabSize, cache}, nil
}

type SuffixResult struct {
	suffixSize int
	count      uint64
	nextToken  uint16
}

func checkSuffixWorker(wg *sync.WaitGroup, m *FMIndexModel, minMatches int, queryIds []uint16, replIdx int, nextTokenJobs <-chan uint16, results chan<- *SuffixResult, quit <-chan bool) {
	defer wg.Done()

	queryIdsCopy := make([]uint16, len(queryIds))
	copy(queryIdsCopy, queryIds)
	for nextToken := range nextTokenJobs {
		select {
		case <-quit:
			return
		default:
			queryIdsCopy[replIdx] = nextToken
			suffixSize, count := getLongestSuffix(queryIdsCopy, m.counts, m.tree, minMatches, m.cache)
			results <- &SuffixResult{suffixSize, count, nextToken}
		}
	}
}

type AccumResult struct {
	distr       []float32
	rawSuffixes [][]int
}

func accumWorker(longestCount uint64, vocabSize int, effectiveN int, results <-chan *SuffixResult, accumResult chan<- *AccumResult, quit chan<- bool, numWorkers int) {
	rawSuffixes := make([][]int, 0, longestCount)
	distr := make([]float32, vocabSize)
	runningTotal := 0

	sentQuit := false
	for res := range results {
		if sentQuit {
			continue
		}

		if runningTotal >= int(longestCount) {
			for i := 0; i < numWorkers; i++ {
				quit <- true
			}
			sentQuit = true

			continue
		}

		if res.suffixSize == (effectiveN + 1) {
			distr[res.nextToken] += float32(res.count) / float32(longestCount)
			rawSuffixes = append(rawSuffixes, []int{int(res.nextToken)})

			runningTotal += int(res.count)
		}
	}

	accumResult <- &AccumResult{distr, rawSuffixes}
}

// Will return the prediction of the next token distribution corresponding to the
// longest suffix in queryIds. For a suffix to be considered valid, there must be
// at least minMatches occurrences of it in the data. The retrieved suffixes will
// include numExtend extra tokens (set to 1 to just get the next token).
// TODO: Only numExtend = 1 is implemented for now.
func (m *FMIndexModel) NextTokenDistribution(queryIds []uint32, numExtend int, minMatches int) *Prediction {
	if len(queryIds) == 0 {
		return &Prediction{[]float32{}, 0, 0, 1, [][]int{}}
	}

	// timing
	start := time.Now()

	// copy and append
	newQueryIds := make([]uint16, len(queryIds)+1)
	for i, x := range queryIds {
		newQueryIds[i] = uint16(x)
	}
	replIdx := len(queryIds)

	effectiveN, longestCount := getLongestSuffix(newQueryIds[:replIdx], m.counts, m.tree, minMatches, m.cache)
	fmt.Printf("longest suffix size=%d, count=%d\n", effectiveN, longestCount)

	// timing
	elapsed := time.Since(start)
	fmt.Printf("Find longest (n-1)-gram elapsed time: %s\n", elapsed)

	numWorkers := 8
	nextTokenJobs := make(chan uint16, numWorkers)
	results := make(chan *SuffixResult, numWorkers)
	quitChannel := make(chan bool, numWorkers+1)
	accumResult := make(chan *AccumResult)

	wgWorkers := &sync.WaitGroup{}

	for w := 0; w < numWorkers; w++ {
		wgWorkers.Add(1)
		go checkSuffixWorker(wgWorkers, m, minMatches, newQueryIds, replIdx, nextTokenJobs, results, quitChannel)
	}

	go accumWorker(longestCount, m.vocabSize, effectiveN, results, accumResult, quitChannel, numWorkers)

	lastTokenFlipped := changeEndUint16(newQueryIds[replIdx-1])
Outer:
	for nextToken := uint16(0); nextToken < uint16(m.vocabSize); nextToken++ {
		select {
		case <-quitChannel:
			break Outer
		default:
			if !m.cache.HasValue(lastTokenFlipped, changeEndUint16(nextToken)) {
				continue
			}
			nextTokenJobs <- nextToken
		}
	}

	close(nextTokenJobs)
	wgWorkers.Wait()

	close(results)
	accumRes := <-accumResult

	// timing
	elapsed = time.Since(start)
	fmt.Printf("Retrieval elapsed time: %s\n", elapsed)

	return &Prediction{accumRes.distr, effectiveN, int(longestCount), 1, accumRes.rawSuffixes}
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

	if FMExists(outpath) {
		fmt.Println("FMIndex already found; loading...")
		return loadFMIndex(outpath, vocabSize)
	}

	fmt.Println("FMIndex not found; creating new one")
	fmt.Println("WARNING: multiple chunks not implemented yet")

	var fmIndex *FMIndexModel

	// check whether suffix arrays already exist
	saChunkPathsPath := path.Join(outpath, "suffix_array_paths.txt")

	_, err = os.Stat(saChunkPathsPath)
	if err == nil {
		fmt.Println("Suffix array(s) already found")

		saChunkPathsStr, err := readStringFromFile(saChunkPathsPath)
		if err != nil {
			return nil, err
		}

		suffixArrayPaths := strings.Split(saChunkPathsStr, "\n")
		if len(suffixArrayPaths) > 1 {
			return nil, errors.New("multiple chunks not implemented yet")
		}

		suffixArray, err := makeMemSA(suffixArrayPaths[0])
		if err != nil {
			return nil, err
		}

		dataBytes, err := loadMMappedArray(dataPath)
		if err != nil {
			return nil, err
		}

		fmIndex = makeFMIndex(
			suffixArray,
			dataBytes,
			vocabSize,
		)

		fmIndex.Save(outpath)

		return fmIndex, nil
	}

	fmt.Println("Creating suffix array")
	currChunk := 0
	chunkBuffer := make([]byte, chunkSize)
	saCallback := func(chunkLength int) error {
		if currChunk > 0 {
			return errors.New("multiple chunks not implemented yet")
		}
		fmt.Printf("making chunk %d of size %d\n", currChunk, chunkLength)

		// need to the entire chunk into memory to make the suffix array
		readValues := chunkBuffer[:chunkLength]
		unalignedSa := createUnalignedSuffixArray(readValues)

		// but, we can use the mmapped data to create the FMindex
		dataBytes, err := loadMMappedArray(dataPath)
		if err != nil {
			return err
		}

		fmIndex = makeFMIndex(
			&MemSA{unalignedSa}, // could instead save to disk and MMap this
			dataBytes,
			vocabSize,
		)

		currChunk += 1

		return nil
	}
	err = documentIter(dataPath, sentinalSize, sentinalVal, chunkBuffer, saCallback)
	if err != nil {
		return nil, err
	}

	fmIndex.Save(outpath)

	return fmIndex, nil
}
