package main

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
