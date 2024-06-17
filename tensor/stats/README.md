# stats

There are several packages here for operating on vector, [tensor](../tensor), and [table](../table) data, for computing standard statistics and performing related computations, such as normalizing the data.

* [clust](clust) implements agglomerative clustering of items based on [simat](simat) similarity matrix data.
* [convolve](convolve) convolves data (e.g., for smoothing).
* [glm](glm) fits a general linear model for one or more dependent variables as a function of one or more independent variables.  This encompasses all forms of regression.
* [histogram](histogram) bins data into groups and reports the frequency of elements in the bins.
* [metric](metric) computes similarity / distance metrics for comparing two vectors
* [norm](norm) normalizes vector data
* [pca](pca) computes principal components analysis (PCA) or singular value decomposition (SVD) on correlation matricies, which is a widely-used way of reducing the dimensionality of high-dimensional data.
* [simat](simat) computes a similarity matrix for the [metric](metric) similarity of two vectors.
* [split](split) provides grouping and aggregation functions operating on `table.Table` data, e.g., like a "pivot table" in a spreadsheet.
* [stats](stats) provides a set of standard summary statistics on a range of different data types, including basic slices of floats, to tensor and table data.

