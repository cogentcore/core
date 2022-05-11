# vGPU: Vulkan GPU Framework for Graphics and Compute, in Go

[![GoDocs for vGPU](https://pkg.go.dev/badge/github.com/goki/vgpu.svg)](https://pkg.go.dev/github.com/goki/vgpu)

How to install: https://vulkan.lunarg.com/sdk/home -- download the Vulkan SDK installer for your platform.  Unfortunately there does not appear to be a package for homebrew.

vGPU is a Vulkan-based framework for both Graphics and Compute Engine use of GPU hardware, in the Go langauge.  It uses the basic cgo-based Go bindings to Vulkan in: https://github.com/vulkan-go/vulkan and was developed starting with the associated example code surrounding that project.  Vulkan is a relatively new, essentially universally-supported interface to GPU hardware across all types of systems from mobile phones to massive GPU-based compute hardware, and it provides high-performance "bare metal" access to the hardware, for both graphics and computational uses.

[Vulkan](https://www.vulkan.org) is very low-level and demands a higher-level framework to manage the complexity and verbosity.  While there are many helpful tutorials covering the basic API, many of the tutorials don't provide much of a pathway for how to organize everything at a higher level of abstraction.  vGPU represents one attempt that enforces some reasonable choices that enable a significantly simpler programming model, while still providing considerable flexibility and high levels of performance.  Everything is a tradeoff, and simplicity definitely was prioritized over performance in a few cases, but in practical use-cases, the performance differences should be minimal.

Most GPU coding is done for gaming, but vGPU is designed for more scientific "desktop" applications based on the [GoGi](https://github.com/goki/gi) GUI framework, which it will soon power, such as visualization of complex 3D spaces and displays (particularly neural networks), and for GPU Compute acceleration.  The design choices also reflect these priorities, as noted below.

# Basic Elements and Organization

* `GPU` represents the hardware `Device` and maintains global settings, info about the hardware.
    + `Device` is a *logical* device and associated Queue info -- each such device can function in parallel.
    + `CmdPool` manages a command pool and buffer, associated with a specific logical device, for submitting commands to the GPU.

* `System` manages multiple vulkan `Pipeline`s and associated variables, variable values, and memory, to accomplish a complete overall rendering / computational job.  The Memory with Vars and Vals are shared across all pipelines within a System.
    + `Pipeline` performs a specific chain of operations, using `Shader` program(s).  In a graphics context, each pipeline typically handles a different type of material or other variation in rendering (textured vs. not, transparent vs. solid, etc).
    + `Memory` manages the memory, organized by `Vars` variables that are referenced in the shader programs, with each Var having any number of associated values in `Vals`.  Vars are organized into `Set`s that manage their bindings distinctly, and can be updated at different time scales. It has 4 different `MemBuff` buffers for different types of memory.  It is assumed that the *sizes* of all the Vals do not change frequently, so everything is Alloc'd afresh if any size changes.  This avoids the need for complex de-fragmentation algorithms, and is maximally efficient, but is not good if sizes change (which is rare in most rendering cases).
  
* `Image` manages a vulkan Image and associated `ImageView`, including potential host staging buffer (shared as in a Val or owned).
* `Texture` extends the `Image` with a `Sampler` that defines how pixels are accessed in a shader.
* `Framebuffer` manages an `Image` along with a `RenderPass` configuration that enables rendering into an offscreen image.

* A `Surface` represents the full hardware-managed `Image` associated with an actual on-screen Window.  One can associate a System with a Surface to manage the Swapchain updating for effective double or triple buffering.

* Unlike most game-oriented GPU setups, vGPU is designed to be used in an event-driven manner where render updates arise from user input or other events, instead of having a constant render loop taking place at all times.  This is vastly more energy efficient and suits the use-cases of GoGi and efficient inter-operation with the Compute engine.

## Memory organization

`Memory` maintains a host-visible, mapped staging buffer, and a corresponding device-local memory buffer that the GPU uses to compute on (the latter of which is optional for unified memory architectures).  Each `Val` records when it is modified, and a global Sync step efficiently transfers only what has changed.  *You must allocate and sync update a unique Val for each different value you will need for the entire render pass* -- although you can dynamically select *which Val* to use for each draw command, you cannot in general update the actual data associated with these values during the course of a single rendering pass.

* `Vars` variables define the `Type` and `Role` of data used in the shaders.  There are 3 major categories of Var roles:
    + `Vertex` and `Index` represent mesh points etc that provide input to Vertex shader -- these are handled very differently from the other two, and must be located in a `VertexSet` which has a set index of -2.  These are updated *dynamically* for each render Draw command, so you can Bind different Vertex Vals as you iterate through objects within a single render pass.
    + `PushConst` are push constants that can only be 128 bytes total that can be directly copied from CPU ram to the GPU via a command -- it is the most high-performance way to update dynamically changing content, such as view matricies or indexes into other data structures.  Must be located in `PushConstSet` set (index -1).
    + `Uniform` (read-only "constants") and `Storage` (read-write) data that contain misc other data, e.g., transformation matricies.  These are also updated *dynamically* using dynamic offsets, so you can also call `BindDynVal` methods to select different such vals as you iterate through objects.  The original binding is done automatically in the Memory Config (via BindDynVarsAll) and usually does not need to be redone.
    + `Texture` vars that provide the raw `Image` data, the `ImageView` through which that is accessed, and a `Sampler` that parameterizes how the pixels are mapped onto coordinates in the Fragment shader.  Each texture object is managed as a distinct item in device memory, and they cannot be accessed through a dynamic offset.  Thus, a unique descriptor is provided for each texture Val, and your shader should describe them as an array.  All such textures must be in place at the start of the render pass, and cannot be updated on the fly during rendering!  Thus, you must dynamically bind a uniform variable or push constant to select which texture item from the array to use on a given step of rendering.
    
* `Var`s are accessed at a given `location` or `binding` number by the shader code, and these bindings can be organized into logical sets called DescriptorSets (see Graphics Rendering example below).  In [HLSL](https://www.lei.chat/posts/hlsl-for-vulkan-resources/), these are specified like this: `[[vk::binding(5, 1)]] Texture2D MyTexture2;` for descriptor 5 in set 1 (set 0 is the first set).
    + You manage these sets explicitly by calling `AddSet` or `AddVertexSet` to create new `VarSet`s, and then add `Var`s directly to each set.
    + `Vals` represent the values of Vars, with each `Val` representing a distinct value of a corresponding `Var`. The `ConfigVals` call on a given `VarSet` specifies how many Vals to create per each Var within a Set -- this is a shared property of the Set, and should be a consideration in organizing the sets.  For example, Sets that are per object should contain one Val per object, etc, while others that are per material would have one Val per material.
    + There is no support for interleaved arrays -- if you want to achieve such a thing, you need to use an array of structs, but the main use-case is for VertexInput, which actually works faster with non-interleaved simple arrays of vec4 points in most cases (e.g., for the points and their normals).
    + You can allocate multiple bindings of all the variables, with each binding being used for a different parallel threaded rendering pass.  However, there is only one shared set of Vals, so your logic will have to take that into account (e.g., allocating 2x Vals per object to allow 2 separate threads to each update and bind their own unique subset of these Vals for their own render pass).  The number of such DescriptorSets is configured in `Memory.Vars.NDescs` (defaults to 1). Note that these are orthogonal to the number of `VarSet`s -- the terminology is confusing.  Various methods take a `descIdx` to determine which such descriptor set to use -- typically in a threaded swapchain logic you would use the acquired frame index as the descIdx to determine what binding to use.

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

It is common practice to use different DescriptorSets for each level in the swapchain, for maintaining high FPS rates by rendering the next frame while the current one is still cooking along itself -- this is the `NDescs` parameter mentioned above.

Because everything is all packed into big buffers organized by different broad categories, in `Memory`, we *exclusively* use the Dynamic mode for Uniform and Storage binding, where the offset into the buffer is specified at the time of the binding call, not in advance in the descriptor set itself.  This has only very minor performance implications and makes everything much more flexible and simple: just bind whatever variables you want and that data will be used.

The examples and provided `vPhong` package retain the Y-is-up coordinate system from OpenGL, which is more "natural" for the physical world, where the Y axis is the height dimension, and up is up, after all.  Some of the defaults reflect this choice, but it is easy to use the native Vulkan Y-is-down coordinate system too.

## Combining many pipeline renders per RenderPass

The various introductory tutorials all seem to focus on just a single simple render pass with one draw operation, but any realistic scene needs different settings for each object!  As noted above, this requires dynamic binding, which is good for Uniforms and Vertex data, but you might not appreciate that this also requires that you pre-allocate and sync up to device memory all the Vals that you will need for the entire render pass -- the dynamic binding only selects different offsets into memory buffers, but the actual contents of those buffers should not change during a single render pass (otherwise things will get very slow and lots of bad sync steps might be required, etc).  The Memory system makes it easy to allocate, update, and dynamically bind these vals.

Here's some info on the logical issues:

* [Stack Overflow](https://stackoverflow.com/questions/54103399/how-to-repeatedly-update-a-uniform-data-for-number-of-objects-inside-a-single-vu) discussion of the issues.

* [NVIDIA github](https://github.com/nvpro-samples/gl_vk_threaded_cadscene/blob/master/doc/vulkan_uniforms.md) has explicit code and benchmarks of different strategies.

This [blog](http://kylehalladay.com/blog/tutorial/vulkan/2018/01/28/Textue-Arrays-Vulkan.html) has a particularly clear discussion of the need for Texture arrays for managing textures within a render pass.  This is automatically how Texture vars are managed .

# GPU Accelerated Compute Engine

# Mac Platform

To have the mac use the `libMoltenVK.dylib` installed by `brew install molten-vk`, you need to change the LDFLAGS here:

`github.com/go-vulkan/vulkan/vulkan_darwin.go`

```
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -Wl,-rpath,/opt/homebrew/lib -F/Library/Frameworks -framework Cocoa -framework IOKit -framework IOSurface -framework QuartzCore -framework Metal -lMoltenVK -lc++
```

However it does not find the `libvulkan` which is not included in molten-vk.

