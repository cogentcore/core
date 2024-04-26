# histogram

This package computes a histogram of values in either a `[]float32` or `[]float64` array: a count of number of values within different bins of value ranges.

`F32` or `F64` computes the raw counts into a corresponding float array, and `F32Table` or `F64Table` construct an `etable` with Value and Count columns, suitable for plotting.

The Value column represents the min value for each bin, with the max being, the value of the next bin, or the max if at the end.

