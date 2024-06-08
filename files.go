package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// Writes the even indices into filename with offset added to each value.
// For writing the suffix array indices to disk: only the even indices align
// with byte boundaries.
// Only even indices are written as the tokens are in uint16 format, but the
// suffix array indices correspond to the location of the bytes.
// The offset is needed as suffix arrays are constructed in chunks. But the
// tokenized corpus is a continguous array.
func writeIndicesToFile(filename string, indicesOut []int64, offset int64) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// create a buffered writer
	bufWriter := bufio.NewWriter(f)

	// write placeholder value for the length
	if err = binary.Write(bufWriter, binary.LittleEndian, int64(0)); err != nil {
		return err
	}

	// write indices
	length := 0
	for _, v := range indicesOut {
		if v%2 == 0 { // if aligns with byte boundary
			if err = binary.Write(bufWriter, binary.LittleEndian, v+offset); err != nil {
				return err
			}
			length++
		}
	}

	// flush the buffer to write the data to the file
	if err = bufWriter.Flush(); err != nil {
		return err
	}

	// write the length
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err = binary.Write(f, binary.LittleEndian, int64(length)); err != nil {
		return err
	}

	return nil
}

func readBytesFromFile(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, info.Size())
	_, err = f.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func readInt64FromFile(filename string) ([]int64, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var length int64
	if err = binary.Read(f, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	int64s := make([]int64, length)
	if err = binary.Read(f, binary.LittleEndian, &int64s); err != nil {
		return nil, err
	}
	return int64s, nil
}

func makeFolder(folderPath string) error {
	info, err := os.Stat(folderPath)
	if err == nil {
		if info.IsDir() {
			return nil
		}
	}

	err = os.Mkdir(folderPath, 0755)
	if err != nil {
		return err
	}

	return nil
}

// Source: https://stackoverflow.com/a/57232670
func splitAt(substring string) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	searchBytes := []byte(substring)
	searchLen := len(searchBytes)
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		dataLen := len(data)

		// Return nothing if at end of file and no data passed
		if atEOF && dataLen == 0 {
			return 0, nil, nil
		}

		// Find next separator and return token
		if i := bytes.Index(data, searchBytes); i >= 0 {
			return i + searchLen, data[0:i], nil
		}

		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return dataLen, data, nil
		}

		// Request more data.
		return 0, nil, nil
	}
}

func readDocuments(filename, lineSplit string, callback func(*string) error) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	scanner.Split(splitAt(lineSplit))

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 16384*1024) // max doc size of ~16mb

	for scanner.Scan() {
		line := scanner.Text()

		err = callback(&line)
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func numLines(filename string, lineBoundary string) (int, error) {
	counter := 0
	err := readDocuments(filename, lineBoundary, func(lineP *string) error {
		counter++
		return nil
	})
	return counter, err
}

// Reads as many documents as possible (delineated by the sentinal) from filename into slice chunk.
// Will try to read as many documents as possible into chunk, then call callback
// with the length of the values read. The function callback is called everytime
// the chunk is full.
func documentIter(filename string, sentinalSize, sentinalValue int, chunk []byte, callback func(int) error) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	bufferSize := 1024 * 1024 // 1mb
	if len(chunk) < bufferSize {
		return errors.New("chunk smaller than read buffer")
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, bufferSize)

	chunkIdx := 0

	callbackReset := func(chunkLength int) error {
		lastSentinal := findLastSentinal(chunk, chunkLength, sentinalSize, sentinalValue)

		// sentinal not found which means that there's a document
		// larger than the chunk size
		if lastSentinal == -1 {
			return errors.New("chunk too small")
		}

		adjustedLength := lastSentinal + sentinalSize*2

		err := callback(adjustedLength)
		if err != nil {
			return err
		}

		// copy the remaining bytes to the start of the chunk
		copy(chunk, chunk[adjustedLength:chunkLength])

		chunkIdx = chunkLength - adjustedLength

		return nil
	}

	for {
		nread, err := reader.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				if chunkIdx == 0 {
					break
				}

				if !hasSentinal(chunk, chunkIdx, sentinalSize, sentinalValue) {
					return errors.New("file does not end with sentinal")
				}

				err := callbackReset(chunkIdx)
				if err != nil {
					return err
				}
			} else {
				return err
			}
			break
		}

		if chunkIdx+nread > cap(chunk) {
			err := callbackReset(chunkIdx)
			if err != nil {
				return err
			}
		}
		copy(chunk[chunkIdx:], buffer[:nread])
		chunkIdx += nread
	}

	return nil
}

func hasSentinal(values []byte, length, sentinalSize, sentinalValue int) bool {
	if length < sentinalSize*2 {
		return false
	}

	for j := 2; j <= sentinalSize*2; j += 2 {
		valInt := binary.LittleEndian.Uint16(values[length-j : length-(j-2)])
		if valInt != uint16(sentinalValue) {
			return false
		}
	}

	return true
}

func findLastSentinal(values []byte, length, sentinalSize, sentinalValue int) int {
	for i := length - sentinalSize*2; i >= 0; i -= 2 {
		found := true
		for j := 2; j <= sentinalSize*2; j += 2 {
			valInt := binary.LittleEndian.Uint16(values[i+j-2 : i+j])
			if valInt != uint16(sentinalValue) {
				found = false
				break
			}
		}

		if found {
			return i
		}
	}

	return -1
}

func writeStringToFile(filename string, data string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(data)
	if err != nil {
		return err
	}

	return nil
}

func readStringFromFile(filename string) (string, error) {
	dataBytes, err := readBytesFromFile(filename)
	if err != nil {
		return "", err
	}
	return string(dataBytes), nil
}
