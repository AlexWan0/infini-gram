from tqdm import tqdm
from datasets import load_dataset

dataset = load_dataset(
    'mit-han-lab/pile-val-backup',
    split='validation',
    streaming=True
)

with open('./data/pile_val.txt', 'w') as f:
    for row in tqdm(dataset):
        f.write(row['text'] + '|||')
