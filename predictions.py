import numpy as np
import scipy.sparse as sps
from dataclasses import dataclass


@dataclass
class NextTokenResult:
    distr: np.array
    count: np.array
    effective_n: int


class SparsePredictions:
    def __init__(self, sparse_data: dict[str, sps.csr_matrix]):
        assert isinstance(sparse_data, dict)
        
        self.sparse_data = sparse_data

    @classmethod
    def from_dense(cls, data: list[NextTokenResult]):
        length = len(data)
        vocab_size = data[0].count.shape[0]

        counts = np.zeros((length, vocab_size))
        distr = np.zeros((length, vocab_size), dtype=float)
        effective_n = np.zeros((length,))

        for i, d in enumerate(data):
            counts[i] = d.count
            distr[i] = d.distr
            effective_n[i] = d.effective_n
        
        counts_sparse = sps.csr_matrix(counts)
        distr_sparse = sps.csr_matrix(distr, dtype=float)
        
        return cls({
            'counts': counts_sparse,
            'distr': distr_sparse,
            'effective_n': effective_n
        })

    @classmethod
    def from_path(cls, fp_format: str):
        counts = sps.load_npz(fp_format.format(fn='counts.npz'))
        distr = sps.load_npz(fp_format.format(fn='distr.npz'))
        effective_n = np.load(fp_format.format(fn='effective_n.npy'))

        return cls({
            'counts': counts,
            'distr': distr,
            'effective_n': effective_n
        })

    def save(self, fp_format: str):
        print(self.sparse_data)
        sps.save_npz(fp_format.format(fn='counts.npz'), self.sparse_data['counts'])
        sps.save_npz(fp_format.format(fn='distr.npz'), self.sparse_data['distr'])
        np.save(fp_format.format(fn='effective_n.npy'), self.sparse_data['effective_n'])

    def get_predictions(self) -> np.array:
        return np.asarray(self.sparse_data['distr'].argmax(axis=1))[:, 0]
