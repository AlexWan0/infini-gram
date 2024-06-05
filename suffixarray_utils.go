package main

import (
	"infinigram/suffixarray"
)

func arraySearch(suffix_array []int64, vec []byte, query []byte) (int64, int64) {
	query_len := int64(len(query))
	sa_len := int64(len(suffix_array))
	vec_len := int64(len(vec))

	start := int64(0)
	end := sa_len
	for start < end {
		mid := int64((start + end) / 2)
		mid_slice := vec[suffix_array[mid]:min(suffix_array[mid]+query_len, vec_len)]

		cmp_value := compareSlices(mid_slice, query)

		if cmp_value < 0 {
			start = mid + 1
		} else {
			end = mid
		}
	}

	if start >= sa_len {
		return -1, -1
	}

	start_slice := vec[suffix_array[start]:min(suffix_array[start]+query_len, vec_len)]
	if (start == sa_len) || (compareSlices(start_slice, query) != 0) {
		return -1, -1
	}

	first_occ := start

	end = sa_len
	for start < end {
		mid := int64((start + end) / 2)
		mid_slice := vec[suffix_array[mid]:min(suffix_array[mid]+query_len, vec_len)]

		cmp_value := compareSlices(mid_slice, query)

		if (cmp_value == 0) || (cmp_value == -1) {
			start = mid + 1
		} else {
			end = mid
		}
	}

	last_occ := start - 1

	return first_occ, last_occ
}

func retrieve(suffix_array []int64, vec []byte, query []byte) []int64 {
	// use binary search to find matching prefixes
	// return start positions of suffixes

	start_idx, end_idx := arraySearch(suffix_array, vec, query)

	if (start_idx == -1) && (end_idx == -1) {
		return make([]int64, 0)
	}

	suffix_starts := make([]int64, 0, end_idx-start_idx+1)

	for s := start_idx; s <= end_idx; s++ {
		start_pos := suffix_array[s]
		suffix_starts = append(suffix_starts, start_pos)
	}

	return suffix_starts
}

func retrieveNum(suffix_array []int64, vec []byte, query []byte) int {
	start_idx, end_idx := arraySearch(suffix_array, vec, query)

	if (start_idx == -1) && (end_idx == -1) {
		return 0
	}

	return int(end_idx - start_idx + 1)
}

func retrieveSubstrings(suffix_array []int64, vec []byte, query []byte, extend int64) [][]byte {
	suffix_starts := retrieve(suffix_array, vec, query)

	n_result := len(suffix_starts)
	query_len := int64(len(query))

	result_slices := make([][]byte, n_result)
	for i, start := range suffix_starts {
		result_slices[i] = vec[start : start+query_len+(extend*2)]
	}

	return result_slices
}

func encodeSequence(values_bytes []byte, values []uint32, sentinal_val int, sentinal_size int) {
	size := len(values)

	for i := 0; i < size; i++ {
		putByte(values_bytes, uint16(values[i]), i)
	}

	for i := 0; i < sentinal_size; i++ {
		putByte(values_bytes, uint16(sentinal_val), size+i)
	}
}

func createSuffixArray(values_bytes []byte) []int64 {
	suffix_array := make([]int64, len(values_bytes))
	suffixarray.Text_64(values_bytes, suffix_array)

	return suffix_array
}
