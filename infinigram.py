import numpy as np
import tqdm
from transformers import AutoTokenizer
from typing import Union
from transformers import PreTrainedTokenizerBase
import os
from tokenization import tokenize_single, tokenize, decode_single, decode, tokenize_mp
import multiprocessing as mp
from joblib import Memory

from predictions import NextTokenResult
from suffix_array_utils import build_suffix_array, retrieve_substrings, retrieve_num_substrings, get_retrieved_substrings


class InfiniGramModel:
    def __init__(
            self,
            documents_tkn: np.array,
            suffix_array: np.array,
            tokenizer: PreTrainedTokenizerBase,

            do_cache: bool=False,
            cache_dir: str='./data/.cache'
        ):
        '''
        Initialize model from preprocessed data.
        :param documents_tkn: Tokenized documents, concatenated and separated by eos token
        :param suffix_array: Suffix array of documents_tkn
        :param tokenizer: Tokenizer

        :param do_cache: Whether to cache results. Caches the substring lookups.
        :param cache_dir: Directory to store cache.
        '''

        self.documents_tkn = documents_tkn
        self.suffix_array = suffix_array
        self.tokenizer = tokenizer

        # caching
        if do_cache:
            print('Caching enabled')
            memory = Memory(cache_dir, verbose=0)
            self.get_matching_next = memory.cache(self.get_matching_next, ignore=['self'])

    @classmethod
    def from_pretrained(
            cls,
            path: str
        ):
        '''
        Load model from pretrained directory.
        :param path: Path to directory containing model files.
        '''

        documents_tkn = np.load(os.path.join(path, 'documents_tkn.npy'))
        suffix_array = np.load(os.path.join(path, 'suffix_array.npy'))
        tokenizer = AutoTokenizer.from_pretrained(path)

        return cls(documents_tkn, suffix_array, tokenizer)
    
    @classmethod
    def from_data(
            cls,
            documents: list[str],
            tokenizer: Union[str, PreTrainedTokenizerBase]='gpt2',
            nworkers: int=1
        ):
        '''
        Initialize model from data. Builds suffix array and tokenizes documents.
        :param documents: List of documents
        :param tokenizer: Tokenizer or string to load tokenizer from
        :param nworkers: Number of workers for tokenization
        '''
        
        assert nworkers > 0
        
        if isinstance(tokenizer, str):
            tokenizer = AutoTokenizer.from_pretrained(tokenizer)
        else:
            tokenizer = tokenizer
        
        print('SuffixArray: tokenizing documents')
        if nworkers == 1:
            _documents_tkn = tokenize(documents, tokenizer, verbose=True)
        else:
            _documents_tkn = tokenize_mp(documents, tokenizer, verbose=True, nworkers=nworkers)
        
        _documents_tkn = [
            np.concatenate([tkn, np.array([tokenizer.eos_token_id])]).astype(int) # sep with eos token
            for tkn in _documents_tkn
        ]
        documents_tkn = np.concatenate(_documents_tkn)

        print('SuffixArray: building suffix array')
        suffix_array = build_suffix_array(documents_tkn)

        return cls(documents_tkn, suffix_array, tokenizer)

    '''
    Tokenization methods: wraps around tokenization.py functions but using the model's tokenizer.
    '''
    def tokenize_single(
            self,
            string: str
        ) -> np.array:

        return tokenize_single(string, self.tokenizer)

    def tokenize(
            self,
            strings: list[str],
            verbose=False
        ) -> list[np.array]:
        
        return tokenize(strings, self.tokenizer, verbose)

    def decode_single(
            self,
            tkn: np.array
        ) -> str:
        
        return decode_single(tkn, self.tokenizer)
    
    def decode(
            self,
            tkns: list[np.array]
        ) -> list[str]:

        return decode(tkns, self.tokenizer)

    def save_pretrained(
            self,
            path: str
        ):
        
        if not os.path.isdir(path):
            os.mkdir(path)
        
        np.save(os.path.join(path, 'documents_tkn.npy'), self.documents_tkn)
        np.save(os.path.join(path, 'suffix_array.npy'), self.suffix_array)
        self.tokenizer.save_pretrained(path)

    def get_matching_next(
            self,
            query: np.array
        ) -> list[np.array]:
        '''
        Finds matching substrings plus one extra token.
        If the query doesn't end with an EOS token then you can always find a substring + one token.
        :param query: Substring to match.
        '''
        assert query[-1] != self.tokenizer.eos_token_id

        return retrieve_substrings(self.suffix_array, self.documents_tkn, query, extend=1)

    def get_longest_matching_next(
            self,
            query: np.array,
            min_count: int=1,
            verbose: bool=False
        ) -> tuple[list[np.array], int]:
        '''
        Finds the longest matching substring plus one extra token.
        :param query: Substring to match.
        :param min_count: Minimum number of matches to consider a substring.
        :param verbose: Whether to print debug information during the binary search.
        '''

        left, right = 0, len(query)
        
        effective_n = 0
        matches_indices = (None, None)

        while left <= right:
            mid = left + (right - left) // 2
            suffix = query[-mid:]

            if verbose:
                print(f'\nleft={left}, right={right}, mid={mid}')
                print(f'testing suffix {suffix}')
            
            new_num_matches, new_matches_indices  = retrieve_num_substrings(
                self.suffix_array,
                self.documents_tkn,
                suffix,
                extend=1
            )

            if verbose:
                print(f'matches: {new_num_matches}')

            if new_num_matches >= min_count:
                effective_n = mid
                matches_indices = new_matches_indices
                left = mid + 1 
            else:
                right = mid - 1
        
        if matches_indices[0] is None or matches_indices[1] is None:
            return [], 0

        matching = get_retrieved_substrings(
            matches_indices[0],
            matches_indices[1],
            self.suffix_array,
            self.documents_tkn,
            query[-effective_n:],
            extend=1
        )

        return matching, effective_n

    def prob_next_distr(
            self,
            prefix: np.array
        ) -> NextTokenResult:
        '''
        Outputs the full probability distribution of the next token given a prefix.
        :param prefix: Prefix to predict next token.
        '''

        assert self.tokenizer.eos_token_id not in prefix

        vocab_size = self.tokenizer.vocab_size

        matching, effective_n = self.get_longest_matching_next(prefix)
        norm_count = len(matching)

        if norm_count == 0:
            return NextTokenResult(
                distr=None,
                count=None,
                effective_n=0
            )

        result_count = np.zeros(vocab_size, dtype=np.float32)
        for m in matching:
            next_index = m[effective_n]
            result_count[next_index] += 1
        
        result = result_count / norm_count
        
        return NextTokenResult(
            distr=result,
            count=result_count,
            effective_n=effective_n
        )

    def greedy_next(
            self,
            prefix: np.array,
            max_len: int=32,
            verbose: bool=False
        ) -> np.array:
        '''
        Greedily predicts the next tokens given a prefix.
        :param prefix: Prefix to predict next tokens on.
        :param max_len: Maximum length of the output (including the prefix).
        :param verbose: Whether to print debug information.
        '''

        pbar = tqdm.tqdm(total=max_len)
        result = prefix
        while len(result) < max_len:
            pbar.update(1)
            next_result = self.prob_next_distr(result)
            next_token = np.argmax(next_result.distr)
            result = np.concatenate([result, np.array([next_token])])

            if verbose:
                print(f'\n-------------- i={len(result)}')
                print(f'next={next_token}, count={next_result.count[next_token]}, n={next_result.effective_n}')
                print(self.decode_single(result))

            if next_token == self.tokenizer.eos_token_id:
                break

        return result

    def _prob_next_distr_batch_worker(
            self,
            input_ids_batch: list[np.array],
            verbose: bool=True,
            pos: int=0
        ) -> list[NextTokenResult]:
        '''
        Predicts a batch of full next token distributions. Single threaded.
        :param input_ids_batch: List of tokenized prefixes.
        :param verbose: Whether to print debug information.
        :param pos: Position of the progress bar.
        '''
        
        pbar = tqdm.tqdm(input_ids_batch, position=pos) if verbose else input_ids_batch

        return [self.prob_next_distr(input_ids) for input_ids in pbar]

    def prob_next_distr_batch(
            self,
            input_ids_batch: list[np.array],
            verbose: bool=True,
            nworkers: int=1
        ) -> list[NextTokenResult]:
        '''
        Predicts a batch of full next token distributions. Set nworkers > 1 for parallelism.
        :param input_ids_batch: List of tokenized prefixes.
        :param verbose: Whether to print debug information.
        :param nworkers: Number of workers.
        '''

        if nworkers == 1:
            return self._prob_next_distr_batch_worker(input_ids_batch, verbose=verbose)

        pool = mp.Pool(nworkers)
        
        chunksize = len(input_ids_batch) // nworkers
        chunks = [
            input_ids_batch[i:i + chunksize]
            for i in range(0, len(input_ids_batch), chunksize)
        ]

        assert sum([len(c) for c in chunks]) == len(input_ids_batch)

        results = pool.starmap(
            self._prob_next_distr_batch_worker,
            [(c, verbose, i) for i, c in enumerate(chunks)]
        )

        pool.close()

        return [r for rs in results for r in rs]

    def forced_gen(
            self,
            input_ids: np.array,
            verbose: bool=True,
            nworkers: int=1
        ) -> list[NextTokenResult]:
        '''
        Predicts the full next token distribution at each position of the input.
        i.e., predict with teacher forcing.
        :param input_ids: Single tokenized input.
        :param verbose: Whether to print debug information.
        '''

        input_ids_batch = [input_ids[:i + 1] for i in range(len(input_ids))]
        next_distr_batch = self.prob_next_distr_batch(
            input_ids_batch,
            verbose=verbose,
            nworkers=nworkers
        )

        return next_distr_batch
