# Infini-gram implementation
This repo contains an (unofficial) pure-python implementation of the infini-gram model described in [Liu et al. (2024)](https://arxiv.org/abs/2401.17377). This implementation builds a suffix array of the corpora to allow for the retrieval of matching substrings in `log N`. At inference time finding the next token probability distribution will then take `(log L + m) (log N)` time$^*$.

$^*$ I *think* the original implementation gets only the *number* of matching substrings when querying the suffix array. To get the full distribution, they then run `V` forward passes (i.e., `m = V`). For my implementation, I iterate through all matching substrings in order to build the distribution. This seemed to be faster on my smaller dataset (when there aren't that many matches compared the the vocab size), but I haven't tested it too comprehensively. Maybe I can try switching dynamically between the two.

## Implementation TODOs
* Parallelism during inference is annoying because the suffix array and corpora should be shared between multiple processes
* Add caching
* Inference that backsoff for arbitrary cutoffs (to avoid sparse predictions)

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

model = InfiniGramModel.from_pretrained(args.model)

gen_output = model.greedy_next(
    input_ids,
    verbose=False
)
```

Other methods: `prob_next_distr` predicts the full distribution of the next token. `get_longest_matching_next` finds the longest matching substring, plus one extra token.
