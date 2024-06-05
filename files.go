package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"os"
)

func writeIndicesToFile(filename string, indicesOut []int64) error {
	// writes the length followed by the values
	// only writes even values (indices that align with byte boundaries)

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
			if err = binary.Write(bufWriter, binary.LittleEndian, v); err != nil {
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

// https://stackoverflow.com/a/57232670
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
