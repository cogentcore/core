# basic

This example just does some basic calculations on data structures and reports the time difference between the CPU and GPU.  Getting about a 20x speedup on a Macbook Pro M3 Max.

# Building

There is a `//go:generate` comment directive in `main.go` that calls `gosl` on the relevant files, so you can do `go generate` followed by `go build` to run it.  There is also a `Makefile` with the same `gosl` command, so `make` can be used instead of go generate.

The generated files go into the `shaders/` subdirectory.


