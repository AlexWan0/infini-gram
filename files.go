package main

import (
	"fmt"
	"encoding/binary"
	"bufio"
	"os"
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
