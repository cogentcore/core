# Tensor

Tensor and related sub-packages provide a simple yet powerful framework for representing n-dimensional data of various types, providing similar functionality to the widely used `numpy` and `pandas` libraries in python, and the commercial MATLAB framework.

* [table](table) organizes multiple Tensors as columns in a data `Table`, aligned by a common row dimension as the outer-most dimension of each tensor.  Because the columns are tensors, each cell (value associated with a given row) can also be n-dimensional, allowing efficient representation of patterns and other high-dimensional data.  Furthermore, the entire column is organized as a single contiguous slice of data, so it can be efficiently processed.  The `table` package also has an `IndexView` that provides an indexed view into the rows of the table for highly efficient filtering and sorting of data.

* [tensorview](tensorview) provides Core views of the `Tensor` and `Table` data.

* [stats](stats) implements a number of different ways of analyzing tensor and table data.
