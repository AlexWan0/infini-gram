# Infini-gram implementation
This repo contains two (unofficial) implementations of the infini-gram model described in [Liu et al. (2024)](https://arxiv.org/abs/2401.17377). This branch contains the Golang implementation. The `main` branch contains a Python implementation.

The tokenizers used here are the [Go bindings to the official Rust library](https://github.com/daulet/tokenizers).

# FM-Index
This particular branch contains a WIP implementation of the infini-gram model using FM-indices + wavelet trees instead of suffix arrays. FM-indices use significantly less disk space while (hopefully) not tk

On my M1 Macbook Air 2020, each query for the next token distribution take from ~80ms to ~900ms depending on the number of vocabulary items that have continuations. The main bottleneck is that you can only efficiently query for *preceding* tokens given a suffix, so to create the full next token distribution we need to query every vocabulary item. To speed this up, I cache whether a 2-gram shows up in the corpus.

However, just checking for whether an n-gram exists does not run into this issue, and is much faster (500-900µs).

The main advantage of the FM-index, though, is that it takes significantly less space: ~500mb + ~90mb on Pile-val for the BWT array and 2-gram cache, compared ~800mb + ~3gb for the tokenized corpora and suffix array (you don't need to store the tokenized corpora at all using FM-indices). Furthermore, the 2-gram cache is just a sparse bit array corresponding to every possible combination of 2-grams (i.e., it's constant wrt corpus size, and will take *no more* than ~500mb). On average, [the storage space should be sublinear wrt the input data.](https://en.wikipedia.org/wiki/FM-index)

Some todos:
* ~~Current issue: making sure that the retrieved values respect byte boundaries.~~ fixed by using tokens directly instead of bytes; queries take longer though...
* Ensure same functionality as before (`numExtend`, chunking not implemented)
* Allow use of MMap with the wavelet trees
* Possibly better implementation of wavelet trees (e.g., with RRR?)
* Do BWT without constructing suffix array?
* Document retrieval: add document ID to sentinal; during inference, recursively find the previous token until you get the sentinal + document ID for the *previous* document. Then, you can start working backwards from the document ID + 1. At the cost of space, you could sprinkle in the document ID uniformly throughout your tokenized corpus.

# Build
First, build the rust tokenizers binary:
```bash
cd tokenizers
make
```

Then, you can build the infinigram binary:
```bash
cd ../
go build -ldflags "-s"
```

# Run
```
./infinigram --train_file corpus.txt --out_dir output --tokenizer_config tokenizer.json
```

where `corpus.txt` contains one document per line. `tokenizer.json` corresponds to the HuggingFace pretrained Tokenizers file (e.g., [for gpt2](https://huggingface.co/openai-community/gpt2/blob/main/tokenizer.json)).

Use the `--use_fm` flag to use the FM-index instead of suffix arrays.

This implementation features:
* Next-token and greedy generation (`--interactive_mode {0,1}`)
* `mmap` to access both the tokenized documents and the suffix array; memory usage during inference should be minimal.
* Creating suffix arrays in chunks to further limit memory usage (`--max_mem`): you should hypothetically be able to train (and infer) on any sized corpus regardless of how much memory you have
* Set the minimum number of continuations needed a for suffix to be valid (`--min_matches`). e.g., you may set this at a value >= 2 to avoid sparse predictions where the $(n-1)$-gram corresponds to only a single document.

Run `./infinigram --help` for more information.

# TODO
- ~~Compare with official API~~ Pile-val with the Llama-2 tokenizer seems to match (with both suffix arrays and FM-Index).
- Parallel inference
- Use an external suffix array algo (e.g., [fSAIS](https://github.com/dominikkempa/fsais)) to build indices for larger datasets.

# Third-party libraries
I use the `text_64` function implemented in the [Go `suffixarray` library](https://pkg.go.dev/index/suffixarray)---the files under `suffixarray/` are from this library with minor modifications.
