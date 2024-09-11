# Tensor

Tensor and related sub-packages provide a simple yet powerful framework for representing n-dimensional data of various types, providing similar functionality to the widely used `numpy` and `pandas` libraries in python, and the commercial MATLAB framework.

The `tensor.Indexed` type provides the universal representation of a homogenous data type throughout all the packages here, from scalar to vector, matrix, and beyond, because it can efficiently represent any kind of element with sufficient flexibility to enable a huge range of computations to be elegantly expressed.  The indexes provide a specific view onto the underlying [Tensor] data, applying to the outer-most _row_ dimension (with default row-major indexing).  For example, sorting and filtering a tensor only requires updating the indexes while doing nothing to the Tensor itself.

The `float64` ("Float") and `string` types are used as universal input / output types, and for intermediate computation in the math functions. Any performance-critical code can be optimized for a specific data type, but these universal interfaces are suitable for misc ad-hoc data analysis.

The [cosl](../cosl) _Cogent [Scripting, Science, Statistics, Shell...] Language_ uses `tensor.Indexed` data types exclusively to allow simple intuitive math expressions to be transpiled into corresponding Go code, providing an efficient, elegant, yet type-safe and computationally powerful framework for data processing of all sorts.  All of the standard math, statistics, etc functionality is available using the [tmath](tmath), [stats](stats), and other associated packages described below.  

Use the `[Set]FloatRowCell` methods wherever possible, for the most efficient and natural indirection through the indexes.  The 1D methods on underlying tensor data do not indirect through the indexes and must be called directly on the [Tensor].

The `Vectorize` function and its variants provide a universal "apply function to tensor data" mechanism (often called a "map" function, but that name is already taken in Go).  It takes an `N` function that determines how many indexes to iterate over (and this function can also do any initialization prior to iterating), a compute function that gets an index and a list of tensors, which is applied to every index, and a varargs list of indexed tensors.  It is completely up to the compute function how to interpret the index.  There is a Threaded version of this for parallelizable functions, and a GPU version.

All tensor package functions are registered using a single name to function map (`Funcs`).

* [table](table) organizes multiple Tensors as columns in a data `Table`, aligned by a common outer row dimension.  Because the columns are tensors, each cell (value associated with a given row) can also be n-dimensional, allowing efficient representation of patterns and other high-dimensional data.  Furthermore, the entire column is organized as a single contiguous slice of data, so it can be efficiently processed.  The `table` package also has an `Indexed` that provides a shared index for the column tensors.

    Data that is encoded as a slice of `struct`s can be bidirectionally converted to / from a Table, which then provides more powerful sorting, filtering and other functionality, including [plot/plotcore](../plot/plotcore).

* [datafs](datafs) provides a virtual filesystem (FS) for organizing arbitrary collections of data, supporting interactive, ad-hoc (notebook style) as well as systematic data processing. Interactive [cosl](../cosl) shell commands (`cd`, `ls`, `mkdir` etc) can be used to navigate the data space, with numerical expressions immediately available to operate on the data and save results back to the filesystem.  Furthermore, the data can be directly copied to / from the OS filesystem to persist it, and `cosl` can transparently access data on remote systems through ssh.  Furthermore, the [databrowser](databrowser) provides a fully interactive GUI for inspecting and plotting data.

* [tensorcore](tensorcore) provides core widgets for graphically displaying the `Tensor` and `Table` data, which are used in `datafs`.

* [tmath](tmath) implements all standard math functions on `tensor.Indexed` data, including the standard `+, -, *, /` operators.  `cosl` then calls these functions.

* [stats](stats) implements a number of different ways of analyzing tensor and table data

* [plot/plotcore](../plot/plotcore) supports interactive plotting of `Table` data.


# History

This package was originally developed as [etable](https://github.com/emer/etable) as part of the _emergent_ software framework.  It always depended on the GUI framework that became Cogent Core, and having it integrated within the Core monorepo makes it easier to integrate updates, and also makes it easier to build advanced data management and visualization applications.  For example, the [plot/plotcore](../plot/plotcore) package uses the `Table` to support flexible and powerful plotting functionality.

