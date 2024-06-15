package main

import (
	"fmt"

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

func (bc *BitCache) Save(filepath string) error {
	cacheBytes, err := bitarray.Marshal(bc.cache)
	if err != nil {
		return err
	}

	err = writeBytesToFile(filepath, cacheBytes)
	if err != nil {
		return err
	}

	return nil
}

func LoadBitCache(filepath string) (*BitCache, error) {
	cacheBytes, err := readBytesFromFile(filepath)
	if err != nil {
		return nil, err
	}

	cache, err := bitarray.Unmarshal(cacheBytes)
	if err != nil {
		return nil, err
	}

	if cache.Capacity() != TWOGRAM_CACHE_SIZE {
		return nil, fmt.Errorf("expected to load %d bits, got %d", TWOGRAM_CACHE_SIZE, cache.Capacity())
	}

	return &BitCache{cache: cache}, nil
}
