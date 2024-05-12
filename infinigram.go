package main

import (
	"path"
	"bufio"
	"os"
	"fmt"
	"strings"
	"flag"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	"github.com/schollz/progressbar/v3"
)

type ModelData struct {
	suffix_array []int64
	bytes_data []byte
}

func make_folder(folder_path string) error {
	info, err := os.Stat(folder_path)
	if err == nil {
		if info.IsDir() {
			return nil
		}
	}
	
	err = os.Mkdir(folder_path, 0755)
	if err != nil {
		return err
	}

	return nil
}

// func (m *ModelData) SaveSuffixArray(folderPath string) error {
//     err := make_folder(folderPath)
//     if err != nil {
//         return err
//     }

//     saPath := path.Join(folderPath, "suffix_array.bin")
//     fmt.Println("Saving suffix array")
//     err = WriteToFile(saPath, m.suffix_array)
//     if err != nil {
//         return err
//     }

//     return nil
// }

// func (m *ModelData) SaveTokenizedData(folderPath string) error {
//     err := make_folder(folderPath)
//     if err != nil {
//         return err
//     }

//     dataPath := path.Join(folderPath, "data.bin")
//     fmt.Println("Saving tokenized data")
//     err = WriteToFile(dataPath, m.bytes_data)
//     if err != nil {
//         return err
//     }

//     return nil
// }

func Load(folder_path string) (*ModelData, error) {
	sa_path := path.Join(folder_path, "suffix_array.bin")
	data_path := path.Join(folder_path, "data.bin")

	suffix_array, err := ReadInt64FromFile(sa_path)
	if err != nil {
		return nil, err
	}

	data_bytes, err_2 := ReadBytesFromFile(data_path)
	if err_2 != nil {
		return nil, err_2
	}

	return &ModelData{suffix_array: suffix_array, bytes_data: data_bytes}, nil
}


func is_all_whitespace(line string) bool {
	return strings.TrimSpace(line) == ""
}

func num_lines(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines, scanner.Err()
}

func read_and_tokenize(filename string, tk *tokenizer.Tokenizer, sentinal_val int, sentinal_size int) ([]byte, error) {
	// counts lines for progress bar
	file_num_lines, err := num_lines(filename)
	if err != nil {
		return nil, err
	}

	// read file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bar := progressbar.Default(int64(file_num_lines))

	// target
	data_bytes := make([]byte, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		bar.Add(1)

		line := scanner.Text()

		if is_all_whitespace(line) {
			continue
		}

		// tokenize
		en, err := tk.EncodeSingle(line)
		if err != nil {
			return nil, err
		}

		start_idx := len(data_bytes)
		end_idx := len(data_bytes) + (len(en.Ids) + sentinal_size) * 2

		// allocate space on this array first, maybe not the best way to do this
		for i := 0; i < (end_idx - start_idx); i++ {
			data_bytes = append(data_bytes, 0)
		}
		
		// bytes are placed in the slice for this particular document
		encode_sequence(data_bytes[start_idx : end_idx], en.Ids, sentinal_val, sentinal_size)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return data_bytes, nil
}

// func TrainFromPath(filename string, tk *tokenizer.Tokenizer, sentinal_val int, sentinal_size int) (*ModelData, error) {
// 	data_bytes, err := read_and_tokenize(filename, tk, sentinal_val, sentinal_size)
// 	if err != nil {
// 		return nil, err
// 	}

// 	suffix_array := create_suffix_array(data_bytes)

// 	return &ModelData{suffix_array: suffix_array, bytes_data: data_bytes}, nil
// }

type Prediction struct {
	distribution []float32
	effective_n int
	num_retrieved int
	num_extend int
	retrieved_suffixes [][]int
}

func (m *ModelData) next_token_distribution(query_ids []int, vocab_size int, num_extend int) *Prediction {
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
	
	model_data := ModelData{suffix_array: suffix_array, bytes_data: data_bytes}

	// // make model
	// model_data, err := TrainFromPath(filename, tk, sentinal_val, sentinal_size)
	// if err != nil {
	// 	panic(err)
	// }
	
	// // save to disk
	// err = model_data.Save(outpath)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("Succesfully saved to %s", outpath)

	// test loading
	// model_data_2, err := Load(outpath)
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(compare_slices(model_data.bytes_data, model_data_2.bytes_data))
	// fmt.Println(compare_slices_int64(model_data.suffix_array, model_data_2.suffix_array))

	// fmt.Println(suffix_array)

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

		prediction := model_data.next_token_distribution(en.Ids, tk.GetVocabSize(true), 1)

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
