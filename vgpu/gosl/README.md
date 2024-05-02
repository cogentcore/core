# gosl

`gosl` implements _Go as a shader language_ for GPU compute shaders (using [Vulkan](https://www.vulkan.org)), **enabling standard Go code to run on the GPU**.

The relevant subsets of Go code are specifically marked using `//gosl` comment directives, and this code must only use basic expressions and concrete types that will compile correctly in a shader (see [Restrictions](#restrictions) below).  Method functions and pass-by-reference pointer arguments to `struct` types are supported and incur no additional compute cost due to inlining (see notes below for more detail).

A large and complex biologically-based neural network simulation framework called [axon](https://github.com/emer/axon) has been implemented using `gosl`, allowing 1000's of lines of equations and data structures to run through standard Go on the CPU, and accelerated significantly on the GPU.  This allows efficient debugging and unit testing of the code in Go, whereas debugging on the GPU is notoriously difficult.

`gosl`  converts Go code to HLSL, and then uses the DirectX shader compiler [dxc](https://github.com/microsoft/DirectXShaderCompiler) to compile that into an `.spv` SPIR-V file that can be loaded into a Vulkan GPU compute shader.  `dxc` is included with the [Vulkan SDK](https://vulkan.lunarg.com/sdk/home), which is probably the easiest way to get it installed.  [glslc](https://github.com/google/shaderc) can also compile the HLSL code, but `dxc` is a better option at this point.

See [examples/basic](examples/basic) and [rand](examples/rand) for examples, using the [vgpu](../../vgpu) Vulkan-based GPU compute shader system.  It is also possible in principle to use gosl to generate shader files for any other GPU application, but this has not been tested.

You must also install `goimports` which is used on the extracted subset of Go code, to get the imports right:
```bash
$ go install golang.org/x/tools/cmd/goimports@latest
```

To install the `gosl` command, do:
```bash
$ go install cogentcore.org/core/vgpu/gosl@latest
```

In your Go code, use these comment directives:

```
//gosl start: <filename>

< Go code to be translated >

//gosl end: <filename>
```

to bracket code to be processed.  The resulting converted code is copied into a `shaders` subdirectory created under the current directory where the `gosl` command is run, using the filenames specified in the comment directives.  Each such filename should correspond to a complete shader program (i.e., a "kernel"), or a file that can be included into other shader programs.  Code is appended to the target file names in the order of the source .go files on the command line, so multiple .go files can be combined into one resulting HLSL file.

HLSL specific code, e.g., for the `main` compute function or to specify `#include` files, can be included either by specifying files with a `.hlsl` extension as arguments to the `gosl` command, or by using a `//gosl hlsl` comment directive as follows:
```
//gosl hlsl: <filename>

// <HLSL shader code to be copied>

//gosl end: <filename>
```
where the HLSL shader code is commented out in the .go file -- it will be copied into the target filename and uncommented.  The HLSL code can be surrounded by `/*` `*/` comment blocks (each on a separate line) for multi-line code (though using a separate `.hlsl` file is preferable in this case). 

For `.hlsl` files, their filename is used to determine the `shaders` destination file name, and they are automatically appended to the end of the corresponding `.hlsl` file generated from the `Go` files -- this is where the `main` function and associated global variables should be specified.

**IMPORTANT:** all `.go`, `.hlsl`, and `.spv` files are removed from the `shaders` directory prior to processing to ensure everything there is current -- always specify a different source location for any custom `.hlsl` files that are included.

# Usage

	gosl [flags] [path ...]

The flags are:

    -exclude string
    	comma-separated list of names of functions to exclude from exporting to HLSL (default "Update,Defaults")
    -out string
    	output directory for shader code, relative to where gosl is invoked (default "shaders")
    -keep
    	keep temporary converted versions of the source files, for debugging

Note: any existing `.go` files in the output directory will be removed prior to processing, because the entire directory is built to establish all the types, which might be distributed across multiple files.  Any existing `.hlsl` files with the same filenames as those extracted from the `.go` files will be overwritten.  Otherwise, you can maintain other custom `.hlsl` files in the `shaders` directory, although it is recommended to treat the entire directory as automatically generated, to avoid any issues.
    
`gosl` path args can include filenames, directory names, or Go package paths (e.g., `cogentcore.org/core/math32/fastexp.go` loads just that file from the given package) -- files without any `//gosl` comment directives will be skipped up front before any expensive processing, so it is not a problem to specify entire directories where only some files are relevant.  Also, you can specify a particular file from a directory, then the entire directory, to ensure that a particular file from that directory appears first -- otherwise alphabetical order is used.  `gosl` ensures that only one copy of each file is included.
  
Any `struct` types encountered will be checked for 16-byte alignment of sub-types and overall sizes as an even multiple of 16 bytes (4 `float32` or `int32` values), which is the alignment used in HLSL and glsl shader languages, and the underlying GPU hardware presumably.  Look for error messages on the output from the gosl run.  This ensures that direct byte-wise copies of data between CPU and GPU will be successful.  The fact that `gosl` operates directly on the original CPU-side Go code uniquely enables it to perform these alignment checks, which are otherwise a major source of difficult-to-diagnose bugs.

# Restrictions    

In general shader code should be simple mathematical expressions and data types, with minimal control logic via `if`, `for` statements, and only using the subset of Go that is consistent with C.  Here are specific restrictions:

## Types

* Can only use `float32`, `[u]int32`, and their 64 bit versions for basic types, and `struct` types composed of these same types -- no other Go types (i.e., `map`, slices, `string`, etc) are compatible.  There are strict alignment restrictions on 16 byte (e.g., 4 `float32`'s) intervals that are enforced via the `alignsl` sub-package.

* Use `slbool.Bool` instead of `bool` -- it defines a Go-friendly interface based on a `int32` basic type.  Using a `bool` in a `uniform` `struct` causes an obscure `glslc` compiler error: `shaderc: internal error: compilation succeeded but failed to optimize: OpFunctionCall Argument <id> '73[%73]'s type does not match Function`  

* Alignment and padding of `struct` fields is key -- this is automatically checked by `gosl`.

* HLSL does not support enum types, but standard go `const` declarations will be converted.  Use an `int32` or `uint32` data type.  It will automatically deal with the simple incrementing `iota` values, but not more complex cases.  Also, for bitflags, define explicitly, not using `bitflags` package.

* HLSL does not do multi-pass compiling, so all dependent types must be specified *before* being used in other ones, and this also precludes referencing the *current* type within itself.  todo: can you just use a forward declaration?

* HLSL does not provide the same auto-init-to-zero for declared variables -- safer to initialize directly:
```Go
    val := float32(0) // guaranteed 0 value
    var val float32 // not guaranteed to be 0!  avoid!
```    

## Syntax

* Cannot use multiple return values, or multiple assignment of variables in a single `=` expression.

* *Can* use multiple variable names with the same type (e.g., `min, max float32`) -- this will be properly converted to the more redundant C form with the type repeated.

## Random numbers: slrand

See [slrand](https://github.com/emer/gosl/v2/tree/main/slrand) for a shader-optimized random number generation package, which is supported by `gosl` -- it will convert `slrand` calls into appropriate HLSL named function calls.  `gosl` will also copy the `slrand.hlsl` file, which contains the full source code for the RNG, into the destination `shaders` directory, so it can be included with a simple local path:

```Go
//gosl: hlsl mycode
// #include "slrand.hlsl"
//gosl: end mycode
```

# Performance

With sufficiently large N, and ignoring the data copying setup time, around ~80x speedup is typical on a Macbook Pro with M1 processor.  The `rand` example produces a 175x speedup!

# Implementation / Design Notes

HLSL is very C-like and provides a much better target for Go conversion than glsl.  See `examples/basic/shaders/basic_nouse.glsl` vs the .hlsl version there for the difference.  Only HLSL supports methods in a struct, and performance is the same as writing the expression directly -- it is suitably [inlined](https://learn.microsoft.com/en-us/windows/win32/direct3dhlsl/dx-graphics-hlsl-function-syntax).

While there aren't any pointers allowed in HLSL, the inlining of methods, along with the use of the `inout` [InputModifier](https://learn.microsoft.com/en-us/windows/win32/direct3dhlsl/dx-graphics-hlsl-function-parameters), effectively supports pass-by-reference.  The [stackoverflow](https://stackoverflow.com/questions/28527622/shaders-function-parameters-performance/28577878#28577878) on this is a bit unclear but the basic example demonstrates that it all goes through.

# Links

Key docs for HLSL as compute shaders:

* https://github.com/microsoft/DirectXShaderCompiler/blob/main/docs/SPIR-V.rst

* https://www.saschawillems.de/blog/2020/05/23/shaders-for-vulkan-samples-now-also-available-in-hlsl/


