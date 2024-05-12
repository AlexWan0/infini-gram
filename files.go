package main

import (
	"fmt"
	"encoding/binary"
	// "io"
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
