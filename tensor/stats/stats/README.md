# stats

The `stats` package provides standard statistic computations operating on the `tensor.Indexed` standard data representation, using this standard function:
```Go
type StatsFunc func(in, out *tensor.Indexed)
```

For 1D data, the output is a scalar value in the out tensor, and otherwise it is an n-dimensional "cell" with outermost row dimension set to 1.

All stats are registered in the `tensor.Funcs` global list, and can be called using the `FuncName` method, e.g.,:
```Go
tensor.Call(Mean.FuncName(), in, out)
```

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

Here is the general info associated with these function calls:

`StatsFunc` is the function signature for a stats function, where the output has the same shape as the input but with the outermost row dimension size of 1, and contains the stat value(s) for the "cells" in higher-dimensional tensors, and a single scalar value for a 1D input tensor.

Critically, the stat is always computed over the outer row dimension, so each cell in a higher-dimensional output reflects the _row-wise_ stat for that cell across the different rows.  To compute a stat on the `tensor.SubSpace` cells themselves, must call on a `tensor.New1DViewOf` the `RowTensor`.  

All stats functions skip over NaN's, as a missing value.

Stats functions cannot be computed in parallel, e.g., using VectorizeThreaded or GPU, due to shared writing to the same output values.  Special implementations are required if that is needed.

## Groups

The `Groups` function (and `TableGroups` convenience function for `table.Table` columns) creates lists of indexes for each unique value in a 1D tensor, and `GroupStats` calls a stats function on those groups, thereby creating a "pivot table" that summarizes data in terms of the groups present within it. The data is stored in a [datafs](../datafs) data filesystem, which can be visualized and further manipulated.

For example, with this data:
```
Person  Score  Time
Alia    40     8
Alia    30     12
Ben     20     10
Ben     10     12
```
The `Groups` function called on the `Person` column would create the following `datafs` structure:
```
Groups
    Person
        Alia:  [0,1]   // int tensor
        Ben:   [2,3]
    // other groups here if passed
```
Then the `GroupStats` function operating on this `datafs` directory, using the `Score` and `Time` data and the `Mean` stat, followed by a second call with the `Sem` stat, would produce:
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

The `Person` directory can be turned directly into a `table.Table` and plotted or otherwise used, in the `datafs` system and associated `databrowser`.

See the [examples/planets](../examples/planets) example for an interactive exploration of data on exoplanets using the `Groups` functions.

## Vectorize functions

See [vecfuncs.go](vecfuncs.go) for corresponding `tensor.Vectorize` functions that are used in performing the computations.  These cannot be parallelized directly due to shared writing to output accumulators, and other ordering constraints.  If needed, special atomic-locking or other such techniques would be required.

