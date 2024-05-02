# rand

This example tests the `slrand` random number generation functions.  The `go test` in `slrand` itself tests the Go version of the code against known values, and this example tests the GPU HLSL versions against the Go versions.

The output shows the first 5 sets of random numbers, with the CPU on the first line and the GPU on the second line.  If there are any `*` on any of the lines, then there is a difference between the two, and an error message will be reported at the bottom.

The total time to generate 10 million random numbers is shown at the end, comparing time on the CPU vs. GPU.  On a Mac Book Pro M3 Max laptop, the CPU took roughly _140 times_ longer to generate the random numbers than the GPU.

# Building

There is a `//go:generate` comment directive in `main.go` that calls `gosl` on the relevant files, so you can do `go generate` followed by `go build` to run it.  There is also a `Makefile` with the same `gosl` command, so `make` can be used instead of go generate.

The generated files go into the `shaders/` subdirectory.

Ignore the type alignment checking errors about Uint2 and Vector2 not being an even multiple of 16 bytes -- we have put in the necessary padding.

