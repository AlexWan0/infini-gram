package main

import (
	"fmt"
	"encoding/binary"
	"bufio"
	"os"
	"bytes"
)

func WriteToFile(filename string, value interface{}) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	switch v := value.(type) {
	case []byte:
		if _, err = f.Write(v); err != nil {
			return err
		}
	case []int64:
		length := int64(len(v))
		if err = binary.Write(f, binary.LittleEndian, length); err != nil {
			return err
		}
		if err = binary.Write(f, binary.LittleEndian, v); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
	return nil
}


func ReadBytesFromFile(filename string) ([]byte, error) {
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

func ReadInt64FromFile(filename string) ([]int64, error) {
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


func readDocuments(filename, line_split string, callback func(string) error) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	scanner.Split(splitAt(line_split))

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		err = callback(line)
		if err != nil {
			return err
		}
	}

	return nil
}

func num_lines(filename string, line_boundary string) (int, error) {
	counter := 0
	err := readDocuments(filename, line_boundary, func(line string) error {
		counter++
		return nil
	})
	return counter, err
}
