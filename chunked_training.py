from transformers import AutoTokenizer
from datasets import load_dataset
from infinigram import InfiniGramModel


num_chunks = 2
assert 100 % num_chunks == 0

for i in range(num_chunks):
    pct_size = 100 // num_chunks
    start, end = i * pct_size, (i + 1) * pct_size

    dataset = load_dataset(
        'mit-han-lab/pile-val-backup',
        split=f'validation[{start}%:{end}%]'
    )

    tokenizer = AutoTokenizer.from_pretrained('openai-community/gpt2')

    model = InfiniGramModel.from_data(dataset['text'], tokenizer, nworkers=4)
    model.save_pretrained(f'data/pile_val_{i}_of_{num_chunks}')
