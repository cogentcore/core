# AlignSL

alignsl performs 16-byte alignment and total size modulus checking of struct types to ensure HLSL (and GSL) compatibility.

Checks that `struct` sizes are an even multiple of 16 bytes (e.g., 4 float32's), fields are 32 or 64 bit types: [U]Int32, Float32, [U]Int64, Float64, and that fields that are other struct types are aligned at even 16 byte multiples.

It is called with a [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools/go/packages) `Package` that provides the `types.Sizes` and `Types.Scope()` to get the types.

The `CheckPackage` method checks all types in a `Package`, and returns an error if there are any violations -- this error string contains a full user-friendly warning message that can be printed.



