# norm

Docs: [GoDoc](https://pkg.go.dev/cogentcore.org/core/tensor/stats/agg)

`norm` provides normalization and norm metric computations, e.g., L2 = sqrt of sum of squares of a vector.  The basic functions operate on either `[]float32` or `[]float64` data, but Tensor versions are available too.

* DivNorm does divisive normalization of elements
* SubNorm does subtractive normalization of elements
* ZScore subtracts the mean and divides by the standard deviation
* Abs performs absolute-value on all elements (e.g., use prior to tsragg to produce Mean of Abs vals etc).


