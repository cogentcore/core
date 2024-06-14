# Tensor

Tensor and related sub-packages provide a simple yet powerful framework for representing n-dimensional data of various types, providing similar functionality to the widely used `numpy` and `pandas` libraries in python, and the commercial MATLAB framework.

* [table](table) organizes multiple Tensors as columns in a data `Table`, aligned by a common row dimension as the outer-most dimension of each tensor.  Because the columns are tensors, each cell (value associated with a given row) can also be n-dimensional, allowing efficient representation of patterns and other high-dimensional data.  Furthermore, the entire column is organized as a single contiguous slice of data, so it can be efficiently processed.  The `table` package also has an `IndexView` that provides an indexed view into the rows of the table for highly efficient filtering and sorting of data.

    Data that is encoded as a slice of `struct`s can be bidirectionally converted to / from a Table, which then provides more powerful sorting, filtering and other functionality, including the plotcore.

* [tensorcore](tensorcore) provides core widgets for the `Tensor` and `Table` data.

* [stats](stats) implements a number of different ways of analyzing tensor and table data.

* [plot/plotcore](../plot/plotcore) supports interactive plotting of `Table` data.


# History

This package was originally developed as [etable](https://github.com/emer/etable) as part of the _emergent_ software framework.  It always depended on the GUI framework that became Cogent Core, and having it integrated within the Core monorepo makes it easier to integrate updates, and also makes it easier to build advanced data management and visualization applications.  For example, the [plot/plotcore](../plot/plotcore) package uses the `Table` to support flexible and powerful plotting functionality.

