package main

import (
	"fmt"

	"golang.org/x/exp/mmap"
)

type TokenArray interface {
	getSlice(int64, int64) []byte
	length() int64
}

type MemArray struct {
	data []byte
}

func (ma *MemArray) getSlice(start int64, end int64) []byte {
	return ma.data[start:end]
}

func (ma *MemArray) length() int64 {
	return int64(len(ma.data))
}

func loadMemArray(filepath string) (*MemArray, error) {
	dataBytes, err := readBytesFromFile(filepath)
	if err != nil {
		return nil, err
	}
	return &MemArray{data: dataBytes}, nil
}

type MMappedArray struct {
	mReader *mmap.ReaderAt
}

func loadMMappedArray(filepath string) (*MMappedArray, error) {
	mReader, err := mmap.Open(filepath)
	if err != nil {
		return nil, err
	}
	return &MMappedArray{mReader: mReader}, nil
}

func (ma *MMappedArray) getSlice(start int64, end int64) []byte {
	dest := make([]byte, end-start)

	_, err := ma.mReader.ReadAt(dest, start)

	if err != nil {
		// the regular array also throws an error if EOF, so that
		// behavior is matched here
		if err.Error() == "EOF" {
			// this panic lets us print the out of bounds index
			panic(fmt.Sprintf("[:%d] is out of bounds", end))
		} else {
			// will also panic if start is out of bounds
			// and will print the index
			panic(err)
		}
	}

	return dest
}

func (ma *MMappedArray) length() int64 {
	return int64(ma.mReader.Len())
}
