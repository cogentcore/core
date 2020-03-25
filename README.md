# mat32

`mat32` is a float32 based vector and matrix package for 2D & 3D graphics, based on the [G3N math32](https://github.com/g3n/engine) package, but using a value-based design instead of pointer-based, which simplifies chained expressions of multiple operators.

The [go-gl/mathgl](https://github.com/go-gl/mathgl) package is also comparable, which in turn is based on [image/math/f32](https://golang.org/x/image/math/f32) types, which use arrays instead of `struct`s with named X, Y, Z components.  The named components make things easier to read overall.  The G3N and this package support a much more complete set of vector and matrix math, covering almost everything you might need, including aggregate types such as triangles, planes, etc.

This package also includes the Matrix class from [fogleman/gg](https://github.com/fogleman/gg) (as `Mat2`) for 2D graphics -- this also includes additional support for SVG-style configuring of a matrix, in the `SetString` method.

# Value-based Vectors

The use of value-based methods means that vectors are passed and returned as values instead of pointers:

So, in this `mat32` package, `Add` looks like this:

```Go
// Add adds other vector to this one and returns result in a new vector.
func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}
```

versus G3N:

```Go
// Add adds other vector to this one.
// Returns the pointer to this updated vector.
func (v *Vector3) Add(other *Vector3) *Vector3 {
	v.X += other.X
	v.Y += other.Y
	v.Z += other.Z
	return v
}
```

The value-based design allows you to just string together sequences of expressions naturally, without worrying about allocating intermediate variables:

```Go
// Normal returns the triangle's normal.
func Normal(a, b, c Vec3) Vec3 {
	nv := c.Sub(b).Cross(a.Sub(b))
   ...
```

There may be a small performance cost for the value-based approach (comparative benchmarks have not yet been run), but the overall simplicity advantages are significant.

The matrix types still do use pointer-based logic because they are significantly larger and thus the performance issues are likely to be more important.

# Struct vs. Array Performance: Struct is much faster

This is a benchmark from Egon Elbre, showing that small arrays can be significantly slower than 
a struct: https://github.com/egonelbre/exp/blob/master/bench/vector_fusing/vector_test.go

```
# array
BenchmarkAddMul-32                      70589480                17.3 ns/op
# struct
BenchmarkStructAddMul-32                1000000000               0.740 ns/op
```

Discussion: https://github.com/golang/go/issues/15925

