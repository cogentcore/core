# gosl: Go as a shader language

`gosl` implements _Go as a shader language_ for GPU compute shaders (using [WebGPU](https://www.w3.org/TR/webgpu/)), **enabling standard Go code to run on the GPU**.

`gosl` converts Go code to WGSL which can then be loaded directly into a WebGPU compute shader. It operates within the overall [Goal](../README.md) framework of an augmented version of the Go langauge. See the [GPU](../GPU.md) documentation for an overview. The `goal` command processes more compact math-mode expressions that 

The relevant subsets of Go code are specifically marked using `//gosl:` comment directives, and this code must only use basic expressions and concrete types that will compile correctly in a shader (see [Restrictions](#restrictions) below).  Method functions and pass-by-reference pointer arguments to `struct` types are supported and incur no additional compute cost due to inlining (see notes below for more detail).

See [examples/basic](examples/basic) and [rand](examples/rand) for examples, using the [gpu](../../gpu) GPU compute shader system.  It is also possible in principle to use gosl to generate shader files for any other WebGPU application, but this has not been tested.

You must also install `goimports` which is used on the extracted subset of Go code, to get the imports right:
```bash
$ go install golang.org/x/tools/cmd/goimports@latest
```

To install the `gosl` command, do:
```bash
$ go install cogentcore.org/core/gpu/gosl@latest
```

In your Go code, use these comment directives:
```
//gosl:start

< Go code to be translated >

//gosl:end
```

to bracket code to be processed for GPU. The resulting converted code is copied into a `shaders` subdirectory created under the current directory where the `gosl` command is run, using the filenames specified in the comment directives, or the name of the current file if not specified.

Use the `//gosl:import package` directive to include files from other packages, similar to the standard Go import directive. It is assumed that many other Go imports are not GPU relevant, so this separate directive is required.

Each such filename should correspond to a complete shader program (i.e., a "kernel"), or a file that can be included into other shader programs.  Code is appended to the target file names in the order of the source .go files on the command line, so multiple .go files can be combined into one resulting WGSL file.

WGSL specific code, e.g., for the `main` compute function or to specify `#include` files, can be included either by specifying files with a `.wgsl` extension as arguments to the `gosl` command, or by using a `//gosl:wgsl` comment directive as follows:
```
//gosl:wgsl <filename>

// <WGSL shader code to be copied>

//gosl:end <filename>
```
where the WGSL shader code is commented out in the .go file -- it will be copied into the target filename and uncommented.  The WGSL code can be surrounded by `/*` `*/` comment blocks (each on a separate line) for multi-line code (though using a separate `.wgsl` file is generally preferable in this case). 

For `.wgsl` files, their filename is used to determine the `shaders` destination file name, and they are automatically appended to the end of the corresponding `.wgsl` file generated from the `Go` files -- this is where the `main` function and associated global variables should be specified.

**IMPORTANT:** all `.go` and `.wgsl` files are removed from the `shaders` directory prior to processing to ensure everything there is current -- always specify a different source location for any custom `.wgsl` files that are included.

# Usage

```
gosl [flags] [path ...]
```
    
The flags are:
```
  -debug
    	enable debugging messages while running
  -exclude string
    	comma-separated list of names of functions to exclude from exporting to HLSL (default "Update,Defaults")
  -keep
    	keep temporary converted versions of the source files, for debugging
  -out string
    	output directory for shader code, relative to where gosl is invoked -- must not be an empty string (default "shaders")
```

`gosl` path args can include filenames, directory names, or Go package paths (e.g., `cogentcore.org/core/math32/fastexp.go` loads just that file from the given package) -- files without any `//gosl:` comment directives will be skipped up front before any expensive processing, so it is not a problem to specify entire directories where only some files are relevant.  Also, you can specify a particular file from a directory, then the entire directory, to ensure that a particular file from that directory appears first -- otherwise alphabetical order is used.  `gosl` ensures that only one copy of each file is included.
  
Any `struct` types encountered will be checked for 16-byte alignment of sub-types and overall sizes as an even multiple of 16 bytes (4 `float32` or `int32` values), which is the alignment used in WGSL and glsl shader languages, and the underlying GPU hardware presumably.  Look for error messages on the output from the gosl run.  This ensures that direct byte-wise copies of data between CPU and GPU will be successful.  The fact that `gosl` operates directly on the original CPU-side Go code uniquely enables it to perform these alignment checks, which are otherwise a major source of difficult-to-diagnose bugs.

# Restrictions    

In general shader code should be simple mathematical expressions and data types, with minimal control logic via `if`, `for` statements, and only using the subset of Go that is consistent with C.  Here are specific restrictions:

* Can only use `float32`, `[u]int32` for basic types (`int` is converted to `int32` automatically), and `struct` types composed of these same types -- no other Go types (i.e., `map`, slices, `string`, etc) are compatible.  There are strict alignment restrictions on 16 byte (e.g., 4 `float32`'s) intervals that are enforced via the `alignsl` sub-package.

* WGSL does _not_ support 64 bit float or int.

* Use `slbool.Bool` instead of `bool` -- it defines a Go-friendly interface based on a `int32` basic type.

* Alignment and padding of `struct` fields is key -- this is automatically checked by `gosl`.

* WGSL does not support enum types, but standard go `const` declarations will be converted.  Use an `int32` or `uint32` data type.  It will automatically deal with the simple incrementing `iota` values, but not more complex cases.  Also, for bitflags, define explicitly, not using `bitflags` package, and use `0x01`, `0x02`, `0x04` etc instead of `1<<2` -- in theory the latter should be ok but in practice it complains.

* Cannot use multiple return values, or multiple assignment of variables in a single `=` expression.

* *Can* use multiple variable names with the same type (e.g., `min, max float32`) -- this will be properly converted to the more redundant C form with the type repeated.

* `switch` `case` statements are _purely_ self-contained -- no `fallthrough` allowed!  does support multiple items per `case` however.

* TODO: WGSL does not do multi-pass compiling, so all dependent types must be specified *before* being used in other ones, and this also precludes referencing the *current* type within itself.  todo: can you just use a forward declaration?

* WGSL does specify that new variables are initialized to 0, like Go, but also somehow discourages that use-case.  It is safer to initialize directly:
```Go
    val := float32(0) // guaranteed 0 value
    var val float32 // ok but generally avoid
```    

## Other language features

* [tour-of-wgsl](https://google.github.io/tour-of-wgsl/types/pointers/passing_pointers/) is a good reference to explain things more directly than the spec.

* `ptr<function,MyStruct>` provides a pointer arg
* `private` scope = within the shader code "module", i.e., one thread.  
* `function` = within the function, not outside it.
* `workgroup` = shared across workgroup -- coudl be powerful (but slow!) -- need to learn more.

## Random numbers: slrand

See [slrand](https://github.com/emer/gosl/v2/tree/main/slrand) for a shader-optimized random number generation package, which is supported by `gosl` -- it will convert `slrand` calls into appropriate WGSL named function calls.  `gosl` will also copy the `slrand.wgsl` file, which contains the full source code for the RNG, into the destination `shaders` directory, so it can be included with a simple local path:

```Go
//gosl:wgsl mycode
// #include "slrand.wgsl"
//gosl:end mycode
```

# Performance

With sufficiently large N, and ignoring the data copying setup time, around ~80x speedup is typical on a Macbook Pro with M1 processor.  The `rand` example produces a 175x speedup!

# Implementation / Design Notes

# Links

Key docs for WGSL as compute shaders:

