from transformers import PreTrainedTokenizerBase
import tqdm
import numpy as np
import multiprocessing as mp


def tokenize_single(string: str, tokenizer: PreTrainedTokenizerBase) -> np.array:
    return tokenizer(
        string,
        add_special_tokens=False,
        return_tensors='np',
        padding=False,
    )['input_ids'][0]


def tokenize_mp(
        strings: list[str],
        tokenizer: PreTrainedTokenizerBase,
        verbose: bool=False,
        nworkers=4
    ) -> list[np.array]:

    pool = mp.Pool(nworkers)
    
    chunksize = len(strings) // nworkers
    chunks = [strings[i:i + chunksize] for i in range(0, len(strings), chunksize)]

    results = pool.starmap(
        tokenize,
        [(c, tokenizer, verbose, i) for i, c in enumerate(chunks)]
    )

    pool.close()

    return [tkn for r in results for tkn in r]

def tokenize(
        strings: list[str],
        tokenizer: PreTrainedTokenizerBase,
        verbose: bool=False,
        pos: int=0
    ) -> list[np.array]:

    result = []
    
    pbar = tqdm.tqdm(strings, position=pos) if verbose else strings
    for s in pbar:
        result.append(tokenize_single(s, tokenizer))
    
    return result

def decode_single(tkn: np.array, tokenizer: PreTrainedTokenizerBase) -> str:
    return tokenizer.decode(tkn)

def decode(tkns: list[np.array], tokenizer: PreTrainedTokenizerBase) -> list[str]:
    return [decode_single(t, tokenizer) for t in tkns]
