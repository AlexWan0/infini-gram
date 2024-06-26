# Infini-gram implementation
This repo contains two (unofficial) implementations of the infini-gram model described in [Liu et al. (2024)](https://arxiv.org/abs/2401.17377). This branch contains the Golang implementation. The `main` branch contains a Python implementation.

The tokenizers used here are the [Go bindings to the official Rust library](https://github.com/daulet/tokenizers).

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

This implementation features:
* Next-token and greedy generation (`--interactive_mode {0,1}`)
* `mmap` to access both the tokenized documents and the suffix array; memory usage during inference should be minimal.
* Creating suffix arrays in chunks to further limit memory usage (`--max_mem`): you should hypothetically be able to train (and infer) on any sized corpus regardless of how much memory you have
* Set the minimum number of continuations needed a for suffix to be valid (`--min_matches`). e.g., you may set this at a value >= 2 to avoid sparse predictions where the $(n-1)$-gram corresponds to only a single document.
* A WIP alteration that uses FM-indices + wavelet trees instead of suffix arrays. Uses ~7.5x less disk space, but some queries take longer. See the FM-index branch for more info.

Run `./infinigram --help` for more information.

# TODO
- ~~Compare with official API~~ Pile-val with the Llama-2 tokenizer seems to match.
- Parallel inference
- Use an external suffix array algo (e.g., [fSAIS](https://github.com/dominikkempa/fsais)) to build indices for larger datasets.

# Third-party libraries
I use the `text_64` function implemented in the [Go `suffixarray` library](https://pkg.go.dev/index/suffixarray)---the files under `suffixarray/` are from this library with minor modifications.
