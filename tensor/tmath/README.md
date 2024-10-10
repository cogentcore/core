# tmath is the Tensor math library

# math functions

All the standard library [math](https://pkg.go.dev/math) functions are implemented on `*tensor.Tensor`.

To properly handle the row-wise indexes, all processing is done using row, cell indexes, with the row indirected through the indexes.

The output result tensor(s) can be the same as the input for all functions (except where specifically noted), to perform an in-place operation on the same data.

The standard `Add`, `Sub`, `Mul`, `Div` (`+, -, *, /`) mathematical operators all operate element-wise, with a separate MatMul for matrix multiplication, which operates through gonum routines, for 2D Float64 tensor shapes with no indexes, so that the raw float64 values can be passed directly to gonum.


