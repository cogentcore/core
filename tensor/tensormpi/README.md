# tensormpi: Message Passing Interface

The `tensormpi` package has methods to support use of MPI with tensor and table data structures, using the [mpi](../../base/mpi) package for Go mpi wrappers.

You must set the `mpi` build tag to actually have it build using the mpi library -- the default is to build a dummy version that has 1 proc of rank 0 always, and nop versions of all the methods.

```bash
$ go build -tags mpi
```

* Gathering `table.Table` and `tensor.Tensor` data across processors.

* `AllocN` allocates n items to process across mpi processors.


