# vGPU: Vulkan GPU Framework for Graphics and Compute, in Go

[![GoDocs for vGPU](https://pkg.go.dev/badge/github.com/goki/vgpu.svg)](https://pkg.go.dev/github.com/goki/vgpu)

vGPU is a Vulkan-based framework for both Graphics and Compute Engine use of GPU hardware, in the Go langauge.  It uses the basic cgo-based Go bindings to Vulkan in: https://github.com/vulkan-go/vulkan and was developed starting with the associated example code surrounding that project.  Vulkan is a relatively new, essentially universally-supported interface to GPU hardware across all types of systems from mobile phones to massive GPU-based compute hardware, and it provides high-performance "bare metal" access to the hardware, for both graphics and computational uses.

[Vulkan](https://www.vulkan.org) is very low-level and demands a higher-level framework to manage the complexity and verbosity.  While there are many helpful tutorials covering the basic API, many of the tutorials don't provide much of a pathway for how to organize everything at a higher level of abstraction.  vGPU represents one attempt that enforces some reasonable choices that enable a significantly simpler programming model, while still providing considerable flexibility and high levels of performance.  Everything is a tradeoff, and simplicity definitely was prioritized over performance in a few cases, but in practical use-cases, the performance differences should be minimal.

Most GPU coding is done for gaming, but vGPU is designed for more scientific "desktop" applications based on the [GoGi](https://github.com/goki/gi) GUI framework, which it will soon power, such as visualization of complex 3D spaces and displays (particularly neural networks), and for GPU Compute acceleration.  The design choices also reflect these priorities, as noted below.

# Basic Elements and Organization

* `GPU` represents the hardware `Device` and maintains global settings, info about the hardware.
    + `Device` is a *logical* device and associated Queue info -- each such device can function in parallel.
    + `CmdPool` manages a command pool and buffer, associated with a specific logical device, for submitting commands to the GPU.

* `System` manages multiple vulkan `Pipeline`s and associated variables, data values, and memory, to accomplish a complete overall rendering / computational job.  The Vars, Vals and Memory are shared across all pipelines within a System.
    + `Pipeline` performs a specific chain of operations, using `Shader` program(s).  In a graphics context, each pipeline typically handles a different type of material or other variation in rendering (textured vs. not, transparent vs. solid, etc).
    + `Vars` variables on the System define the `Type` and `Role` of data used in the shaders.  There are 3 major categories of Var roles:
        - `VertexInput` such as mesh points etc that provide input to Vertex shader -- these are handled very differently from the other two: 
        - `Uniform` (read-only "constants") and `Storage` (read-write) data that contain misc other data, e.g., transformation matricies.
        - `Image` data that provide textures etc.
    + `Var`s are accessed at a given `location` or `binding` number by the shader code, and these bindings can be organized into logical sets called DescriptorSets (see Graphics Rendering example below).  In [HLSL](https://www.lei.chat/posts/hlsl-for-vulkan-resources/), these are specified like this: `[[vk::binding(5, 1)]] Texture2D MyTexture2;` for descriptor 5 in set 1 (set 0 is the first set).
    + The logical configuration of these bindings is established by the Vars up-front when configuring the pipeline, and then the binding of actual values to these vars takes place during run-time.
    + `Vals` represent the values of Vars, with each `Val` representing a distinct value of a corresponding `Var`.  There is no support for interleaved arrays -- if you want to achieve such a thing, you need to use an array of structs, but the main use-case is for VertexInput, which actually works faster with non-interleaved simple arrays of vec4 points in most cases (e.g., for the points and their normals).  Vals are at the System level, for shared access from each individual Pipeline.
    + `Memory` manages the memory for all the `Vals`.  It has 4 different `MemBuff` buffers, one for Image data (textures), which have different constraints, and another for everything else.  It is assumed that the *sizes* of all the Vals do not change frequently, so everything is Alloc'd afresh if any size changes.  This avoids the need for complex de-fragmentation algorithms, and is maximally efficient, but is not good if sizes change (which is rare in most rendering cases).
        + Memory maintains a host-visible, mapped staging buffer, and a corresponding device-local memory buffer that the GPU uses to compute on (the latter of which is optional for unified memory architectures).  Each `Val` records when it is modified, and a global Sync step efficiently transfers only what has changed.

* `Image` manages a vulkan Image and associated `ImageView`, including potential host staging buffer (shared as in a Val or owned).
* `Texture` extends the `Image` with a `Sampler` that defines how pixels are accessed in a shader.
* `Framebuffer` manages an `Image` along with a `RenderPass` configuration that enables rendering into an offscreen image.

* A `Surface` represents the full hardware-managed `Image` associated with an actual on-screen Window.  One can associate a System with a Surface to manage the Swapchain updating for effective double or triple buffering.

* Unlike most game-oriented GPU setups, vGPU is designed to be used in an event-driven manner where render updates arise from user input or other events, instead of having a constant render loop taking place at all times.  This is vastly more energy efficient and suits the use-cases of GoGi and efficient inter-operation with the Compute engine.

## Naming conventions

* `New` returns a new object
* `Init` operates on existing object, doing initialization needed for subsequent setting of options
* `Config` operates on an existing object and settings, and does everything to get it configured for use.
* `Destroy` destroys allocated vulkan objects
* `Alloc` is for allocating memory (vs. making a new object)
* `Free` is for freeing memory (vs. destroying an object)

# Graphics Rendering

See https://developer.nvidia.com/vulkan-shader-resource-binding for a clear description of DescriptorSets etc.

Here's a widely-used rendering logic, supported by the GoGi Scene (and tbd std Pipeline), and how you should organize the Uniform data into different sets at each level, to optimize the binding overhead:

```
for each view {
  bind view resources [Set 0]         // camera, environment...
  for each shader type (based on material type: textured, transparent..) {
    bind shader pipeline  
    bind shader resources [Set 1]    // shader control values (maybe none)
    for each specific material {
      bind material resources  [Set 2] // material parameters and textures
      for each object {
        bind object resources  [Set 3] // object transforms
        draw object [VertexInput binding to locations]
        (only finally calls Pipeline here!)
      }
    }
  }
}
```

There is also a common practice of using different DescriptorSets for each level in the swapchain, for maintaining high FPS rates by rendering the next frame while the current one is still cooking along itself.  However, this is not the default mode supported by vGPU -- if this is desired, you can do it using different Vars for each frame.

Because everything is all packed into big buffers organized by different broad categories, in `Memory`, we *exclusively* use the Dynamic mode for Uniform and Storage binding, where the offset into the buffer is specified at the time of the binding call, not in advance in the descriptor set itself.  This has only very minor performance implications and makes everything much more flexible and simple: just bind whatever variables you want and that data will be used.

The examples and provided `vPhong` package retain the Y-is-up coordinate system from OpenGL, which is more "natural" for the physical world, where the Y axis is the height dimension, and up is up, after all.  Some of the defaults reflect this choice, but it is easy to use the native Vulkan Y-is-down coordinate system too.

# GPU Accelerated Compute Engine


