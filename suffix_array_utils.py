from pydivsufsort import divsufsort
import numpy as np


def build_suffix_array(s: np.array) -> np.array:
    return divsufsort(s)


def leq(x, y):
    for i in np.arange(x.size):
        if x[i] == y[i]:
            continue
        return x[i] < y[i]
    return True


def binary_search(sa, s, query):
    start, end = 0, len(sa)
    while start < end:
        mid = (start + end) // 2
        mid_slice = s[sa[mid]:sa[mid]+len(query)]

        if np.array_equal(mid_slice, query):
            cmp_val = 0
        elif leq(mid_slice, query):
            cmp_val = -1
        else:
            cmp_val = 1
        
        if cmp_val < 0:
            start = mid + 1
        else:
            end = mid
    if start == len(sa) or not np.array_equal(s[sa[start]:sa[start]+len(query)], query):
        return -1, -1
    first_occurrence = start

    end = len(sa)
    while start < end:
        mid = (start + end) // 2
        mid_slice = s[sa[mid]:sa[mid]+len(query)]
        if np.array_equal(mid_slice, query) or leq(mid_slice, query):
            start = mid + 1
        else:
            end = mid
    last_occurrence = start - 1
    return first_occurrence, last_occurrence


def retrieve_num_substrings(sa, s, query, extend=0):
    assert extend <= 1

    first, last = binary_search(sa, s, query)
    if first == -1:
        return 0, (None, None)
    
    return (last - first + 1), (first, last)


def get_retrieved_substrings(first, last, sa, s, query, extend=1):
    assert extend <= 1

    # maybe slow
    matching_substrings = []
    for i in range(first, last + 1):    
        start_index = sa[i]
        matching_substrings.append(s[start_index:start_index + len(query) + extend])

    return matching_substrings


def retrieve_substrings(sa, s, query, extend=1):
    assert extend <= 1

    num_matches, (first, last) = retrieve_num_substrings(sa, s, query, extend)
    
    if num_matches == 0:
        return []

    return get_retrieved_substrings(first, last, sa, s, query, extend)
