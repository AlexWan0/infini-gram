package main

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/exp/mmap"
)

const (
	MMAP_INT_WIDTH = 8
)

// Wraps the suffix array.
type SuffixArrayData interface {
	get(index int64) int64
	length() int64
}

// Loads the entire suffix array into memory.
type MemSA struct {
	data []int64
}

func (msa *MemSA) get(idx int64) int64 {
	return msa.data[idx]
}

func (msa *MemSA) length() int64 {
	return int64(len(msa.data))
}

func makeMemSA(filepath string) (*MemSA, error) {
	fmt.Println("loading suffix array from", filepath)

	data, err := readInt64FromFile(filepath)
	if err != nil {
		return nil, err
	}

	return &MemSA{data: data}, nil
}

// Access the suffix array from a memory-mapped file.
type MMappedSA struct {
	mReader *mmap.ReaderAt
}

func (msa *MMappedSA) get(idx int64) int64 {
	dest := make([]byte, 8)

	_, err := msa.mReader.ReadAt(dest[:MMAP_INT_WIDTH], idx*MMAP_INT_WIDTH)

	if err != nil {
		// the regular array also throws an error if EOF, so that
		// behavior is matched here
		if err.Error() == "EOF" {
			// this panic lets us print the out of bounds index
			panic(fmt.Sprintf("%d is out of bounds", idx))
		} else {
			// will also panic if start is out of bounds
			// and will print the index
			panic(err)
		}
	}

	return int64(binary.LittleEndian.Uint64(dest))
}

func (msa *MMappedSA) length() int64 {
	// fmt.Println("length of suffix array is", msa.mReader.Len())

	lengthBytes := int64(msa.mReader.Len())

	if lengthBytes%MMAP_INT_WIDTH != 0 {
		panic(fmt.Sprintf("suffix array length is not a multiple of %d", MMAP_INT_WIDTH))
	}

	return lengthBytes / MMAP_INT_WIDTH
}

func makeMMappedSA(filepath string) (*MMappedSA, error) {
	mReader, err := mmap.Open(filepath)
	if err != nil {
		return nil, err
	}
	return &MMappedSA{mReader: mReader}, nil
}
