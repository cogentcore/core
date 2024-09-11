# tmath is the Tensor math library

# math functions

All the standard library [math](https://pkg.go.dev/math) functions are implemented on `*tensor.Indexed`.

To properly handle the row-wise indexes, all processing is done using row, cell indexes, with the row indirected through the indexes.

The output result tensor(s) can be the same as the input for all functions (except where specifically noted), to perform an in-place operation on the same data.

The standard `Add`, `Sub`, `Mul`, `Div` (`+, -, *, /`) mathematical operators all operate element-wise, with a separate MatMul for matrix multiplication, which operates through gonum routines, for 2D Float64 tensor shapes with no indexes, so that the raw float64 values can be passed directly to gonum.

# norm functions

* `UnitNorm` subtracts `min` and divides by resulting `max` to normalize to 0..1 unit range.
* `ZScore` subtracts the mean and divides by the standard deviation.
* `Clamp` enforces min, max range, clamping values to those bounds if they exceed them.
* `Binarize` sets all values below a given threshold to 0, and those above to 1.
