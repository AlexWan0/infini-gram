package main

import (
	"path"
	"bufio"
	"os"
	"fmt"
	"strings"
	"flag"

	"github.com/sugarme/tokenizer/pretrained"
)

type ModelData struct {
	suffix_array []int64
	bytes_data []byte
	vocab_size int
}


type Prediction struct {
	distribution []float32
	effective_n int
	num_retrieved int
	num_extend int
	retrieved_suffixes [][]int
}

func (m *ModelData) next_token_distribution(query_ids []int, num_extend int) *Prediction {
	vocab_size := m.vocab_size
	suffix_array := m.suffix_array
	data_bytes := m.bytes_data

	for query_start := 0; query_start < len(query_ids); query_start++ {
		query_ids_suffix := query_ids[query_start:]

		query_suffix_enc := int_to_byte(query_ids_suffix)

		substrings := retrieve_substrings(suffix_array, data_bytes, query_suffix_enc, int64(num_extend))

		if len(substrings) == 0 {
			continue
		}

		raw_suffixes := make([][]int, len(substrings))
		distr := make([]float32, vocab_size)
		total := 0
		for i, s := range substrings {
			retrieved_suffix := byte_to_int(s)
			
			new_ids := retrieved_suffix[len(retrieved_suffix) - num_extend:]
			new_ids = append([]int{}, new_ids...)
			
			// add to raw retrievals
			raw_suffixes[i] = new_ids

			// populate distribution
			distr[new_ids[0]] += 1
			total += 1
		}

		for i, _ := range distr {
			distr[i] /= float32(total)
		}

		return &Prediction{distr, len(query_ids_suffix), total, num_extend, raw_suffixes}
	}

	return &Prediction{nil, -1, 0, num_extend, make([][]int, 0)}
}

func main() {
	var _ = fmt.Printf

	var (
		filename string
		outpath  string
		n_workers int
		tokenizer_config string
		sentinal_val int
		sentinal_size int
		top_k int
	)
	
	flag.StringVar(&filename, "train_file", "", "Path to training data")
	flag.StringVar(&outpath, "out_dir", "", "Directory to save trained model")
	flag.IntVar(&n_workers, "n_workers", 4, "Number of workers to use")
	flag.StringVar(&tokenizer_config, "tokenizer_config", "tokenizer_gpt2.json", "Path to .json file containing tokenizer configuration")
	flag.IntVar(&sentinal_val, "sentinal_val", 0, "Value to add at the end of every document")
	flag.IntVar(&sentinal_size, "sentinal_size", 2, "Number of sentinals to add at the end of every document")
	flag.IntVar(&top_k, "top_k", 8, "Number of most frequent continuations to print during interactive mode")

	flag.Parse()

	// load tokenizer
	tk, err := pretrained.FromFile(tokenizer_config)
	if err != nil {
		panic(err)
	}

	// check whether tokenized data already exists
	data_path := path.Join(outpath, "data.bin")
	_, err = os.Stat(data_path)
	if err != nil {
		// tokenize data: streams documents from text file into binary file
		fmt.Println("Tokenizing data to disk")
		_, err := tokenize_multiprocess(filename, outpath, tokenizer_config, sentinal_val, sentinal_size, n_workers)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("Tokenized data already found")
	}

	// load *entire* binary file into memory
	data_bytes, err := ReadBytesFromFile(data_path)
	if err != nil {
		panic(err)
	}
	
	// load suffix array
	sa_path := path.Join(outpath, "suffix_array.bin")
	_, err = os.Stat(sa_path)
	var suffix_array []int64
	if err != nil {
		fmt.Println("Building suffix array")
		suffix_array = create_suffix_array(data_bytes)

		fmt.Println("Saving suffix array to disk")
		err = WriteToFile(sa_path, suffix_array)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("Suffix array already found, loading from disk")
		suffix_array, err = ReadInt64FromFile(sa_path)
		if err != nil {
			panic(err)
		}
	}
	
	model_data := ModelData{suffix_array: suffix_array, bytes_data: data_bytes, vocab_size: tk.GetVocabSize(true)}

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

		en, err := tk.EncodeSingle(input)
		if err != nil {
			panic(err)
		}

		prediction := model_data.next_token_distribution(en.Ids, 1)

		if prediction.num_retrieved == 0 {
			fmt.Println("No continuations found")
			continue
		}

		top_indices := argsort(prediction.distribution, true)
		if len(top_indices) > top_k{
			top_indices = top_indices[:top_k]
		}

		full_generation := append([]int{}, en.Ids...)
		full_generation = append(full_generation, 0)
		for i, tkn_idx := range top_indices {
			prob := prediction.distribution[tkn_idx]

			full_generation[len(full_generation) - 1] = tkn_idx

			total := prediction.num_retrieved
			fmt.Printf(
				"n=%d, p=%.3f (%d/%d), k=%d: %s\n",
				prediction.effective_n,
				prob,
				int(prob * float32(total)),
				total,
				i,
				tk.Decode(full_generation, true),
			)
		}

		// for _, next_ids := range prediction.retrieved_suffixes {
		// 	full_generation := append([]int{}, en.Ids...)
		// 	full_generation = append(full_generation, next_ids...)
		// 	// fmt.Println(prediction.effective_n, "-", tk.Decode(full_generation, true)
		// }
	}
}
