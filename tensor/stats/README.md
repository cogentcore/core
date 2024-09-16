# stats

There are several packages here for operating on vector, [tensor](../), and [table](../table) data, for computing standard statistics and performing related computations, such as normalizing the data.

* [cluster](cluster) implements agglomerative clustering of items based on [metric](metric) distance / similarity matrix data.
* [convolve](convolve) convolves data (e.g., for smoothing).
* [glm](glm) fits a general linear model for one or more dependent variables as a function of one or more independent variables.  This encompasses all forms of regression.
* [histogram](histogram) bins data into groups and reports the frequency of elements in the bins.
* [metric](metric) computes similarity / distance metrics for comparing two tensors, and associated distance / similarity matrix functions, including PCA and SVD analysis functions that operate on a covariance matrix.
* [stats](stats) provides a set of standard summary statistics on a range of different data types, including basic slices of floats, to tensor and table data.  It also includes the ability to extract Groups of values and generate statistics for each group, as in a "pivot table" in a spreadsheet.

