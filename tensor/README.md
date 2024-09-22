# Tensor

Tensor and related sub-packages provide a simple yet powerful framework for representing n-dimensional data of various types, providing similar functionality to the widely used [NumPy](https://numpy.org/doc/stable/index.html) and [pandas](https://pandas.pydata.org/) libraries in Python, and the commercial MATLAB framework.

The [Goal](../goal) augmented version of the _Go_ language directly supports numpy-like operations on tensors. A `Tensor` is comparable to the NumPy `ndarray` type, and it provides the universal representation of a homogenous data type throughout all the packages here, from scalar to vector, matrix and beyond. All functions take and return `Tensor` arguments.

The `Tensor` interface is implemented at the basic level with n-dimensional indexing into flat Go slices of any numeric data type (by `Number`), along with `String`, and `Bool` (which uses [bitslice](bitslice) for maximum efficiency). The `Shape` type provides all the n-dimensional indexing with arbitrary strides to allow any ordering, although _row major_ is the default and other orders have to be manually imposed.

In addition, there are three important "view" implementations of `Tensor` that wrap another Tensor to provide more flexible and efficient access to the data, consistent with the NumPy functionality.  See [Basic and Advanced Indexing](#basic-and-advanced-indexing) for more info.

* `Rows` provides an index-based view, with the `Indexes` applying to the outermost _row_ dimension, which allows sorting and filtering to operate only on the indexes, leaving the underlying Tensor unchanged. This view is returned by the [table](table) data table, which organizes multiple heterogenous Tensor columns along a common outer row dimension. Organizing data systematically along the row dimension eliminates many sources of ambiguity about how to process higher-dimensional data, and any given n-dimensional structure can be reshaped to fit in this row-based format.

* `Sliced` is a more general version of `Rows` that provides a sub-sliced view into the wrapped `Tensor`, using an indexed list along each dimension, not just the outermost one. Thus, it can provide a systematic, reordered and filtered view onto the raw data, and it has a well-defined shape in terms of the number of indexes per dimension. This corresponds well with the NumPy basic sliced indexing model.

* `Masked` provides a `Bool` masked view onto each element in the wrapped `Tensor`, where the two maintain the same shape).  Any cell with a `false` value in the bool mask returns a `NaN` (missing data), and `Set` functions are no-ops.

* `Indexed` uses a tensor of indexes where the final, innermost dimension is the same size as the number of dimensions in the wrapped source tensor. The overall shape of this view is that of the remaining outer dimensions of the Indexes tensor, and like other views, assignment and return values are taken from the corresponding indexed value in the wrapped source tensor.

The `float64` ("Float"), `int` ("Int"), and `string` ("String") types are used as universal input / output types, and for intermediate computation in the math functions. Any performance-critical code can be optimized for a specific data type, but these universal interfaces are suitable for misc ad-hoc data analysis.

The `[Set]FloatRow[Cell]` methods are used wherever possible, for the most efficient and natural indirection for row-major organized data. See [Standard shapes](#standard-shapes) for more info.

The `Vectorize` function and its variants provide a universal "apply function to tensor data" mechanism (often called a "map" function, but that name is already taken in Go). It takes an `N` function that determines how many indexes to iterate over (and this function can also do any initialization prior to iterating), a compute function that gets an index and a list of tensors, which is applied to every index, and a varargs list of indexed tensors. In general it is completely up to the compute function how to interpret the index, although we also support the [broadcasting](#broadcasting) principles from NumPy for binary functions operating on two tensors. There is a Threaded version of this for parallelizable functions, and a GPU version.

All tensor package functions are registered using a global name-to-function map (`Funcs`), and can be called by name via `tensor.Call` or `tensor.CallOut` (creates the appropriate output tensor for you). Standard enumerated functions in `stats` and `metrics` have a `FuncName` method that appends the package name, which is how they are registered and called. These functions are the equivalent of the [ufunc](https://numpy.org/doc/stable/user/basics.ufuncs.html) universal functions in NumPy.

* [table](table) organizes multiple Tensors as columns in a data `Table`, aligned by a common outer row dimension. Because the columns are tensors, each cell (value associated with a given row) can also be n-dimensional, allowing efficient representation of patterns and other high-dimensional data. Furthermore, the entire column is organized as a single contiguous slice of data, so it can be efficiently processed. A `Table` automatically supplies a shared list of row Indexes for its `Indexed` columns, efficiently allowing all the heterogeneous data columns to be sorted and filtered together.

    Data that is encoded as a slice of `struct`s can be bidirectionally converted to / from a Table, which then provides more powerful sorting, filtering and other functionality, including [plot/plotcore](../plot/plotcore).

* [datafs](datafs) provides a virtual filesystem (FS) for organizing arbitrary collections of data, supporting interactive, ad-hoc (notebook style) as well as systematic data processing. Interactive [goal](../goal) shell commands (`cd`, `ls`, `mkdir` etc) can be used to navigate the data space, with numerical expressions immediately available to operate on the data and save results back to the filesystem. Furthermore, the data can be directly copied to / from the OS filesystem to persist it, and `goal` can transparently access data on remote systems through ssh. Furthermore, the [databrowser](databrowser) provides a fully interactive GUI for inspecting and plotting data.

* [tensorcore](tensorcore) provides core widgets for graphically displaying the `Tensor` and `Table` data, which are used in `datafs`.

* [tmath](tmath) implements all standard math functions on `tensor.Indexed` data, including the standard `+, -, *, /` operators. `goal` then calls these functions.

* [plot/plotcore](../plot/plotcore) supports interactive plotting of `Table` data.

* [bitslice](bitslice) is a Go slice of bytes `[]byte` that has methods for setting individual bits, as if it was a slice of bools, while being 8x more memory efficient. This is used for encoding null entries in  `etensor`, and as a Tensor of bool / bits there as well, and is generally very useful for binary (boolean) data.

* [stats](stats) implements a number of different ways of analyzing tensor and table data, including:
    - [cluster](cluster) implements agglomerative clustering of items based on [metric](metric) distance / similarity matrix data.
    - [convolve](convolve) convolves data (e.g., for smoothing).
    - [glm](glm) fits a general linear model for one or more dependent variables as a function of one or more independent variables. This encompasses all forms of regression.
    - [histogram](histogram) bins data into groups and reports the frequency of elements in the bins.
    - [metric](metric) computes similarity / distance metrics for comparing two tensors, and associated distance / similarity matrix functions, including PCA and SVD analysis functions that operate on a covariance matrix.
    - [stats](stats) provides a set of standard summary statistics on a range of different data types, including basic slices of floats, to tensor and table data. It also includes the ability to extract Groups of values and generate statistics for each group, as in a "pivot table" in a spreadsheet.

# Standard shapes

In general, **1D** refers to a flat, 1-dimensional list. There are various standard shapes of tensor data that different functions expect:

* **Flat, 1D**: this is the simplest data shape. For example, the [stats](stats) functions report summary statistics for all values of such data, across the one dimension. `Rows` views of this 1D data provide fine-grained filtering and sorting of all the data. Any `Tensor` can be accessed via a flat 1D index, which goes directly into the underlying Go slice for the basic types, and is appropriately (though somewhat expensively in some cases) indirected through the effective geometry in `Sliced` and `Rows` types.

* **Row, Cell 2D**: The outermost row dimension can be sorted, filtered in an `Rows` view, and the inner "cells" of data are organized in a simple flat 1D `SubSpace`, so they can be easily processed. In the [stats](stats) and [metric](metric) packages, 2+ dimensional data will be automatically re-shaped into this Row, Cell format, and processed as row-wise list of cell-wise patterns. For example, `stats` will aggregate each cell separately across rows, so you end up with the "average pattern" when you do `stats.Mean` for example. The [tmath](tmath) package, which defines binary functions, uses the [broadcasting](#broadcasting) logic to align n-dimensional data, and the row, cell structure provides a concrete simplification for thinking about how that works.

    A higher-dimensional tensor can also be re-shaped into this row, cell format by collapsing any number of additional outer dimensions into a longer, effective "row" index, with the remaining inner dimensions forming the cell-wise patterns. You can decide where to make the cut, and the `RowCellSplit` function makes it easy to create a new view of an existing tensor with this split made at a given dimension.

* **Matrix 2D**: For matrix algebra functions, a 2D tensor is treated as a standard row-major 2D matrix, which can be processed using `gonum` based matrix and vector operations, as in the [matrix](matrix) package.

* **Matrix 3+D**: For functions that specifically process 2D matricies, a 3+D shape can be used as well, which iterates over the outer dimensions to process the inner 2D matricies.

## Dynamic row sizing (e.g., for logs)

The `SetNumRows` function can be used to progressively increase the number of rows to fit more data, as is typically the case when logging data (often using a [table](table)). You can set the row dimension to 0 to start -- that is (now) safe. However, for greatest efficiency, it is best to set the number of rows to the largest expected size first, and _then_ set it back to 0. The underlying slice of data retains its capacity when sized back down. During incremental increasing of the slice size, if it runs out of capacity, all the elements need to be copied, so it is more efficient to establish the capacity up front instead of having multiple incremental re-allocations.

# Cheat Sheet

`ix` is the `Rows` tensor for these examples:

## Tensor Access

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
// The row index will be indirected through any `Indexes` present on the Rows view.
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
// returns a flat, 1D Rows view into n-dimensional tensor values at 
// given row. This is used in compute routines that operate generically
// on the entire row as a flat pattern.
ci := tensor.Cells1D(ix, 5)
```

### Full N-dimensional Indexes

```Go
// for 3D data
val := ix.Float(3,2,1)
```

# `Tensor` vs. Python NumPy

The [Goal](../goal) language provides a reasonably faithful translation of NumPy `array` syntax into the corresponding Go tensor package implementations. For those already familiar with NumPy, it should mostly "just work", but the following provides a more in-depth explanation for how the two relate, and when you might get different results.

## Basic and Advanced Indexing

NumPy distinguishes between _basic indexing_ (using a single index or sliced ranges of indexes along each dimension) versus _advanced indexing_ (using an array of indexes or bools) indexing. Basic indexing returns a **view** into the original data (where changes to the view directly affect the underlying type), while advanced indexing returns a **copy**.

However, rather confusingly (per this [stack overflow question](https://stackoverflow.com/questions/15691740/does-assignment-with-advanced-indexing-copy-array-data)), you can do direct assignment through advanced indexing (more on this below):
```Python
a[np.array([1,2])] = 5  # or:
a[a > 0.5] = 1          # boolean advanced indexing
```

Although powerful, the semantics of all of this is a bit confusing. In the `tensor` package, we provide what are hopefully more clear and concrete _view_ types that have well-defined semantics, and cover the relevant functionality, while hopefully being a bit easier to reason with. These were described at the start of this README.  The correspondence to NumPy indexing is as follows:

* Basic indexing by individual integer index coordinate values is supported by the basic `Number`, `String`, `Bool` value `Tensor`s.  For example, `Float(3,1,2)` returns the value at the given coordinates.  The `Sliced` (and `Rows`) view then completes the basic indexing with arbitrary reordering and filtering along entire dimension values.

* The `Masked` view corresponds to the NumPy _advanced_ indexing using a same-shape boolean mask, although in the NumPy case it makes a copy (although practically it is widely used for direct assignment as shown above.) Critically, you can always extract just the `true` values from a Masked view by using the `AsValues` method on the view, which returns a 1D tensor of those values, similar to what the boolean advanced indexing produces in NumPy. In addition, the `CloneIndexes` method returns a 1D list of indexes of the `true` values, which can be used for the `Indexed` view.
    
* The `Indexed` view corresponds to the array-based advanced indexing case in NumPy, but again it is a view, not a copy, so the assignment semantics are as expected from a view (and how NumPy behaves some of the time). Note that the NumPy version uses `n` separate index tensors, where each such tensor specifies the value of a corresponding dimension index, and all such tensors _must have the same shape_; that form can be converted into the single Indexes form with a utility function.  Also, NumPy advanced indexing has a somewhat confusing property where it de-duplicates index references during some operations, such that `+=1` only increments +1 even when there are multiple elements in the view. The tensor version does not implement that special case, due to its direct view semantics.

To reiterate, all view tensors have a `AsValues` function, equivalent to the `copy` function in NumPy, which turns the view into a corresponding basic concrete value Tensor, so the copy semantics of advanced indexing (modulo the direct assignment behavior) can be achieved when assigning to a new variable.

## Broadcasting

The NumPy concept of [broadcasting](https://numpy.org/doc/stable/user/basics.broadcasting.html) is critical for flexibly defining the semantics for how functions taking two n-dimensional Tensor arguments behave when they have different shapes.  If both tensors are 1D and the same length, then a simple matched iteration over both can take place. However, the broadcasting logic defines what happens when there is a systematic relationship between the two, enabling powerful (but sometimes difficult to understand) computations to be specified.

The following examples demonstrate the logic:

Innermost dimensions that match in dimension are iterated over as you'd expect:
```
Image  (3d array): 256 x 256 x 3
Scale  (1d array):             3
Result (3d array): 256 x 256 x 3
```

Anything with a dimension size of 1 will match against any other sized dimension:
```
A      (4d array):  8 x 1 x 6 x 1
B      (3d array):      7 x 1 x 5
Result (4d array):  8 x 7 x 6 x 5
```
In the innermost dimension here, the single value in A acts like a "scalar" in relationship to the 5 values in B along that same dimension, operating on each one in turn.

Any non-1 mismatch represents an error:
```
A      (2d array):      2 x 1
B      (3d array):  8 x 4 x 3 # second from last dimensions mismatched
```

Computationally, the broadcast logic is straightforward to implement, in terms of computing the resulting shape. Any missing outer dimensions can be replaced with 1s, and the full 1D product index on the result shape can be applied directly to the source shapes, using the modulo operator on length so it just repeatedly samples as needed.

# History

This package was originally developed as [etable](https://github.com/emer/etable) as part of the _emergent_ software framework. It always depended on the GUI framework that became Cogent Core, and having it integrated within the Core monorepo makes it easier to integrate updates, and also makes it easier to build advanced data management and visualization applications. For example, the [plot/plotcore](../plot/plotcore) package uses the `Table` to support flexible and powerful plotting functionality.

It was completely rewritten in Sept 2024 to use a single data type (`tensor.Indexed`) and call signature for compute functions taking these args, to provide a simple and efficient data processing framework that greatly simplified the code and enables the [goal](../goal) language to directly transpile simplified math expressions into corresponding tensor compute code.


