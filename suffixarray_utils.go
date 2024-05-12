package main

import (
	"infinigram/suffixarray"
)


func binary_search(suffix_array []int64, vec []byte, query []byte) (int64, int64) {
	query_len := int64(len(query))
	sa_len := int64(len(suffix_array))

	start := int64(0)
	end := sa_len
	for start < end {
		mid := int64((start + end) / 2)
		mid_slice := vec[suffix_array[mid] : suffix_array[mid] + query_len]

		cmp_value := compare_slices(mid_slice, query)

		if cmp_value < 0 {
			start = mid + 1
		} else {
			end = mid
		}
	}

	if start >= sa_len {
		return -1, -1
	}

	start_slice := vec[suffix_array[start] : suffix_array[start] + query_len]
	if (start == sa_len) || (compare_slices(start_slice, query) != 0) {
		return -1, -1
	}

	first_occ := start
	
	end = sa_len
	for start < end {
		mid := int64((start + end) / 2)
		mid_slice := vec[suffix_array[mid] : suffix_array[mid] + query_len]

		cmp_value := compare_slices(mid_slice, query)
		
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
	// filter to only include ones that match byte boundaries
	// return start positions of suffixes

	start_idx, end_idx := binary_search(suffix_array, vec, query)

	if (start_idx == -1) && (end_idx == -1) {
		return make([]int64, 0, 0)
	}

	suffix_starts := make([]int64, 0, end_idx - start_idx + 1)

	for s := start_idx; s <= end_idx; s++ {
		start_pos := suffix_array[s]
		if start_pos % 2 == 0 {
			suffix_starts = append(suffix_starts, start_pos)
		}
	}

	return suffix_starts
}

func retrieve_num(suffix_array []int64, vec []byte, query []byte) int {
	start_idx, end_idx := binary_search(suffix_array, vec, query)

	if (start_idx == -1) && (end_idx == -1) {
		return 0
	}

	num := 0

	for s := start_idx; s <= end_idx; s++ {
		start_pos := suffix_array[s]
		if start_pos % 2 == 0 {
			num += 1
		}
	}

	return num
}

func retrieve_substrings(suffix_array []int64, vec []byte, query []byte, extend int64) [][]byte {
	suffix_starts := retrieve(suffix_array, vec, query)

	n_result := len(suffix_starts)
	query_len := int64(len(query))

	result_slices := make([][]byte, n_result)
	for i, start := range suffix_starts {
		result_slices[i] = vec[start : start + query_len + (extend * 2)]
	}

	return result_slices
}

func encode_sequence(values_bytes []byte, values []int, sentinal_val int, sentinal_size int) {
	size := len(values)
	
	for i := 0; i < size; i++ {
		put_byte(values_bytes, uint16(values[i]), i)
	}

	for i := 0; i < sentinal_size; i ++ {
		put_byte(values_bytes, uint16(sentinal_val), size + i)
	}
}

func create_suffix_array(values_bytes []byte) []int64 {
	suffix_array := make([]int64, len(values_bytes))
	suffixarray.Text_64(values_bytes, suffix_array)
	
	return suffix_array
}

// func main() {
// 	min_val := int(math.Pow(2, 2))
// 	max_val := int(math.Pow(2, 4))
// 	length := 32

// 	sentinal_val := 0
// 	sentinal_size := 2

// 	// test_vec := rand_vector(length + 1, min_val, max_val)
// 	// test_vec[len(test_vec) - 1] = sentinal

// 	test_byte_vec := rand_vector_byte(length + sentinal_size, min_val, max_val)

// 	for i := 0; i < sentinal_size; i ++ {
// 		put_byte(test_byte_vec, uint16(sentinal_val), length + i)
// 	}

// 	fmt.Println(test_byte_vec)

// 	// index := suffixarray.New(test_byte_vec)
// 	// fmt.Println((*index))
// 	suffix_array := make([]int64, len(test_byte_vec))
// 	suffixarray.Text_64(test_byte_vec, suffix_array)

// 	for i, start_idx := range suffix_array {
// 		fmt.Println(i, test_byte_vec[start_idx:])
// 	}

// 	fmt.Println("------")

// 	test_query := make([]byte, 1 * 2)
// 	put_byte(test_query, uint16(10000), 0)
// 	// test_query := test_byte_vec[:2]

// 	// retrieved_suffixes := retrieve(suffix_array, test_byte_vec, test_query)
// 	// fmt.Println(retrieved_suffixes)

// 	// for _, s := range retrieved_suffixes {
// 	// 	fmt.Println(test_byte_vec[s:])
// 	// }

// 	retrieved_suffixes := retrieve_substrings(suffix_array, test_byte_vec, test_query, 2)
// 	fmt.Println(retrieved_suffixes)
// }
