# tmath is the Tensor math library

# math functions

All the standard library [math](https://pkg.go.dev/math) functions are implemented on `*tensor.Indexed`.

To properly handle the row-wise indexes, all processing is done using row, cell indexes, with the row indirected through the indexes.

# norm functions

* DivNorm does divisive normalization of elements
* SubNorm does subtractive normalization of elements
* ZScore subtracts the mean and divides by the standard deviation
* Abs performs absolute-value on all elements (e.g., use prior to [stats](../stats) to produce Mean of Abs vals etc).


