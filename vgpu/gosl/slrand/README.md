# slrand

This package contains HLSL header files and matching Go code for various random number generation (RNG) functions.  The `gosl` tool will automatically copy the `slrand.hlsl` self-contained file into the destination `shaders` directory if the Go code contains `slrand.` prefix.  Here's how you include:

```Go
//gosl: hlsl mycode
// #include "slrand.hlsl"
//gosl: end mycode
```

`slrand` uses the [Philox2x32](https://github.com/DEShawResearch/random123) algorithm which is also available on CUDA on their [cuRNG](https://docs.nvidia.com/cuda/curand/host-api-overview.html) and in [Tensorflow](https://www.tensorflow.org/guide/random_numbers#general).  A recent [evaluation](https://www.mdpi.com/2079-3197/9/12/142#B69-computation-09-00142) showed it to be the fastest GPU RNG, which also passes the standards for statistical quality (e.g., BigCrush).  It is a counter based RNG, [CBRNG](https://en.wikipedia.org/wiki/Counter-based_random_number_generator_(CBRNG), where the random number is a direct function of the input state, with no other internal state.  For a useful discussion of other alternatives, see [reddit cpp thread](https://www.reddit.com/r/cpp/comments/u3cnkk/old_rand_method_faster_than_new_alternatives/).  The code is based on the D.E. Shaw [github](https://github.com/DEShawResearch/random123/blob/main/include/Random123/philox.h) implementation.

The key advantage of this algorithm is its *stateless* nature, where the result is a deterministic but highly nonlinear function of its two inputs:
```
    uint2 res = Philox2x32(inout uint2 counter, uint key);
```
where the HLSL `uint2` type is 2 `uint32` 32-bit unsigned integers.  For GPU usage, the `key` is always set to the unique element being processed (e.g., the index of the data structure being updated), ensuring that different numbers are generated for each such element, and the `counter` should be configured as a shared global value that is incremented after every RNG call.  For example, if 4 RNG calls happen within a given set of GPU code, each thread starts with the same starting `counter` value, which is passed around as a local `uint2` variable and incremented locally for each RNG.  Then, after all threads have been performed, the shared starting `counter` is incremented using `CounterAdd` by 4.

The `Float` and `Uint32` etc wrapper functions around Philox2x32 will automatically increment the counter var passed to it, using the `CounterIncr()` method that manages the two 32 bit numbers as if they are a full 64 bit uint.

The `slrand.Counter` struct provides a 16-byte aligned type for storing and incrementing the global counter.  The `Seed` method initializes the starting counter value by setting the Hi uint32 value to given seed, which thus provides a random sequence length of over 4 billion numbers within the Lo uint32 counter -- use more widely spaced seed values for longer unique sequences.

`gosl` will automatically translate the Go versions of the `slrand` package functions into their HLSL equivalents.

See the [axon](https://github.com/emer/gosl/v2/tree/main/examples/axon) and [rand](https://github.com/emer/gosl/v2/tree/main/examples/rand) examples for how to use in combined Go / GPU code.  In the axon example, the `slrand.Counter` is added to the `Time` context struct, and incremented after each cycle based on the number of random numbers generated for a single pass through the code, as determined by the parameter settings.  The index of each neuron being processed is used as the `key`, which is consistent in CPU and GPU versions.  Within each cycle, a *local* arg variable is incremented on each GPU processor as the computation unfolds, passed by reference after the top-level, so it updates as each RNG call is made within each pass.

Critically, these examples show that the CPU and GPU code produce identical random number sequences, which is otherwise quite difficult to achieve without this specific form of RNG.

# Implementational details

Unfortunately, vulkan `glslang` does not support 64 bit integers, even though the shader language model has somehow been updated to support them: https://github.com/KhronosGroup/glslang/issues/2965 --   https://github.com/microsoft/DirectXShaderCompiler/issues/2067.  This would also greatly speed up the impl: https://github.com/microsoft/DirectXShaderCompiler/issues/2821.

The result is that we have to use the slower version of the MulHiLo algorithm using only 32 bit uints.



