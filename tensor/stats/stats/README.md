# stats

The `stats` package provides standard statistic computations operating on the `tensor.Indexed` standard data representation, using this standard function:
```Go
type StatsFunc func(in, out *tensor.Indexed)
```

For 1D data, the output is a scalar value in the out tensor, and otherwise it is an n-dimensional "cell" with outer-most row dimension set to 1.

There is a `StatsFuncs` map of named stats funcs, which is initialized with the standard Stats per below, and any additional user-defined functions can be added to.

## Stats

The following statistics are supported (per the `Stats` enum in `stats.go`):

* `Count`:  count of number of elements
* `Sum`:  sum of elements
* `SumAbs`:  sum of absolute-value-of elements (same as L1Norm)
* `L1Norm`: L1 Norm: sum of absolute values (same as SumAbs)
* `Prod`:  product of elements
* `Min`:  minimum value
* `Max`:  maximum value
* `MinAbs`: minimum of absolute values
* `MaxAbs`: maximum of absolute values
* `Mean`:  mean value
* `Var`:  sample variance (squared diffs from mean, divided by n-1)
* `Std`:  sample standard deviation (sqrt of Var)
* `Sem`:  sample standard error of the mean (Std divided by sqrt(n))
* `SumSq`:  sum of squared element values
* `L2Norm`:  L2 Norm: square-root of sum-of-squares
* `VarPop`:  population variance (squared diffs from mean, divided by n)
* `StdPop`:  population standard deviation (sqrt of VarPop)
* `SemPop`:  population standard error of the mean (StdPop divided by sqrt(n))
* `Median`:  middle value in sorted ordering (only for Indexed)
* `Q1`:  Q1 first quartile = 25%ile value = .25 quantile value (only for Indexed)
* `Q3`:  Q3 third quartile = 75%ile value = .75 quantile value (only for Indexed)
 
## Vectorize functions

See [vecfuncs.go](vecfuncs.go) for corresponding `tensor.Vectorize` functions that are used in performing the computations.  These cannot be parallelized directly due to shared writing to output accumulators, and other ordering constraints.  If needed, special atomic-locking or other such techniques would be required.

