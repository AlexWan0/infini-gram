# Infini-gram implementation
This repo contains two (unofficial) implementations of the infini-gram model described in [Liu et al. (2024)](https://arxiv.org/abs/2401.17377). This branch (`main`) contains the Python implementation. A Golang implementation can be found in the [`go_rust_tokenizers` branch](https://github.com/AlexWan0/infini-gram/tree/go_rust_tokenizers). This particular Golang implementation uses bindings to the official tokenizers library.

## Implementation TODOs
* Parallelism during inference is annoying because the suffix array and corpora should be shared between multiple processes
* Fix caching
* Inference that backsoff for arbitrary cutoffs (to avoid sparse predictions)
* I *think* the original implementation gets the full distribution during inference by running `|V|` forward passes. For my implementation, I iterate through all matching substrings in order to build the distribution. This seemed to be faster on my smaller dataset (when there aren't that many matches compared the the vocab size), but I haven't tested it too comprehensively. Maybe I can try switching dynamically between the two.

## Usage
Training:
```python
from infinigram import InfiniGramModel

model = InfiniGramModel.from_data(training_data, tokenizer, nworkers=4)
model.save_pretrained('model_path/')
```

Generation:
```python
from infinigram import InfiniGramModel

model = InfiniGramModel.from_pretrained(args.model_dir)

gen_output = model.greedy_next(
    input_ids,
    verbose=False
)
```

Other methods: `prob_next_distr` predicts the full distribution of the next token. `get_longest_matching_next` finds the longest matching substring, plus one extra token.

## Prebuilt indices
[~860M token pile-val w/ openai-community/gpt2 tokenizer](https://drive.google.com/drive/folders/11WLVso4tMiqUrnERfYGbhl5wCy8WqKsH?usp=sharing) -- 4.6gb total <sub>(Note: the predictions on this model don't seem to exactly match up with their [demo](https://huggingface.co/spaces/liujch1998/infini-gram). I think this is due to different gpt-2 tokenizers that we're using (e.g., I don't see a significant discrepancy when using the Llama-2 tokenizer).)</sub>
