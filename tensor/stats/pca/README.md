# pca

This performs principal component's analysis and associated covariance matrix computations, operating on `table.Table` or `tensor.Tensor` data, using the [gonum](https://github.com/gonum/gonum) matrix interface.

There is support for the SVD version, which is much faster and produces the same results, with options for how much information to compute trading off with compute time.


