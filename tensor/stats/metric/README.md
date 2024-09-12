# metric

`metric` provides various similarity / distance metrics for comparing tensors, operating on the `tensor.Indexed` standard data representation.

The `Matrix` function returns a distance / similarity matrix computed from the n-dimensional "cells" of row-organized tensor data, and the `SimMat` type provides labels for displaying such matricies.

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

`MetricFunc` is the function signature for a metric function, where the output has the same shape as the inputs but with the outermost row dimension size of 1, and contains the metric value(s) for the "cells" in higher-dimensional tensors, and a single scalar value for a 1D input tensor.

Critically, the metric is always computed over the outer row dimension, so each cell in a higher-dimensional output reflects the _row-wise_ metric for that cell across the different rows.  To compute a metric on the `tensor.SubSpace` cells themselves, must call on a `tensor.New1DViewOf` the sub space.  See [simat](../simat) package.

All metric functions skip over NaN's, as a missing value, and use the min of the length of the two tensors.

Metric functions cannot be computed in parallel, e.g., using VectorizeThreaded or GPU, due to shared writing to the same output values.  Special implementations are required if that is needed.

