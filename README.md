# vGPU: Vulkan GPU Framework for Graphics and Compute, in Go

[![GoDocs for vGPU](https://pkg.go.dev/badge/github.com/goki/vgpu.svg)](https://pkg.go.dev/github.com/goki/vgpu)

**Mac Installation prerequisite:** https://vulkan.lunarg.com/sdk/home -- download the Vulkan SDK installer for the mac.  Unfortunately there does not appear to be a full version of this on homebrew -- the `molten-vk` package is not enough by itself.

vGPU is a Vulkan-based framework for both Graphics and Compute Engine use of GPU hardware, in the Go langauge.  It uses the basic cgo-based Go bindings to Vulkan in: https://github.com/vulkan-go/vulkan and was developed starting with the associated example code surrounding that project.  Vulkan is a relatively new, essentially universally-supported interface to GPU hardware across all types of systems from mobile phones to massive GPU-based compute hardware, and it provides high-performance "bare metal" access to the hardware, for both graphics and computational uses.

[Vulkan](https://www.vulkan.org) is very low-level and demands a higher-level framework to manage the complexity and verbosity.  While there are many helpful tutorials covering the basic API, many of the tutorials don't provide much of a pathway for how to organize everything at a higher level of abstraction.  vGPU represents one attempt that enforces some reasonable choices that enable a significantly simpler programming model, while still providing considerable flexibility and high levels of performance.  Everything is a tradeoff, and simplicity definitely was prioritized over performance in a few cases, but in practical use-cases, the performance differences should be minimal.

# Selecting a GPU Device

For systems with multiple GPU devices, by default the discrete device is selected, and if multiple of those are present, the one with the most RAM is used.  To see what is available and their properties, use:

```
$ vulkaninfo --summary
```

The following environment variables can be set to specifically select a particular device by name (`deviceName`): 

* `MESA_VK_DEVICE_SELECT` (standard for mesa-based drivers) or `VK_DEVICE_SELECT` -- for graphics or compute usage.
* `VK_COMPUTE_DEVICE_SELECT` -- only used for compute, if present -- will override above, so you can use different GPUs for graphics vs compute.

# vPhong and vShape

The [vPhong](https://github.com/goki/vgpu/tree/main/vphong) package provides a complete rendering implementation with different pipelines for different materials, and support for 4 different types of light sources based on the classic Blinn-Phong lighting model.  See the `examples/phong` example for how to use it.  It does not assume any kind of organization of the rendering elements, and just provides name and index-based access to all the resources needed to render a scene.

[vShape](https://github.com/goki/vgpu/tree/main/vshape) generates standard 3D shapes (sphere, cylinder, box, etc), with all the normals and texture coordinates.  You can compose shape elements into more complex groups of shapes, programmatically. It separates the calculation of the number of vertex and index elements from actually setting those elements, so you can allocate everything in one pass, and then configure the shape data in a second pass, consistent with the most efficient memory model provided by vgpu.  It only has a dependency on the [mat32](https://github.com/goki/mat32) package and could be used for anything.

# Basic Elements and Organization

* `GPU` represents the hardware `Device` and maintains global settings, info about the hardware.
    + `Device` is a *logical* device and associated Queue info -- each such device can function in parallel.
    + `CmdPool` manages a command pool and buffer, associated with a specific logical device, for submitting commands to the GPU.

* `System` manages multiple vulkan `Pipeline`s and associated variables, variable values, and memory, to accomplish a complete overall rendering / computational job.  The Memory with Vars and Vals are shared across all pipelines within a System.
    + `Pipeline` performs a specific chain of operations, using `Shader` program(s).  In a graphics context, each pipeline typically handles a different type of material or other variation in rendering (textured vs. not, transparent vs. solid, etc).
    + `Memory` manages the memory, organized by `Vars` variables that are referenced in the shader programs, with each Var having any number of associated values in `Vals`.  Vars are organized into `Set`s that manage their bindings distinctly, and can be updated at different time scales. It has 4 different `MemBuff` buffers for different types of memory.  It is assumed that the *sizes* of all the Vals do not change frequently, so everything is Alloc'd afresh if any size changes.  This avoids the need for complex de-fragmentation algorithms, and is maximally efficient, but is not good if sizes change (which is rare in most rendering cases).
  
* `Image` manages a vulkan Image and associated `ImageView`, including potential host staging buffer (shared as in a Val or owned separately).
* `Texture` extends the `Image` with a `Sampler` that defines how pixels are accessed in a shader.
* `Framebuffer` manages an `Image` along with a `RenderPass` configuration for managing a `Render` target (shared for rendering onto a window `Surface` or an offscreen `RenderFrame`)

* `Surface` represents the full hardware-managed `Image`s associated with an actual on-screen Window.  One can associate a System with a Surface to manage the Swapchain updating for effective double or triple buffering.
* `RenderFrame` is an offscreen render target with Framebuffers and a logical device if being used without any Surface -- otherwise it should use the Surface device so images can be copied across them.

* Unlike most game-oriented GPU setups, vGPU is designed to be used in an event-driven manner where render updates arise from user input or other events, instead of requiring a constant render loop taking place at all times (which can optionally be established too).  The event-driven model is vastly more energy efficient for non-game applications.

## Memory organization

`Memory` maintains a host-visible, mapped staging buffer, and a corresponding device-local memory buffer that the GPU uses to compute on (the latter of which is optional for unified memory architectures).  Each `Val` records when it is modified, and a global Sync step efficiently transfers only what has changed.  *You must allocate and sync update a unique Val for each different value you will need for the entire render pass* -- although you can dynamically select *which Val* to use for each draw command, you cannot in general update the actual data associated with these values during the course of a single rendering pass.

* `Vars` variables define the `Type` and `Role` of data used in the shaders.  There are 3 major categories of Var roles:
    + `Vertex` and `Index` represent mesh points etc that provide input to Vertex shader -- these are handled very differently from the others, and must be located in a `VertexSet` which has a set index of -2.  The offsets into allocated Vals are updated *dynamically* for each render Draw command, so you can Bind different Vertex Vals as you iterate through objects within a single render pass (again, the underlying vals must be sync'd prior).
    + `PushConst` are push constants that can only be 128 bytes total that can be directly copied from CPU ram to the GPU via a command -- it is the most high-performance way to update dynamically changing content, such as view matricies or indexes into other data structures.  Must be located in `PushConstSet` set (index -1).
    + `Uniform` (read-only "constants") and `Storage` (read-write) data that contain misc other data, e.g., transformation matricies.  These are also updated *dynamically* using dynamic offsets, so you can also call `BindDynVal` methods to select different such vals as you iterate through objects.  The original binding is done automatically in the Memory Config (via BindDynVarsAll) and usually does not need to be redone.
    + `Texture` vars that provide the raw `Image` data, the `ImageView` through which that is accessed, and a `Sampler` that parameterizes how the pixels are mapped onto coordinates in the Fragment shader.  Each texture object is managed as a distinct item in device memory, and they cannot be accessed through a dynamic offset.  Thus, a unique descriptor is provided for each texture Val, and your shader should describe them as an array.  All such textures must be in place at the start of the render pass, and cannot be updated on the fly during rendering!  Thus, you must dynamically bind a uniform variable or push constant to select which texture item from the array to use on a given step of rendering.
        + There is a low maximum number of Texture descriptors (vals) available within one descriptor set on many platforms, including the Mac, only 16, which is enforced via the `MaxTexturesPerSet` const.  There are two (non mutually-exclusive) strategies for increasing the number of available textures:
        + Each individual Texture can have up to 128 (again a low limit present on the Mac) layers in a 2d Array of images, in addition to all the separate texture vals being in an Array -- arrays of arrays.  Each of the array layers must be the same size -- they are allocated and managed as a unit.  The `szalloc` package provides manager for efficiently allocating images of various sizes to these 16 x 128 (or any N's) groups of layer arrays.  This is integrated into the `Vals` value manager and can be engaged by calling `AllocTexBySize` there.  The texture UV coordinates need to be processed by the actual pct size of a given texture relative to the allocated group size -- this is all done in the `vphong` package and the `texture_frag.frag` file there can be consulted for a working example.
        + If you allocate more than 16 texture Vals, then multiple entire collections of these descriptors will be allocated, as indicated by the `NTextureDescs` on `VarSet` and `Vars` (see that for more info, and the `vdraw` `Draw` method for an example).  You can use `Vars.BindAllTextureVals` to bind all texture vals (iterating over NTextureDescs), and `System.CmdBindTextureVarIdx` to automatically bind the correct set.
        +, here's some `glsl` shader code showing how to use the `sampler2DArray`, as used in the `vdraw` `draw_frag.frag` fragment shader code:
        
```
#version 450
#extension GL_EXT_nonuniform_qualifier : require

// must use mat4 -- mat3 alignment issues are horrible.
// each mat4 = 64 bytes, so full 128 byte total, but only using mat3.
// pack the tex index into [0][3] of mvp,
// and the fill color into [3][0-3] of uvp
layout(push_constant) uniform Mtxs {
	mat4 mvp;
	mat4 uvp;
};

layout(set = 0, binding = 0) uniform sampler2DArray Tex[]; //
layout(location = 0) in vec2 uv;
layout(location = 0) out vec4 outputColor;

void main() {
	int idx = int(mvp[3][0]);   // packing into unused part of mat4 matrix push constant
	int layer = int(mvp[3][1]);
	outputColor = texture(Tex[idx], vec3(uv,layer)); // layer selection as 3rd dim here
}
```
    
* `Var`s are accessed at a given `location` or `binding` number by the shader code, and these bindings can be organized into logical sets called DescriptorSets (see Graphics Rendering example below).  In [HLSL](https://www.lei.chat/posts/hlsl-for-vulkan-resources/), these are specified like this: `[[vk::binding(5, 1)]] Texture2D MyTexture2;` for descriptor 5 in set 1 (set 0 is the first set).
    + You manage these sets explicitly by calling `AddSet` or `AddVertexSet` / `AddPushConstSet` to create new `VarSet`s, and then add `Var`s directly to each set.
    + `Vals` represent the values of Vars, with each `Val` representing a distinct value of a corresponding `Var`. The `ConfigVals` call on a given `VarSet` specifies how many Vals to create per each Var within a Set -- this is a shared property of the Set, and should be a consideration in organizing the sets.  For example, Sets that are per object should contain one Val per object, etc, while others that are per material would have one Val per material.
    + There is no support for interleaved arrays -- if you want to achieve such a thing, you need to use an array of structs, but the main use-case is for VertexInput, which actually works faster with non-interleaved simple arrays of vec4 points in most cases (e.g., for the points and their normals).
    + You can allocate multiple bindings of all the variables, with each binding being used for a different parallel threaded rendering pass.  However, there is only one shared set of Vals, so your logic will have to take that into account (e.g., allocating 2x Vals per object to allow 2 separate threads to each update and bind their own unique subset of these Vals for their own render pass).  See discussion under `Texture` about how this is done in that case.  The number of such DescriptorSets is configured in `Memory.Vars.NDescs` (defaults to 1). Note that these are orthogonal to the number of `VarSet`s -- the terminology is confusing.  Various methods take a `descIdx` to determine which such descriptor set to use -- typically in a threaded swapchain logic you would use the acquired frame index as the descIdx to determine what binding to use.

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

See `examples/compute1` for a very simple compute shader, and [compute.go](vgpu/compute.go) for `Compute*` methods specifically useful for this case.

See the [gosl](https://github.com/goki/gosl) repository for a tool that converts Go code into HLSL shader code, so you can effectively run Go on the GPU.

Here's how it works:

* Each Vulkan `Pipeline` holds **1** compute `shader` program, which is equivalent to a `kernel` in CUDA -- this is the basic unit of computation, accomplishing one parallel sweep of processing across some number of identical data structures.

* You must organize at the outset your `Vars` and `Vals` in the `System` `Memory` to hold the data structures your shaders operate on.  In general, you want to have a single static set of Vars that cover everything you'll need, and different shaders can operate on different subsets of these.  You want to minimize the amount of memory transfer.

* Because the `vkQueueSubmit` call is by far the most expensive call in Vulkan, you want to minimize those.  This means you want to combine as much of your computation into one big Command sequence, with calls to various different `Pipeline` shaders (which can all be put in one command buffer) that gets submitted *once*, rather than submitting separate commands for each shader.  Ideally this also involves combining memory transfers to / from the GPU in the same command buffer as well.

* Although rarely used in graphics, the most important tool for synchronizing commands _within a single command stream_ is the [vkEvent](https://registry.khronos.org/vulkan/specs/1.3-extensions/man/html/VkEvent.html), which is described a bit in the [Khronos Blog](https://www.khronos.org/blog/understanding-vulkan-synchronization).  Much of vulkan discussion centers instead around `Semaphores`, but these are only used for synchronization _between different commands_ --- each of which requires a different `vkQueueSubmit` (and is therefore suboptimal).

* Thus, you should create named events in your compute `System`, and inject calls to set and wait on those events in your command stream.

# Mac Platform

To have the mac use the `libMoltenVK.dylib` installed by `brew install molten-vk`, you need to change the LDFLAGS here:

`github.com/goki/vulkan/vulkan_darwin.go`

```
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -Wl,-rpath,/opt/homebrew/lib -F/Library/Frameworks -framework Cocoa -framework IOKit -framework IOSurface -framework QuartzCore -framework Metal -lMoltenVK -lc++
```

However it does not find the `libvulkan` which is not included in molten-vk.

# Platform properties

See MACOS.md file for full report of properties on Mac.

These are useful for deciding what kinds of limits are likely to work in practice:

* 4 max bound descriptor sets: keep below this in general. https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxBoundDescriptorSets&platform=all

* maxPerStageDescriptorSamplers is only 16 on mac -- this is the relevant limit on textures!  also SampledImages is basically the same:
https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxPerStageDescriptorSamplers&platform=all
https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxPerStageDescriptorSampledImages&platform=all

This is a significant constraint!  need to work around it.

* 8 dynamic uniform descriptors (mac has 155) https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxDescriptorSetUniformBuffersDynamic&platform=all

Note that this constraint is largely irrelevant because each dynamic descriptor can have an unlimited number of offset values used for it. 

* 128 push constant bytes actually quite prevalent: https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxPushConstantsSize&platform=all

* image formats: https://vulkan.gpuinfo.org/listsurfaceformats.php

