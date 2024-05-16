package main

import (
	// "github.com/sugarme/tokenizer"
	// "github.com/sugarme/tokenizer/pretrained"
	"github.com/tokenizers"
	"github.com/schollz/progressbar/v3"
	"os"
	"sync"
	"path"
	"strings"
)


func is_all_whitespace(line_p *string) bool {
	return strings.TrimSpace(*line_p) == ""
}

// func tokenize(filename, doc_split string, tk *tokenizers.Tokenizer, sentinal_val, sentinal_size int) ([]byte, error) {
// 	// counts lines for progress bar
// 	file_num_lines, err := num_lines(filename, doc_split)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// read file
// 	file, err := os.Open(filename)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	bar := progressbar.Default(int64(file_num_lines))

// 	// target
// 	data_bytes := make([]byte, 0)

// 	err = readDocuments(filename, doc_split, func(line_p *string) error {
// 		line := *line_p

// 		bar.Add(1)

// 		if !is_all_whitespace(line) {
// 			en, err := tk.EncodeSingle(line)
// 			if err != nil {
// 				return err
// 			}

// 			start_idx := len(data_bytes)
// 			end_idx := len(data_bytes) + (len(en.Ids) + sentinal_size) * 2

// 			// allocate space on this array first, maybe not the best way to do this
// 			for i := 0; i < (end_idx - start_idx); i++ {
// 				data_bytes = append(data_bytes, 0)
// 			}
			
// 			// bytes are placed in the slice for this particular document
// 			encode_sequence(data_bytes[start_idx : end_idx], en.Ids, sentinal_val, sentinal_size)
// 		}

// 		return nil
// 	})

// 	if err != nil {
// 		return nil, err
// 	}

// 	return data_bytes, nil
// }


func init_tokenizer(tokenizer_config string) (*tokenizers.Tokenizer, error) {
	tk, err := tokenizers.FromFile(tokenizer_config)
	return tk, err
}

func worker(wg *sync.WaitGroup, id int, tokenizer_config string, sentinel_val int, sentinel_size int, text_jobs <-chan *string, results chan<- []byte) {
	defer wg.Done()
	
	tk, err := init_tokenizer(tokenizer_config)
	if err != nil {
		panic(err)
	}
	defer tk.Close()
	
	for text_p := range text_jobs {
		en, _ := tk.Encode(*text_p, false)

		data_bytes := make([]byte, (len(en) + sentinel_size) * 2)
		encode_sequence(data_bytes, en, sentinel_val, sentinel_size)

		results <- data_bytes
	}
}

func write_worker(wg *sync.WaitGroup, filename string, results <-chan []byte) error {
	defer wg.Done()
	
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	for res := range results {
		if _, err := f.Write(res); err != nil {
			return err
		}
	}
	return nil
}

func tokenize_multiprocess(filename, doc_split, outpath, tokenizer_config string, sentinel_val, sentinel_size, num_workers int) (string, error) {
	// Initialize output path
	if err := make_folder(outpath); err != nil {
		return "", err
	}
	sa_path := path.Join(outpath, "data.bin")

	// Count lines for the progress bar
	file_num_lines, err := num_lines(filename, doc_split)
	if err != nil {
		return "", err
	}

	// Initialize workers
	text_jobs := make(chan *string, num_workers*4)
	results := make(chan []byte, num_workers*4)

	wg_workers := &sync.WaitGroup{}
	wg_writer := &sync.WaitGroup{}

	for w := 0; w < num_workers; w++ {
		wg_workers.Add(1)
		go worker(wg_workers, w, tokenizer_config, sentinel_val, sentinel_size, text_jobs, results)
	}

	wg_writer.Add(1)
	go write_worker(wg_writer, sa_path, results)

	// Read input file and enqueue lines for processing
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	bar := progressbar.Default(int64(file_num_lines))

	err = readDocuments(filename, doc_split, func(line_p *string) error {
		bar.Add(1)

		if !is_all_whitespace(line_p) {
			text_jobs <- line_p
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	close(text_jobs)
	wg_workers.Wait()

	close(results)
	wg_writer.Wait()

	return sa_path, nil
}
