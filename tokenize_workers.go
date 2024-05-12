package main

import (
	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	// "fmt"
	"github.com/schollz/progressbar/v3"
	"os"
	"bufio"
	"sync"
	"path"
	// "errors"
)

func init_tokenizer(tokenizer_config string) (*tokenizer.Tokenizer, error) {
	tk, err := pretrained.FromFile(tokenizer_config)
	return tk, err
}

func worker(wg *sync.WaitGroup, id int, tokenizer_config string, sentinel_val int, sentinel_size int, text_jobs <-chan string, results chan<- []byte) {
	defer wg.Done()
	
	tk, err := init_tokenizer(tokenizer_config)
	if err != nil {
		panic(err)
	}
	
	for text := range text_jobs {
		en, err := tk.EncodeSingle(text)

		if err != nil {
			panic(err)
		}

		data_bytes := make([]byte, (len(en.Ids) + sentinel_size) * 2)
		encode_sequence(data_bytes, en.Ids, sentinel_val, sentinel_size)

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

func tokenize_multiprocess(filename, outpath, tokenizer_config string, sentinel_val, sentinel_size, num_workers int) (string, error) {
	// Initialize output path
	if err := make_folder(outpath); err != nil {
		return "", err
	}
	sa_path := path.Join(outpath, "data.bin")

	// Count lines for the progress bar
	file_num_lines, err := num_lines(filename)
	if err != nil {
		return "", err
	}

	// Initialize workers
	text_jobs := make(chan string, num_workers*2)
	results := make(chan []byte, num_workers*2)

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
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		bar.Add(1)

		line := scanner.Text()

		if is_all_whitespace(line) {
			continue
		}

		text_jobs <- line
	}

	close(text_jobs)
	wg_workers.Wait()

	close(results)
	wg_writer.Wait()

	return sa_path, nil
}
