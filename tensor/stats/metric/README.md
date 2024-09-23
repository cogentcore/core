# metric

`metric` provides various similarity / distance metrics for comparing two tensors, operating on the `tensor.Tensor` standard data representation, using this standard function:
```Go
type MetricFunc func(a, b, out tensor.Tensor) error
```

The metric functions always operate on the outermost _row_ dimension, and it is up to the caller to reshape the tensors to accomplish the desired results. The two tensors must have the same shape.

* To obtain a single summary metric across all values, use `tensor.As1D`.

* For `RowMajor` data that is naturally organized as a single outer _rows_ dimension with the remaining inner dimensions comprising the _cells_, the results are the metric for each such cell computed across the outer rows dimension.  For the `Euclidean` metric for example, each cell has the difference for that cell value across all the rows between the two tensors. See [Matrix functions](#matrix-functions) below for a function that computes the distances _between each cell pattern and all the others_, as a distance or similarity matrix.

* Use `tensor.NewRowCellsView` to reshape any tensor into a 2D rows x cells shape, with the cells starting at a given dimension. Thus, any number of outer dimensions can be collapsed into the outer row dimension, and the remaining dimensions become the cells.

## Metrics

### Value _increases_ with increasing distance (i.e., difference metric)

* `Euclidean` or `L2Norm`: the square root of the sum of squares differences between tensor values.
* `SumSquares`:  the sum of squares differences between tensor values.
* `Abs`or `L2Norm`: the sum of the absolute value of differences between tensor values.
* `Hamming`: the sum of 1s for every element that is different, i.e., "city block" distance.
* `EuclideanBinTol`:  the `Euclidean` square root of the sum of squares differences between tensor values, with binary tolerance: differences < 0.5 are thresholded to 0.
* `SumSquaresBinTol`: the `SumSquares` differences between tensor values,  with binary tolerance: differences < 0.5 are thresholded to 0.
* `InvCosine`: is 1-`Cosine`, which is useful to convert it to an Increasing metric where more different vectors have larger metric values.
* `InvCorrelation`: is 1-`Correlation`, which is useful to convert it to an Increasing metric where more different vectors have larger metric values.
* `CrossEntropy`: is a standard measure of the difference between two probabilty distributions, reflecting the additional entropy (uncertainty) associated with measuring probabilities under distribution b when in fact they come from distribution a.  It is also the entropy of a plus the divergence between a from b, using Kullback-Leibler (KL) divergence.  It is computed as: a * log(a/b) + (1-a) * log(1-a/1-b).

### Value _decreases_ with increasing distance (i.e., similarity metric)

* `InnerProduct`:  the sum of the co-products of the tensor values.
* `Covariance`: the co-variance between two vectors, i.e., the mean of the co-product of each vector element minus the mean of that vector: cov(A,B) = E[(A - E(A))(B - E(B))].
* `Correlation`: the standardized `Covariance` in the range (-1..1), computed as the mean of the co-product of each vector element minus the mean of that vector, normalized by the product of their standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B). Equivalent to the `Cosine` of mean-normalized vectors.
* `Cosine`: the high-dimensional angle between two vectors, in range (-1..1) as the normalized `InnerProduct`: inner product / sqrt(ssA * ssB).  See also `Correlation`.

Here is general info about these functions:

The output must be a `tensor.Values` tensor, and it is automatically shaped to hold the stat value(s) for the "cells" in higher-dimensional tensors, and a single scalar value for a 1D input tensor.

All metric functions skip over NaN's, as a missing value.

Metric functions cannot be computed in parallel, e.g., using VectorizeThreaded or GPU, due to shared writing to the same output values.  Special implementations are required if that is needed.

# Matrix functions

* `Matrix` computes a distance / similarity matrix using a metric function, operating on the n-dimensional sub-space patterns on a given tensor (i.e., a row-wise list of patterns). The result is a square rows x rows matrix where each cell is the metric value for the pattern at the given row. The diagonal contains the self-similarity metric.

* `CrossMatrix` is like `Matrix` except it compares two separate lists of patterns.

* `CovarianceMatrix` computes the _covariance matrix_ for row-wise lists of patterns, where the result is a square matrix of cells x cells size ("cells" is number of elements in the patterns per row), and each value represents the extent to which value of a given cell covaries across the rows of the tensor with the value of another cell. For example, if the rows represent time, then the covariance matrix represents the extent to which the patterns tend to move in the same way over time.

* `PCA` and `SVD` operate on the `CovarianceMatrix` to extract the "principal components" of covariance, in terms of the _eigenvectors_ and corresponding _eigenvalues_ of this matrix. The eigenvector (component) with the largest eigenvalue is the "direction" in n-dimensional pattern space along which there is the greatest variance in the patterns across the rows.

* `ProjectOnMatrixColumn` is a convenient function for projecting data along a vector extracted from a matrix, which allows you to project data along an eigenvector from the PCA or SVD functions. By doing this projection along the strongest 2 eigenvectors (those with the largest eigenvalues), you can visualize high-dimensional data in a 2D plot, which typically reveals important aspects of the structure of the underlying high-dimensional data, which is otherwise hard to see given the difficulty in visualizing high-dimensional spaces.


