# stats

The `stats` package provides standard statistic computations operating on the `tensor.Tensor` standard data representation, using this standard function:
```Go
type StatsFunc func(in, out tensor.Tensor) error
```
n
The stats functions always operate on the outermost _row_ dimension, and it is up to the caller to reshape the tensor to accomplish the desired results.

* To obtain a single summary statistic across all values, use `tensor.As1D`.

* For `RowMajor` data that is naturally organized as a single outer _rows_ dimension with the remaining inner dimensions comprising the _cells_, the results are the statistic for each such cell computed across the outer rows dimension. For the `Mean` statistic for example, each cell contains the average of that cell across all the rows.

* Use `tensor.NewRowCellsView` to reshape any tensor into a 2D rows x cells shape, with the cells starting at a given dimension. Thus, any number of outer dimensions can be collapsed into the outer row dimension, and the remaining dimensions become the cells.

By contrast, the [NumPy Statistics](https://numpy.org/doc/stable/reference/generated/numpy.mean.html#numpy.mean) functions take an `axis` dimension to compute over, but passing such arguments via the universal function calling api for tensors introduces complications, so it is simpler to just have a single designated behavior and reshape the data to achieve the desired results.

All stats are registered in the `tensor.Funcs` global list (for use in Goal), and can be called through the `Stats` enum e.g.:
```Go
stats.Mean.Call(in, out)
```

All stats functions skip over `NaN`s as a missing value, so they are equivalent to the `nanmean` etc versions in NumPy.

## Stats

The following statistics are supported (per the `Stats` enum in `stats.go`):

* `Count`:  count of number of elements
* `Sum`:  sum of elements
* `L1Norm`: L1 Norm: sum of absolute values
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
* `Median`:  middle value in sorted ordering (uses a `Rows` view)
* `Q1`:  Q1 first quartile = 25%ile value = .25 quantile value (uses `Rows`)
* `Q3`:  Q3 third quartile = 75%ile value = .75 quantile value (uses `Rows`)

Here is the general info associated with these function calls:

The output must be a `tensor.Values` tensor, and it is automatically shaped to hold the stat value(s) for the "cells" in higher-dimensional tensors, and a single scalar value for a 1D input tensor.

Stats functions cannot be computed in parallel, e.g., using VectorizeThreaded or GPU, due to shared writing to the same output values.  Special implementations are required if that is needed.

## Normalization functions

The stats package also has the following standard normalization functions for transforming data into standard ranges in various ways:

* `UnitNorm` subtracts `min` and divides by resulting `max` to normalize to 0..1 unit range.
* `ZScore` subtracts the mean and divides by the standard deviation.
* `Clamp` enforces min, max range, clamping values to those bounds if they exceed them.
* `Binarize` sets all values below a given threshold to 0, and those above to 1.

## Groups

The `Groups` function (and `TableGroups` convenience function for `table.Table` columns) creates lists of indexes for each unique value in a 1D tensor, and `GroupStats` calls a stats function on those groups, thereby creating a "pivot table" that summarizes data in terms of the groups present within it. The data is stored in a [tensorfs](../tensorfs) data filesystem, which can be visualized and further manipulated.

For example, with this data:
```
Person  Score  Time
Alia    40     8
Alia    30     12
Ben     20     10
Ben     10     12
```
The `Groups` function called on the `Person` column would create the following `tensorfs` structure:
```
Groups
    Person
        Alia:  [0,1]   // int tensor
        Ben:   [2,3]
    // other groups here if passed
```
Then the `GroupStats` function operating on this `tensorfs` directory, using the `Score` and `Time` data and the `Mean` stat, followed by a second call with the `Sem` stat, would produce:
```
Stats
    Person
       Person:    [Alia,Ben] // string tensor of group values of Person
       Score
           Mean:  [35, 15] // float64 tensor of means
           Sem:   [5, 5]
       Time
           Mean:  [10, 11]
           Sem:   [1, 0.5]
    // other groups here..
```

The `Person` directory can be turned directly into a `table.Table` and plotted or otherwise used, in the `tensorfs` system and associated `databrowser`.

See the [examples/planets](../examples/planets) example for an interactive exploration of data on exoplanets using the `Groups` functions.

## Vectorize functions

See [vec.go](vec.go) for corresponding `tensor.Vectorize` functions that are used in performing the computations.  These cannot be parallelized directly due to shared writing to output accumulators, and other ordering constraints.  If needed, special atomic-locking or other such techniques would be required.

