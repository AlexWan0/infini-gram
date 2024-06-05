package main

import (
	"fmt"
	"infinigram/tokenizers"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
)

func isAllWhitespace(line_p *string) bool {
	return strings.TrimSpace(*line_p) == ""
}

func initTokenizer(tokenizer_config string) (*tokenizers.Tokenizer, error) {
	tk, err := tokenizers.FromFile(tokenizer_config)
	return tk, err
}

func worker(wg *sync.WaitGroup, tokenizer_config string, sentinel_val int, sentinel_size int, text_jobs <-chan *string, results chan<- []byte) {
	defer wg.Done()

	tk, err := initTokenizer(tokenizer_config)
	if err != nil {
		panic(err)
	}
	defer tk.Close()

	for text_p := range text_jobs {
		en, _ := tk.Encode(*text_p, false)

		data_bytes := make([]byte, (len(en)+sentinel_size)*2)
		encodeSequence(data_bytes, en, sentinel_val, sentinel_size)

		results <- data_bytes
	}
}

func writeWorker(wg *sync.WaitGroup, filename string, results <-chan []byte) error {
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

func tokenizeMultiprocess(filename, doc_split, outpath, tokenizer_config string, sentinel_val, sentinel_size, num_workers int) (string, error) {
	// Initialize output path
	if err := makeFolder(outpath); err != nil {
		return "", err
	}
	sa_path := path.Join(outpath, "data.bin")

	// Count lines for the progress bar
	file_num_lines, err := numLines(filename, doc_split)
	if err != nil {
		return "", err
	}

	fmt.Println("Num lines: ", file_num_lines)

	// Initialize workers
	text_jobs := make(chan *string, num_workers*4)
	results := make(chan []byte, num_workers*4)

	wg_workers := &sync.WaitGroup{}
	wg_writer := &sync.WaitGroup{}

	for w := 0; w < num_workers; w++ {
		wg_workers.Add(1)
		go worker(wg_workers, tokenizer_config, sentinel_val, sentinel_size, text_jobs, results)
	}

	wg_writer.Add(1)
	go writeWorker(wg_writer, sa_path, results)

	// Read input file and enqueue lines for processing
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	bar := progressbar.Default(int64(file_num_lines))

	err = readDocuments(filename, doc_split, func(line_p *string) error {
		bar.Add(1)

		if !isAllWhitespace(line_p) {
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
