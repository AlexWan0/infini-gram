package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"infinigram/tokenizers"
)

type Model interface {
	NextTokenDistribution(queryIds []uint32, numExtend int, minMatches int) *Prediction
}

// Wrapper around infini-gram model predictions results.
type Prediction struct {
	distribution      []float32 // next-token probability distribution
	effectiveN        int       // length of the longest suffix used
	numRetrieved      int       // number of continuations retrieved
	numExtend         int       // in the retrievedSuffixes, number of additional tokens added
	retrievedSuffixes [][]int   // raw retrieved suffixes
}

// Will generate a sequence of numNewTokens tokens greedily using the longest matched
// suffix. For a suffix to be considered valid, there must be at least minMatches
// occurrences of it in the data. queryIds are the initial prompt tokens.
func GenerateGreedy(m Model, queryIds []uint32, numNewTokens, minMatches int) []uint32 {
	result := make([]uint32, 0, len(queryIds)+numNewTokens)
	result = append(result, queryIds...)

	for i := 0; i < numNewTokens; i++ {
		prediction := m.NextTokenDistribution(result, 1, minMatches)

		if prediction.numRetrieved == 0 {
			return result
		}

		newToken := uint32(argmax(prediction.distribution))
		result = append(result, newToken)
	}

	return result
}

// Same as GenerateGreedy, but will send intermediate results to the generatedTokens
func GenerateGreedyStream(m Model, queryIds []uint32, numNewTokens, minMatches int, generatedTokens chan<- []uint32) {
	defer close(generatedTokens)

	result := make([]uint32, 0, len(queryIds)+numNewTokens)
	result = append(result, queryIds...)

	for i := 0; i < numNewTokens; i++ {
		prediction := m.NextTokenDistribution(result, 1, minMatches)

		if prediction.numRetrieved == 0 {
			return
		}

		newToken := uint32(argmax(prediction.distribution))
		result = append(result, newToken)

		generatedTokens <- result
	}
}

// Given a sequence of tokens (queryIds) will print the top-k most likely continuations using
// the longest possible suffix. The suffix must have at least minMatches occurrences in the data.
// modelData and tk are the model and tokenizer, respectively.
func InteractiveNextToken(queryIds []uint32, m Model, tk *tokenizers.Tokenizer, top_k, minMatches int) {
	prediction := m.NextTokenDistribution(queryIds, 1, minMatches)

	if prediction.numRetrieved == 0 {
		fmt.Println("No continuations found")
		return
	}

	topIndices := intToUint32(argsort(prediction.distribution, true))
	if len(topIndices) > top_k {
		topIndices = topIndices[:top_k]
	}

	fullGeneration := append([]uint32{}, queryIds...)
	fullGeneration = append(fullGeneration, 0)
	for i, tkn_idx := range topIndices {
		prob := prediction.distribution[tkn_idx]

		fullGeneration[len(fullGeneration)-1] = tkn_idx

		total := prediction.numRetrieved
		fmt.Printf(
			"n=%d, p=%.3f (%d/%d), k=%d: %s\n",
			prediction.effectiveN,
			prob,
			int(prob*float32(total)),
			total,
			i,
			tk.Decode(fullGeneration, true),
		)
	}
}

// Given a sequence of tokens (queryIds) will greedily generate numNewTokens tokens using the
// longest possible suffix. The suffix must have at least minMatches occurrences in the data.
// modelData and tk are the model and tokenizer, respectively.
func InteractiveGenerateGreedy(queryIds []uint32, m Model, tk *tokenizers.Tokenizer, numNewTokens, minMatches int) {
	generated_tokens := make(chan []uint32, 8)

	go GenerateGreedyStream(m, queryIds, numNewTokens, minMatches, generated_tokens)

	for tkns := range generated_tokens {
		fmt.Printf("====\n%s\n", tk.Decode(tkns, true))
	}
}

func main() {
	var _ = fmt.Printf

	var (
		filename        string
		outpath         string
		nWorkers        int
		tokenizerConfig string
		sentinalVal     int
		sentinalSize    int
		minMatches      int
		topK            int
		interactiveMode int
		numGenerate     int
		lineSplit       string
		maxMem          int
		useFMIndex      bool
	)

	flag.StringVar(&filename, "train_file", "", "Path to training data")
	flag.StringVar(&lineSplit, "line_split", "\n", "String to split documents in training data file")
	flag.StringVar(&outpath, "out_dir", "", "Directory to save trained model")
	flag.IntVar(&nWorkers, "n_workers", 4, "Number of workers to use")
	flag.StringVar(&tokenizerConfig, "tokenizer_config", "tokenizer_gpt2.json", "Path to .json file containing tokenizer configuration")
	flag.IntVar(&sentinalVal, "sentinal_val", 0, "Value to add at the end of every document")
	flag.IntVar(&sentinalSize, "sentinal_size", 2, "Number of sentinals to add at the end of every document")
	flag.IntVar(&minMatches, "min_matches", 1, "Minimum number of continuations needed for suffix to be valid")

	flag.IntVar(&maxMem, "max_mem", 1024, "Maximum size (in MiB) of documents for each chunk")

	flag.IntVar(&interactiveMode, "interactive_mode", 0, "0: print the top-k best next-token continuations 1: greedily generate k tokens")
	flag.IntVar(&topK, "top_k", 8, "Number of most frequent continuations to print during interactive mode 0")
	flag.IntVar(&numGenerate, "num_generate", 32, "Number of new tokens to generate")

	flag.BoolVar(&useFMIndex, "use_fm", false, "Use FM-index instead of suffix array")

	flag.Parse()

	// load tokenizer
	tk, err := tokenizers.FromFile(tokenizerConfig)
	if err != nil {
		panic(err)
	}

	defer tk.Close()

	var modelDataP Model
	if useFMIndex {
		fmt.Println("Using FM-index")
		modelDataP, err = InitializeFMIndexModel(
			filename,
			lineSplit,
			outpath,
			tokenizerConfig,
			sentinalVal,
			sentinalSize,
			nWorkers,
			int(tk.VocabSize()),
			maxMem*1024*1024,
		)
	} else {
		fmt.Println("Using suffix array")
		modelDataP, err = InitializeSuffixArrayModel(
			filename,
			lineSplit,
			outpath,
			tokenizerConfig,
			sentinalVal,
			sentinalSize,
			nWorkers,
			int(tk.VocabSize()),
			maxMem*1024*1024,
		)
	}
	if err != nil {
		panic(err)
	}

	modelData := modelDataP

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("enter query: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSuffix(input, "\n")
		input = strings.TrimSuffix(input, "\r")

		en, _ := tk.Encode(input, false)

		fmt.Println("encoded tokens:", en)

		if interactiveMode == 0 {
			InteractiveNextToken(en, modelData, tk, topK, minMatches)
		} else if interactiveMode == 1 {
			InteractiveGenerateGreedy(en, modelData, tk, numGenerate, minMatches)
		}
	}
}
