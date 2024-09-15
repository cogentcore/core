# Tensor

Tensor and related sub-packages provide a simple yet powerful framework for representing n-dimensional data of various types, providing similar functionality to the widely used `numpy` and `pandas` libraries in python, and the commercial MATLAB framework.

The `tensor.Indexed` type provides the universal representation of a homogenous data type throughout all the packages here, from scalar to vector, matrix, and beyond, because it can efficiently represent any kind of element with sufficient flexibility to enable a huge range of computations to be elegantly expressed.  The indexes provide a specific view onto the underlying [Tensor] data, applying to the outermost _row_ dimension (with default row-major indexing).  For example, sorting and filtering a tensor only requires updating the indexes while doing nothing to the Tensor itself.

The `float64` ("Float") and `string` types are used as universal input / output types, and for intermediate computation in the math functions. Any performance-critical code can be optimized for a specific data type, but these universal interfaces are suitable for misc ad-hoc data analysis.

The [cosl](../cosl) _Cogent [Scripting, Science, Statistics, Shell...] Language_ uses `tensor.Indexed` data types exclusively to allow simple intuitive math expressions to be transpiled into corresponding Go code, providing an efficient, elegant, yet type-safe and computationally powerful framework for data processing of all sorts.  All of the standard math, statistics, etc functionality is available using the [tmath](tmath), [stats](stats), and other associated packages described below.  

Use the `[Set]FloatRowCell` methods wherever possible, for the most efficient and natural indirection through the indexes.  The 1D methods on underlying tensor data do not indirect through the indexes and must be called directly on the [Tensor].

The `Vectorize` function and its variants provide a universal "apply function to tensor data" mechanism (often called a "map" function, but that name is already taken in Go).  It takes an `N` function that determines how many indexes to iterate over (and this function can also do any initialization prior to iterating), a compute function that gets an index and a list of tensors, which is applied to every index, and a varargs list of indexed tensors.  It is completely up to the compute function how to interpret the index.  There is a Threaded version of this for parallelizable functions, and a GPU version.

All tensor package functions are registered using a single name to function map (`Funcs`).

* [table](table) organizes multiple Tensors as columns in a data `Table`, aligned by a common outer row dimension.  Because the columns are tensors, each cell (value associated with a given row) can also be n-dimensional, allowing efficient representation of patterns and other high-dimensional data.  Furthermore, the entire column is organized as a single contiguous slice of data, so it can be efficiently processed.  A `Table` automatically supplies a shared list of row Indexes for its `Indexed` columns, efficiently allowing all the heterogeneous data columns to be sorted and filtered together.

    Data that is encoded as a slice of `struct`s can be bidirectionally converted to / from a Table, which then provides more powerful sorting, filtering and other functionality, including [plot/plotcore](../plot/plotcore).

* [datafs](datafs) provides a virtual filesystem (FS) for organizing arbitrary collections of data, supporting interactive, ad-hoc (notebook style) as well as systematic data processing. Interactive [cosl](../cosl) shell commands (`cd`, `ls`, `mkdir` etc) can be used to navigate the data space, with numerical expressions immediately available to operate on the data and save results back to the filesystem.  Furthermore, the data can be directly copied to / from the OS filesystem to persist it, and `cosl` can transparently access data on remote systems through ssh.  Furthermore, the [databrowser](databrowser) provides a fully interactive GUI for inspecting and plotting data.

* [tensorcore](tensorcore) provides core widgets for graphically displaying the `Tensor` and `Table` data, which are used in `datafs`.

* [tmath](tmath) implements all standard math functions on `tensor.Indexed` data, including the standard `+, -, *, /` operators.  `cosl` then calls these functions.

* [plot/plotcore](../plot/plotcore) supports interactive plotting of `Table` data.

* [bitslice](bitslice) is a Go slice of bytes `[]byte` that has methods for setting individual bits, as if it was a slice of bools, while being 8x more memory efficient.  This is used for encoding null entries in  `etensor`, and as a Tensor of bool / bits there as well, and is generally very useful for binary (boolean) data.

* [stats](stats) implements a number of different ways of analyzing tensor and table data, including:
    - [split](split) supports splitting a Table into any number of indexed sub-views and aggregating over those (i.e., pivot tables), grouping, summarizing data, etc.
    - [metric](metric) provides similarity / distance metrics such as `Euclidean`, `Cosine`, or `Correlation` that operate on slices of `[]float64` or `[]float32`.
    - TODO: now in metric: [simat](simat) provides similarity / distance matrix computation methods operating on `etensor.Tensor` or `etable.Table` data.  The `SimMat` type holds the resulting matrix and labels for the rows and columns, which has a special `SimMatGrid` view in `etview` for visualizing labeled similarity matricies.
    - TODO: where? [pca](pca) provides principal-components-analysis (PCA) and covariance matrix computation functions.
    - TODO: in metric? [clust](clust) provides standard agglomerative hierarchical clustering including ability to plot results in an eplot.

# Standard shapes and dimensional terminology

In general, **1D** refers to a flat, 1-dimensional list.  There are various standard shapes of tensor data that different functions expect:

* **Flat, 1D**: this is the simplest data shape.  For example, the [stats](stats) functions report summary statistics for all values of such data, across the one row-wise dimension.  `Indexed` views of this 1D data provide fine-grained filtering and sorting of all the data (indexes are only available for the outermost row dimension).

* **Row, Cell 2D**: The outermost row dimension can be sorted, filtered in an `Indexed` view, and the inner "cells" of data are organized in a simple flat 1D `SubSpace`, so they can be easily processed.  In most packages including [tmath](tmath) and [stats](stats), 2+ dimensional data will be automatically re-shaped into this Row, Cell format, and processed as row-wise list of cell-wise patterns.  For example, `stats` will aggregate each cell separately across rows, so you end up with the "average pattern" when you do `stats.Mean` for example.

    A higher-dimensional tensor can also be re-shaped into this row, cell format by collapsing any number of additional outer dimensions into a longer, effective "row" index, with the remaining inner dimensions forming the cell-wise patterns.  You can decide where to make the cut, and the `RowCellSplit` function makes it easy to create a new view of an existing tensor with this split made at a given dimension.

* **Matrix 2D**: For matrix algebra functions, a 2D tensor is treated as a standard row-major 2D matrix, which can be processed using `gonum` based matrix and vector operations.

# Cheat Sheet

`ix` is the `Indexed` tensor for these examples:

## Table Access

### 1D

```Go
// 5th element in tensor regardless of shape:
val := ix.Float1D(5)
```

```Go
// value as a string regardless of underlying data type; numbers converted to strings.
str := ix.String1D(2)
```

### 2D Row, Cell

```Go
// value at row 3, cell 2 (flat index into entire `SubSpace` tensor for this row)
// The row index will be indirected through any `Indexes` present on the Indexed view.
val := ix.FloatRowCell(3, 2)
// string value at row 2, cell 0. this is safe for 1D and 2D+ shapes
// and is a robust way to get 1D data from tensors of unknown shapes.
str := ix.FloatRowCell(2, 0)
```

```Go
// get the whole n-dimensional tensor of data cells at given row.
// row is indirected through indexes.
// the resulting tensor is a "subslice" view into the underlying data
// so changes to it will automatically update the parent tensor.
tsr := ix.RowTensor(4)
....
// set all n-dimensional tensor values at given row from given tensor.
ix.SetRowTensor(tsr, 4) 
```

```Go
// returns a flat, 1D Indexed view into n-dimensional tensor values at 
// given row.  This is used in compute routines that operate generically
// on the entire row as a flat pattern.
ci := ix.Cells1D(5)
```

### Full N-dimensional Indexes

```Go
// for 3D data
val := ix.Float(3,2,1)
```

# History

This package was originally developed as [etable](https://github.com/emer/etable) as part of the _emergent_ software framework.  It always depended on the GUI framework that became Cogent Core, and having it integrated within the Core monorepo makes it easier to integrate updates, and also makes it easier to build advanced data management and visualization applications.  For example, the [plot/plotcore](../plot/plotcore) package uses the `Table` to support flexible and powerful plotting functionality.

