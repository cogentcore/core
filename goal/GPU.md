# Goal GPU support

The use of massively parallel _Graphical Processsing Unit_ (_GPU_) hardware has revolutionized machine learning and other fields, producing many factors of speedup relative to traditional _CPU_ (_Central Processing Unit_) computation. However, there are numerous challenges for supporting GPU-based computation, relative to the more flexible CPU coding.

Goal provides a solution to these challenges that enables the same Go-based code to work efficiently and reasonably naturally on both the GPU and CPU (i.e., standard Go execution), for maximum portability. The ability to run the same code on both types of hardware is also critical for debugging the otherwise difficult to debug GPU version, and avoiding bugs in the first place by catching them first on the CPU, while providing known correct comparison results.

The two most important challenges are:

* The GPU _has its own separate memory space_ that needs to be synchronized explicitly and bidirectionally with the standard CPU memory (this is true programmatically even if at a hardware level there is shared memory).

* Computation must be organized into discrete chunks that can be computed efficiently in parallel, and each such chunk of computation lives in its own separate _kernel_ (_compute shader_) in the GPU, as an entirely separate, self-contained program, operating on _global variables_ that define the entire memory space of the computation.

To be maximally efficient, both of these factors must be optimized, such that:

* The bidirectional syncing of memory between CPU and GPU should be minimized, because such transfers incur a significant overhead.

* The overall computation should be broken down into the _largest possible chunks_ to minimize the number of discrete kernel runs, each of which incurs significant overhead.

Thus, it is unfortunately _highly inefficient_ to implement GPU-based computation by running each elemental vectorizable tensor operation (add, multiply, etc) as a separate GPU kernel, with its own separate bidirectional memory sync, even though that is a conceptually attractive and simple way to organize GPU computation, with minimal disruption relative to the CPU model.

The [JAX](https://github.com/jax-ml/jax) framework in Python provides one solution to this situation, optimized for neural network machine learning uses, by imposing strict _functional programming_ constraints on the code you write (i.e., all functions must be _read-only_), and leveraging those to automatically combine elemental computations into larger parallelizable chunks, using a "just in time" (_jit_) compiler.

We take a different approach, which is much simpler implementationally but requires a bit more work from the developer, which is to provide tools that allow _you_ to organize your computation into kernel-sized chunks according to your knowledge of the problem, and transparently turn that code into the final CPU and GPU programs.

In many cases, a human programmer can most likely out-perform the automatic compilation process, by knowing the full scope of what needs to be computed, and figuring out how to package it most efficiently per the above constraints. In the end, you get maximum efficiency and complete transparency about exactly what is being computed, perhaps with fewer "gotcha" bugs arising from all the magic happening under the hood, but again it may take a bit more work to get there.

The role of Goal is to allow you to express the full computation in the clear, simple, Go language, using intuitive data structures that minimize the need for additional boilerplate to run efficiently on CPU and GPU. This ability to write a single codebase that runs efficiently on CPU and GPU is similar to the [SYCL](https://en.wikipedia.org/wiki/SYCL) framework (and several others discussed on that wikipedia page), which builds on [OpenCL](https://en.wikipedia.org/wiki/OpenCL), both of which are based on the C / C++ programming language.

In addition to the critical differences between Go and C++ as languages, Goal targets only one hardware platform: WebGPU (via our [gpu](../gpu) package), so it is more specifically optimized for this use-case. Furthermore, SYCL and other approaches require you to write GPU-like code that can also run on the CPU (with lots of explicit fine-grained memory and compute management), whereas Goal provides a more natural CPU-like programming model, while imposing some stronger constraints that encourage more efficient implementations.

The [gosl](gosl) (_Go shader language_) package within Goal does the heavy lifting of translating Go code into WGSL shader language code that can run on the WebGPU, and generally manages most of the gpu-specific functionality.  It has various important functions defined in the `gosl.` package space, and a number of `//gosl:` comment directives described below, that make everything work.

Meanwhile, the `goal build` command provides an outer-loop of orchestration and math code transpiling prior to handing off to gosl to run on the relevant files.  

For example, `gosl.UseCPU()` causes subsequent execution to use CPU code, while `gosl.UseGPU()` causes it to use GPU kernels. Other `gosl` calls configure and activate the GPU during an initialization step.

## Overall Code Organization

First, we assume the scope is a single Go package that implements a set of computations on some number of associated data representations. The package will likely contain a lot of CPU-only Go code that manages all the surrounding infrastructure for the computations, in terms of creating and configuring the data in memory, visualization, i/o, etc.

The GPU-specific computation is organized into some (hopefully small) number of **kernel** functions, that are called using a special `parallel` version of a `for range` loop:

```Go
for i := range parallel(data) {
    MyCompute(i)
}
```

Where the `parallel` function is a special Goal keyword that triggers GPU vs. CPU execution, depending on prior configuration setup, and `MyCompute` is a kernel function. The `i` index effectively iterates over the range of the values of the `tensor` variable `data` (using specific dimension(s) with optional parameter(s)), with the GPU version launching the kernel on the GPU.

We assume that multiple kernels will in general be required, and that there is likely to be a significant amount of shared code infrastructure across these kernels.

> We support multiple CPU-based kernels within a single Go package directory.

Even though the GPU kernels must each be compiled separately into a single distinct WGSL _shader_ file that is run under WebGPU, they can `import` a shared codebase of files, and thus replicate the same overall shared code structure as the CPU versions.

The GPU code can only handle a highly restricted _subset_ of Go code, with data structures having strict alignment requirements, and no `string` or other composite variable-length data structures (maps, slices etc). Thus, the [gosl](gosl) package recognizes `//gosl:start` and `//gosl:end` comment directives surrounding the GPU-safe (and relevant) portions of the overall code. Any `.go` or `.goal` file can contribute GPU relevant code, including in other packages, and the gosl system automatically builds a shadow package-based set of `.wgsl` files accordingly.

> Each kernel must be written in a separate `.goal` file, marked with the `//gosl:kernel` directive at the top. There must be a "main" function entry point for each kernel, marked with the //gosl:main directive, which takes the index argument as shown in the code example above.

For CPU (regular Go) mode, the parallel `for range` loop as shown above translates into a  `tensor.VectorizeThreaded` call of the named Go function. For GPU mode, it launches the kernel on the GPU.

## Memory Organization

Perhaps the strongest constraints for GPU programming stem from the need to organize and synchronize all the memory buffers holding the data that the GPU kernel operates on. Furthermore, within a GPU kernel, the variables representing this data are _global variables_, which is sensible given the standalone nature of each kernel.

> To provide a common programming environment, all GPU-relevant variables must be Go global variables.

Thus, names must be chosen appropriately for these variables, given their global scope within the Go package. The specific _values_ for these variables can be dynamically set in an easy way, but the variables themselves are global.

Within the [gpu](../gpu) framework, each `ComputeSystem` defines a specific organization of such GPU buffer variables, and maximum efficiency is achieved by minimizing the number of such compute systems, and associated memory buffers. Each system also encapsulates the associated kernel shaders that operate on the associated memory data, so

> Kernels and variables both must be defined within a specific system context.

The following comment directive can be used in any kernel file to specify which system it uses, and there is an initial `default` system that is used if none is ever specified.
```Go
//gosl:system <system name>
```

To define the global variables for each system, use a standard Go `var` block declaration (with optional system name qualifier):
```Go
var ( //gosl:vars [system name]

    //gosl:group -uniform Params

    // Layer-level parameters
    Layers   []LayerParam // note: struct with appropriate memory alignment

    // Path-level parameters
    Paths    []PathParam  

    //gosl:group Units

    // Unit state values
    Units    tensor.Float32
    
    // Synapse weight state values
    Weights  tensor.Float32
)
```

The `//gosl:vars` directive flags this block of variables as GPU-accessible variables, which will drive the automatic generation of [gpu](../gpu) code to define these variables for the current (named) system, and to declare them in each kernel associated with that system.

The `//gosl:group` directives specify groups of variables, which generally should have similar memory syncing behavior, as documented in the [gpu](../gpu) system.

### datafs mapping

The grouped global variables can be mapped directly to a corresponding [datafs](../tensor/datafs) directory, which provides direct accessibility to this data within interactive Goal usage. Further, different sets of variable values can be easily managed by saving and loading different such directories.

```Go
    gosl.ToDataFS("path/to/dir" [, system]) // set datafs items in given path to current global vars
    
    gosl.FromDataFS("path/to/dir" [,system]) // set global vars from given datafs path
```

These and all such `gosl` functions use the current system if none is explicitly specified, which is settable using the `gosl.SetSystem` call. Any given variable can use the `get` or `set` Goal math mode functions directly.

## Memory syncing

It is up to the programmer to manage the syncing of memory between the CPU and the GPU, using simple `gosl` wrapper functions that manage all the details:

```Go
    gosl.ToGPU(varName...) // sync current contents of given GPU variable(s) from CPU to GPU

    gosl.FromGPU(varName...) // sync back from GPU to CPU
```

These are no-ops if operating in CPU mode.

## Memory access

In general, all global GPU variables will be arrays (slices) or tensors, which are exposed to the GPU as an array of floats.

The tensor-based indexing syntax in Goal math mode transparently works across CPU and GPU modes, and is thus the preferred way to access tensor data.

It is critical to appreciate that none of the other convenient math-mode operations will work as you expect on the GPU, because:

> There is only one outer-loop, kernel-level parallel looping operation allowed at a time.

You cannot nest multiple such loops within each other. A kernel cannot launch another kernel. Therefore, as noted above, you must directly organize your computation to maximize the amount of parallel computation happening wthin each such kernel call.

> Therefore, tensor indexing on the GPU only supports direct index values, not ranges.

Furthermore:

> Pointer-based access of global variables is not supported in GPU mode.

You have to use _indexes_ into arrays exclusively. Thus, some of the data structures you may need to copy up to the GPU include index variables that determine how to access other variables. TODO: do we need helpers for any of this?

## Optimization

can run naga on wgsl code to get wgsl code out, but it doesn't seem to do much dead code elimination: https://github.com/gfx-rs/wgpu/tree/trunk/naga

```
naga --compact gpu_applyext.wgsl tmp.wgsl
```

https://github.com/LucentFlux/wgsl-minifier does radical minification but the result is unreadable so we don't know if it is doing dead code elimination.  in theory it is just calling naga --compact for that.

