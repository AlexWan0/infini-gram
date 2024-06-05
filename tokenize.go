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

func isAllWhitespace(lineP *string) bool {
	return strings.TrimSpace(*lineP) == ""
}

func initTokenizer(tokenizerConfig string) (*tokenizers.Tokenizer, error) {
	tk, err := tokenizers.FromFile(tokenizerConfig)
	return tk, err
}

func worker(wg *sync.WaitGroup, tokenizerConfig string, sentinalVal int, sentinalSize int, textJobs <-chan *string, results chan<- []byte) {
	defer wg.Done()

	tk, err := initTokenizer(tokenizerConfig)
	if err != nil {
		panic(err)
	}
	defer tk.Close()

	for textP := range textJobs {
		en, _ := tk.Encode(*textP, false)

		dataBytes := make([]byte, (len(en)+sentinalSize)*2)
		encodeSequence(dataBytes, en, sentinalVal, sentinalSize)

		results <- dataBytes
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

func tokenizeMultiprocess(filename, docSplit, outpath, tokenizerConfig string, sentinalVal, sentinalSize, numWorkers int) (string, error) {
	// Initialize output path
	if err := makeFolder(outpath); err != nil {
		return "", err
	}
	saPath := path.Join(outpath, "data.bin")

	// Count lines for the progress bar
	fileNumLines, err := numLines(filename, docSplit)
	if err != nil {
		return "", err
	}

	fmt.Println("Num lines: ", fileNumLines)

	// Initialize workers
	textJobs := make(chan *string, numWorkers*4)
	results := make(chan []byte, numWorkers*4)

	wgWorkers := &sync.WaitGroup{}
	wgWriter := &sync.WaitGroup{}

	for w := 0; w < numWorkers; w++ {
		wgWorkers.Add(1)
		go worker(wgWorkers, tokenizerConfig, sentinalVal, sentinalSize, textJobs, results)
	}

	wgWriter.Add(1)
	go writeWorker(wgWriter, saPath, results)

	// Read input file and enqueue lines for processing
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	bar := progressbar.Default(int64(fileNumLines))

	err = readDocuments(filename, docSplit, func(lineP *string) error {
		bar.Add(1)

		if !isAllWhitespace(lineP) {
			textJobs <- lineP
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	close(textJobs)
	wgWorkers.Wait()

	close(results)
	wgWriter.Wait()

	return saPath, nil
}
