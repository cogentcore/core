# tensormpi: Message Passing Interface

The `tensormpi` package has methods to support use of MPI with tensor and table data structures, using the [mpi](../../base/mpi) package for Go mpi wrappers.

As documented in the [mpi](../../base/mpi) package, You must set the `mpi` or `mpich` build tag to actually have it build using the mpi library.  The default is to build a dummy version that has 1 proc of rank 0 always, and nop versions of all the methods.

The code here is not build tag conditionalized, but is harmless for the default 1 processor case.

Supported functionality:

* `GatherTensorRows`, `GatherTableRows`: Gathering `table.Table` and `tensor.Tensor` data across processors.

* `RandCheck` checks that the current random number generated across different processors is the same, which is often needed.

* `AllocN` allocates n items to process across mpi processors.

