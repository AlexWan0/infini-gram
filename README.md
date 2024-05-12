# Infini-gram implementation
This repo contains an (unofficial, **and very rough**) Golang implementation of the infini-gram model described in [Liu et al. (2024)](https://arxiv.org/abs/2401.17377).

# Build
```
go build
```

# Run
```
./infinigram --train_file corpus.txt --out_dir output --tokenizer_config tokenizer.json
```

where `corpus.txt` contains one document per line. `tokenizer.json` corresponds to the HuggingFace pretrained Tokenizers file (e.g., [for gpt2](https://huggingface.co/openai-community/gpt2/blob/main/tokenizer.json))

The argument `--interactive_mode {0,1}` lets you query for next-token and greedy generation, respectively.

# Third-party libraries
I use the `text_64` function implemented in the [Go `suffixarray` library](https://pkg.go.dev/index/suffixarray)---the files under `suffixarray/` are from this library with minor modifications.
