# stats

The `stats` package provides standard statistic computations operating over floating-point data (both 32 and 64 bit) in the following formats.  Each statistic returns a single scalar value summarizing the data in a different way.  Some formats also support multi-dimensional tensor data, returning a summary stat for each tensor value, using the outer-most ("row-wise") dimension to summarize over.

* `[]float32` and `[]float64` slices, as e.g., `Mean32` and `Mean64`, skipping any `NaN` values as missing data. 

* `tensor.Float32`, `tensor.Float64` using the underlying `Values` slice, and other generic `Tensor` using the `Floats` interface (less efficient).

* `table.IndexView` indexed views of `table.Table` data, with `*Column` functions (e.g., `MeanColumn`) using names to specify columns, and `*Index` versions operating on column indexes.  Also available for this type are `CountIf*`, `PctIf*`, `PropIf*` functions that return count, percentage, or propoprtion of values according to given function.

## Stats

The following statistics are supported (per the `Stats` enum in `stats.go`):

* `Count`:  count of number of elements
* `Sum`:  sum of elements
* `Prod`:  product of elements
* `Min`:  minimum value
* `Max`:  max maximum value
* `MinAbs`: minimum absolute value
* `MaxAbs`: maximum absolute value
* `Mean`:  mean mean value
* `Var`:  sample variance (squared diffs from mean, divided by n-1)
* `Std`:  sample standard deviation (sqrt of Var)
* `Sem`:  sample standard error of the mean (Std divided by sqrt(n))
* `L1Norm`: L1 Norm: sum of absolute values
* `SumSq`:  sum of squared element values
* `L2Norm`:  L2 Norm: square-root of sum-of-squares
* `VarPop`:  population variance (squared diffs from mean, divided by n)
* `StdPop`:  population standard deviation (sqrt of VarPop)
* `SemPop`:  population standard error of the mean (StdPop divided by sqrt(n))
* `Median`:  middle value in sorted ordering (only for IndexView)
* `Q1`:  Q1 first quartile = 25%ile value = .25 quantile value (only for IndexView)
* `Q3`:  Q3 third quartile = 75%ile value = .75 quantile value (only for IndexView)
 

