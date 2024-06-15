package main

import (
	"bytes"
	"compress/gzip"
	"errors"

	bitarray "github.com/Workiva/go-datastructures/bitarray"
)

const (
	TWOGRAM_CACHE_SIZE = 256 * 256 * 256 * 256
)

type TwoGramCache interface {
	HasValue(first, second uint16) bool
	AddValue(first, second uint16)
	Save(filepath string) error
}

type BitCache struct {
	cache bitarray.BitArray
}

func NewBitCache() *BitCache {
	cache := bitarray.NewBitArray(TWOGRAM_CACHE_SIZE)
	return &BitCache{cache: cache}
}

func getIndex(first, second uint16) uint64 {
	return uint64(first)<<16 + uint64(second)
}

// Should never be out of range as we set the size in advance.
func (bc *BitCache) HasValue(first, second uint16) bool {
	index := getIndex(first, second)
	result, err := bc.cache.GetBit(index) // panics if out of range
	if err != nil {
		panic(err)
	}
	return result
}

// Should never be out of range as we set the size in advance.
func (bc *BitCache) AddValue(first, second uint16) {
	index := getIndex(first, second)
	err := bc.cache.SetBit(index) // panics if out of range
	if err != nil {
		panic(err)
	}
}

func copyToBitArray(source, dest bitarray.BitArray) error {
	for i := uint64(0); i < source.Capacity(); i++ {
		isSet, err := source.GetBit(i)
		if err != nil {
			return err
		}
		if isSet {
			err = dest.SetBit(i)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (bc *BitCache) Save(filepath string) error {
	// convert to sparse array first
	// sparseArray := bitarray.NewSparseBitArray()
	// err := copyToBitArray(bc.cache, sparseArray)
	// if err != nil {
	// 	return err
	// }

	cacheBytes, err := bitarray.Marshal(bc.cache)
	if err != nil {
		return err
	}

	// compress
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(cacheBytes)
	w.Close()

	err = writeBytesToFile(filepath, b.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func LoadBitCache(filepath string) (*BitCache, error) {
	// read the sparse array
	cacheBytes, err := readBytesFromFile(filepath)
	if err != nil {
		return nil, err
	}

	// spraseArray, err := bitarray.Unmarshal(cacheBytes)
	// if err != nil {
	// 	return nil, err
	// }

	// decompress
	r, err := gzip.NewReader(bytes.NewReader(cacheBytes))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var b bytes.Buffer
	b.ReadFrom(r)

	// convert to dense array
	// cache := bitarray.NewBitArray(TWOGRAM_CACHE_SIZE)
	// err = copyToBitArray(spraseArray, cache)
	cache, err := bitarray.Unmarshal(b.Bytes())
	if err != nil {
		return nil, err
	}

	if cache.Capacity() != TWOGRAM_CACHE_SIZE {
		return nil, errors.New("cache size mismatch")
	}

	return &BitCache{cache: cache}, nil
}
