# norm

`norm` provides normalization of vector and tensor values.  The basic functions operate on either `[]float32` or `[]float64` data, with Tensor versions using those, only for Float32 and Float64 tensors.

* DivNorm does divisive normalization of elements
* SubNorm does subtractive normalization of elements
* ZScore subtracts the mean and divides by the standard deviation
* Abs performs absolute-value on all elements (e.g., use prior to [stats](../stats) to produce Mean of Abs vals etc).


