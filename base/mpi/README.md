# mpi

`mpi` contains Go wrappers around the MPI message passing interface for distributed memory computation.  This has no other dependencies and uses code generation to provide support for all Go types.

The default without any build tags is to build a dummy version that has 1 proc of rank 0 always, and nop versions of all the methods.

There are two supported versions of mpi, selected using the corresponding build tag:
* `mpi` = [open mpi](https://www.open-mpi.org/), installed with a [pkg-config](https://en.wikipedia.org/wiki/Pkg-config) named `ompi`
* `mpich` = [mpich](https://www.mpich.org/), installed with a `pkg-config` named `mpich`

It is possible to add other versions by adding another build tag and adding the corresponding line to the `#cgo` directives in the relevant files in the `mpi` package.

For example, to build with open mpi, build your program like this:
```sh
$ go build -tags mpi
```

# Install

On a mac, you can use `brew install open-mpi` or `brew install mpich` to install.  Corresponding package manager versions are presumably available on linux, and mpi is usually already supported on HPC clusters.

# Development

After updating any of the template files, you need to update the generated go files like so:
```bash
cd mpi
go install github.com/apache/arrow/go/arrow/_tools/tmpl@latest
make generate
```
