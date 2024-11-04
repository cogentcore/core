# gosl: Go as a shader language

`gosl` implements _Go as a shader language_ for GPU compute shaders (using [WebGPU](https://www.w3.org/TR/webgpu/)), **enabling standard Go code to run on the GPU**.

`gosl` converts Go code to WGSL which can then be loaded directly into a WebGPU compute shader, using the [gpu](../../gpu) GPU compute shader system. It operates within the overall [Goal](../README.md) framework of an augmented version of the Go language. See the [GPU](../GPU.md) documentation for an overview of issues in GPU computation.

The relevant regions of Go code to be run on the GPU are tagged using the `//gosl:start` and `//gosl:end` comment directives, and this code must only use basic expressions and concrete types that will compile correctly in a GPU shader (see [Restrictions](#restrictions) below).  Method functions and pass-by-reference pointer arguments to `struct` types are supported and incur no additional compute cost due to inlining (see notes below for more detail).

See [examples/basic](examples/basic) and [rand](examples/rand) for complete working examples.

Although `gosl` is typically run via the `goal build` command, you can also run `gosl` directly.  Here's how to install the standalone `gosl` command:
```bash
$ go install cogentcore.org/core/goal/gosl@latest
```

# Usage

There are two critical elements for GPU-enabled code:

1. One or more [Kernel](#kernels) compute functions that take an _index_ argument and perform computations for that specific index of data, _in parallel_. **GPU computation is effectively just a parallel `for` loop**. On the GPU, each such kernel is implemented by its own separate compute shader code, and one of the main functions of `gosl` is to generate this code from the Go sources, in the automatically created `shaders/` directory.

2. [Global variables](#global-variables) on which the kernel functions _exclusively_ operate: all relevant data must be specifically copied from the CPU to the GPU and back. As explained in the [GPU](../GPU.md) docs, each GPU compute shader is effectively a _standalone_ program operating on these global variables. To replicate this environment on the CPU, so the code is transferrable, we need to make these variables global in the CPU (Go) environment as well.

`gosl` generates a file named `gosl.go` in your package directory that initializes the GPU with all of the global variables, and functions for running the kernels and syncing the gobal variable data back and forth between the CPu and GPU.

## Kernels

Each distinct compute kernel must be tagged with a `//gosl:kernel` comment directive, as in this example:
```Go
// Compute does the main computation.
func Compute(i uint32) { //gosl:kernel
	Params[0].IntegFromRaw(int(i))
}
```

The kernel functions receive a `uint32` index argument, and use this to index into the global variables containing the relevant data. Typically the kernel code itself just calls other relevant function(s) using the index, as in the above example. Critically, _all_ of the data that a kernel function ultimately depends on must be contained with the global variables, and these variables must have been sync'd up to the GPU from the CPU prior to running the kernel (more on this below).

In the CPU mode, the kernel is effectively run in a `for` loop like this:
```Go
	for i := range n {
		Compute(uint32(i))
	}
```
A parallel goroutine-based mechanism is actually used, but conceptually this is what it does, on both the CPU and the GPU. To reiterate: **GPU computation is effectively just a parallel for loop**.

## Global variables

The global variables on which the kernels operate are declared in the usual Go manner, as a single `var` block, which is marked at the top using the `//gosl:vars` comment directive:

```Go
//gosl:vars
var (
	// Params are the parameters for the computation.
	//gosl:read-only
	Params []ParamStruct

	// Data is the data on which the computation operates.
	// 2D: outer index is data, inner index is: Raw, Integ, Exp vars.
	//gosl:dims 2
	Data tensor.Float32
)
```

All such variables must be either:
1. A `slice` of GPU-alignment compatible `struct` types, such as `ParamStruct` in the above example.
2. A `tensor` of a GPU-compatible elemental data type (`float32`, `uint32`, or `int32`), with the number of dimensions indicated by the `//gosl:dims <n>` tag as shown above.

You can also just declare a slice of elemental GPU-compatible data values such as `float32`, but it is generally preferable to use the tensor instead.

### Tensor data

On the GPU, the tensor data is represented using a simple flat array of the basic data type, with the _strides_ for each dimension encoded in the first `n` elements. `gosl` automatically generates the appropriate indexing code using these strides (which is why the number of dimensions is needed).

The tensor must be initialized using this special [sltensor](sltensor) function to encode the stride values in the "header" section of the tensor data:
```Go
	sltensor.SetShapeSizes(&Data, n, 3) // critically, makes GPU compatible Header with strides
```

### Systems and Groups

Each kernel belongs to a `gpu.ComputeSystem`, and each such system has one specific configuration of memory variables. In general, it is best to use a single set of global variables, and perform as much of the computation as possible on this set of variables, to minimize the number of memory transfers. However, if necessary, multiple systems can be defined, using an optional additional system name argument for the `args` and `kernel` tags.

In addition, the vars can be organized into _groups_, which generally should have similar memory syncing behavior, as documented in the [gpu](../gpu) system.

Here's an example with multiple groups:
```Go
//gosl:vars [system name]
var (
    // Layer-level parameters
    //gosl:group -uniform Params
    Layers   []LayerParam // note: struct with appropriate memory alignment

    // Path-level parameters
    Paths    []PathParam  

    // Unit state values
    //gosl:group Units
    Units    tensor.Float32
    
    // Synapse weight state values
    Weights  tensor.Float32
)
```

## Memory syncing

Each global variable gets an automatically-generated `*Var` enum (e.g., `DataVar` for global variable named `Data`), that used for the memory syncing functions, to make it easy to specify any number of such variables to sync, which is by far the most efficient. All of this is in the generated `gosl.go` file. For example:

```Go
	ToGPU(ParamsVar, DataVar)
```

Specifies that the current contents of `Params` and `Data` are to be copied up to the GPU, which is guaranteed to complete by the time the next kernel run starts, within a given system.

## Kernel running

As with memory transfers, it is much more efficient to run multiple kernels in sequence, all operating on the current data variables, followed by a single sync of the updated global variable data that has been computed. Thus, there are separate functions for specifying the kernels to run, followed by a single "Done" function that actually submits the entire batch of kernels, along with memory sync commands to get the data back from the GPU. For example:

```Go
    RunCompute1(n)
    RunCompute2(n)
    ...
    RunDone(Data1Var, Data2Var) // launch all kernels and get data back to given vars
```

For CPU mode, `RunDone` is a no-op, and it just runs each kernel during each `Run` command.

It is absolutely essential to understand that _all data must already be on the GPU_ at the start of the first Run command, and that any CPU-based computation between these calls is completely irrelevant for the GPU. Thus, it typically makes sense to just have a sequence of Run commands grouped together into a logical unit, with the relevant `ToGPU` calls at the start and the final `RunDone` grabs everything of relevance back from the GPU.

## GPU relevant code taggng

In a large GPU-based application, you should organize your code as you normally would in any standard Go application, distributing it across different files and packages. The GPU-relevant parts of each of those files can be tagged with the gosl tags:
```
//gosl:start

< Go code to be translated >

//gosl:end
```
to make this code available to all of the shaders that are generated.

Use the `//gosl:import "package/path"` directive to import GPU-relevant code from other packages, similar to the standard Go import directive. It is assumed that many other Go imports are not GPU relevant, so this separate directive is required.

`gosl` automatically includes _all_ tagged code with each shader, and lets the compiler sort out the subset of code that is actually relevant to each specific kernel. Ideally, we could do this with a pre-processing step that performs dead code elimination, but that does not appear to be functional yet.

**IMPORTANT:** all `.go` and `.wgsl` files are removed from the `shaders` directory prior to processing to ensure everything there is current -- always specify a different source location for any custom `.wgsl` files that are included.

# Command line usage

```
gosl [flags] 
```
    
The flags are:
```
  -debug
    	enable debugging messages while running
  -exclude string
    	comma-separated list of names of functions to exclude from exporting to WGSL (default "Update,Defaults")
  -keep
    	keep temporary converted versions of the source files, for debugging
  -out string
    	output directory for shader code, relative to where gosl is invoked -- must not be an empty string (default "shaders")
```

`gosl` always operates on the current directory, looking for all files with `//gosl:` tags, and accumulating all the `import` files that they include, etc.
  
Any `struct` types encountered will be checked for 16-byte alignment of sub-types and overall sizes as an even multiple of 16 bytes (4 `float32` or `int32` values), which is the alignment used in WGSL and glsl shader languages, and the underlying GPU hardware presumably.  Look for error messages on the output from the gosl run.  This ensures that direct byte-wise copies of data between CPU and GPU will be successful.  The fact that `gosl` operates directly on the original CPU-side Go code uniquely enables it to perform these alignment checks, which are otherwise a major source of difficult-to-diagnose bugs.

# Restrictions    

In general shader code should be simple mathematical expressions and data types, with minimal control logic via `if`, `for` statements, and only using the subset of Go that is consistent with C.  Here are specific restrictions:

* Can only use `float32`, `[u]int32` for basic types (`int` is converted to `int32` automatically), and `struct` types composed of these same types -- no other Go types (i.e., `map`, slices, `string`, etc) are compatible.  There are strict alignment restrictions on 16 byte (e.g., 4 `float32`'s) intervals that are enforced via the `alignsl` sub-package.

* WGSL does _not_ support 64 bit float or int.

* Use `slbool.Bool` instead of `bool` -- it defines a Go-friendly interface based on a `int32` basic type.

* Alignment and padding of `struct` fields is key -- this is automatically checked by `gosl`.

* WGSL does not support enum types, but standard go `const` declarations will be converted.  Use an `int32` or `uint32` data type.  It will automatically deal with the simple incrementing `iota` values, but not more complex cases.  Also, for bitflags, define explicitly, not using `bitflags` package, and use `0x01`, `0x02`, `0x04` etc instead of `1<<2` -- in theory the latter should be ok but in practice it complains.

* Cannot use multiple return values, or multiple assignment of variables in a single `=` expression.

* *Can* use multiple variable names with the same type (e.g., `min, max float32`) -- this will be properly converted to the more redundant form with the type repeated, for WGSL.

* `switch` `case` statements are _purely_ self-contained -- no `fallthrough` allowed!  does support multiple items per `case` however. Every `switch` _must_ have a `default` case.

* WGSL does specify that new variables are initialized to 0, like Go, but also somehow discourages that use-case.  It is safer to initialize directly:
```Go
    val := float32(0) // guaranteed 0 value
    var val float32 // ok but generally avoid
```    

* A local variable to a global `struct` array variable (e.g., `par := &Params[i]`) can only be created as a function argument. There are special access restrictions that make it impossible to do otherwise.

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

