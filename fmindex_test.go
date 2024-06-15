package main

import (
	"fmt"
	"infinigram/tokenizers"
	"testing"
)

func tokenize(text string) ([]uint16, int, error) {
	tk, err := tokenizers.FromFile("data/tokenizer_llama2.json")
	if err != nil {
		return nil, -1, err
	}
	defer tk.Close()

	en, _ := tk.Encode(text, false)

	fmt.Println("Text:", text)
	fmt.Println("Encoded:", en)

	en16 := make([]uint16, len(en))
	for i, v := range en {
		en16[i] = uint16(v)
	}

	return en16, int(tk.VocabSize()), nil
}

// func TestRun(t *testing.T) {
// 	vecPath := "./data/simpletest/data.bin"

// 	memVec, err := loadMemArray(vecPath)
// 	if err != nil {
// 		panic(err)
// 	}

// 	sa := createUnalignedSuffixArray(memVec.data)

// 	fmt.Println(len(sa))
// 	fmt.Println(len(memVec.data))

// 	// test bwt equal to lastChars
// 	bwt, counts := saToBWT(sa, memVec.data)

// 	fmt.Println(counts)
// 	fmt.Println(bwt)

// 	lastChars := make([]byte, 0)
// 	for rowIdx, idx := range sa {
// 		fmt.Print(rowIdx, memVec.data[idx:])
// 		if idx > 0 {
// 			fmt.Println("", memVec.data[idx-1])
// 			lastChars = append(lastChars, memVec.data[idx-1])
// 		} else {
// 			fmt.Println("", memVec.data[len(memVec.data)-1])
// 			lastChars = append(lastChars, memVec.data[len(memVec.data)-1])
// 		}
// 	}
// 	if !reflect.DeepEqual(bwt, lastChars) {
// 		fmt.Println("real bwt", lastChars)
// 		t.Fatalf("bwt not equal to lastChars")
// 	}

// 	// create wavelet tree
// 	wt := makeWaveletTree(bwt)

// 	// save and load wavelet tree
// 	err = saveWaveletTree(wt, "./data/simpletest/wt.bin")
// 	if err != nil {
// 		panic(err)
// 	}

// 	wt, err = loadWaveletTree("./data/simpletest/wt.bin")
// 	if err != nil {
// 		panic(err)
// 	}

// 	queryBytes, err := tokenize("hello world")
// 	if err != nil {
// 		panic(err)
// 	}

// 	// search for query
// 	longestSuffix := getLongestSuffix(queryBytes, counts, wt)

// 	fmt.Println("longestSuffix:", longestSuffix)
// }

func TestStruct(t *testing.T) {
	basePath := "./data/aaaa"
	vecPath := basePath + "/data.bin"

	fmt.Println("loading data")
	memVec, err := loadMemArray(vecPath)
	if err != nil {
		panic(err)
	}
	// fmt.Println(memVec.data)

	enc, vocabSize, err := tokenize("tree")
	if err != nil {
		panic(err)
	}

	fmt.Println("making suffix array")

	changeEndb16(memVec.data)
	sa := createUnalignedSuffixArray(memVec.data)
	memSA := &MemSA{data: sa}
	fmindex := makeFMIndex(memSA, memVec, vocabSize)

	fmt.Println(getLongestSuffix(enc, fmindex.counts, fmindex.tree, 1, fmindex.cache))

	// save and load
	// fmindex.Save(basePath)
	// fmindex, err = loadFMIndex(basePath, vocabSize)
	// if err != nil {
	// 	panic(err)
	// }

	// // test retrieval
	// longestSuffix, numOcc := fmindex.GetLongestSuffix(enc)
	// fmt.Println("longestSuffix:", longestSuffix)
	// fmt.Println("numOcc:", numOcc)

	// // test next token
	// tk, err := tokenizers.FromFile("data/tokenizer_llama2.json")
	// if err != nil {
	// 	panic(err)
	// }
	// defer tk.Close()

	// en, _ := tk.Encode("the", false)
	// fmt.Println("encoded tokens:", en)

	// InteractiveNextToken(en, fmindex, tk, 16, 1)
}
