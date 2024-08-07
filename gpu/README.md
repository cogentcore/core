# GPU for Graphics and Compute

The `gpu` package manages all the details of [WebGPU](https://www.w3.org/TR/webgpu/) to provide a higher-level interface where you can specify the data variables and values, shader pipelines, and other parameters that tell the GPU what to do, without having to worry about all the lower-level implementational details.  It maps directly onto the underlying WebGPU structure, and does not decrease performance in any way.  It supports both graphics and compute functionality.

The main gpu code is in the top-level `gpu` package, with the following sub-packages available:

* [phong](phong) is a Blinn-Phong lighting model implementation on top of `gpu`, which then serves as the basis for the higherlevel [xyz](https://github.com/cogentcore/core/tree/main/xyz) 3D scenegraph system.

* [shape](shape) generates standard 3D shapes (sphere, cylinder, box, etc), with all the normals and texture coordinates.  You can compose shape elements into more complex groups of shapes, programmatically. It separates the calculation of the number of vertex and index elements from actually setting those elements, so you can allocate everything in one pass, and then configure the shape data in a second pass, consistent with the most efficient memory model provided by gpu.  It only has a dependency on the [math32](../math32) package and could be used for anything.

* [gpudraw](gpudraw) implements GPU-accelerated texture-based versions of the Go [image/draw](https://pkg.go.dev/image/draw) api.  This is used for compositing images in the `core` GUI to construct the final rendered scene, and for drawing that scene on the actual hardware window.

* [gosl](gosl) translates Go code into GPU shader language code for running compute shaders in `gpu`, playing the role of NVIDIA's "cuda" language in other frameworks.

# Platforms

* On desktop (mac, windows, linux), [glfw](https://github.com/go-gl/glfw) is used for initializing the GPU.
* Mobile (android, ios)...
  - When developing for Android on macOS, it is critical to set `Emulated Performance` -> `Graphics` to `Software` in the `Android Virtual Device Manager (AVD)`; otherwise, the app will crash on startup. This is because macOS does not support direct access to the underlying hardware GPU in the Android Emulator. You can see more information how to do this [in the Android developer documentation](https://developer.android.com/studio/run/emulator-acceleration). Please note that this issue will not affect end-users of your app, only you while you develop it. Also, due to the typically bad performance of the emulated device GPU on macOS, it is recommended that you use a more modern emulated device than the default Pixel 3a. Finally, you should always test your app on a real mobile device if possible to see what it is actually like.

# Selecting a GPU Device

For systems with multiple GPU devices, by default the discrete device is selected, and if multiple of those are present, the one with the most RAM is used.  To see what is available and their properties, use:

```
$ go run cogentcore.org/core/gpu/cmd/webgpuinfo@latest
```

(you can `install` that tool for later use as well)

The following environment variables can be set to specifically select a particular device by name (`deviceName`): 

* `MESA_VK_DEVICE_SELECT` (standard for mesa-based drivers) or `VK_DEVICE_SELECT` -- for graphics or compute usage.
* `VK_COMPUTE_DEVICE_SELECT` -- only used for compute, if present -- will override above, so you can use different GPUs for graphics vs compute.

* `GPU` represents the hardware `Adapter` and maintains global settings, info about the hardware.

* `Device` is a *logical* device and associated `Queue` info. Each such device can function in parallel.

There are many distinct mechanisms for graphics vs. compute functionality, so we review the Graphics system first, then the Compute.

# Graphics System

* `GraphicsSystem` manages multiple `GraphicsPipeline`s and associated variables (`Var`) and `Value`s, to accomplish a complete overall rendering / computational job.  The `Vars` and `Values` are shared across all pipelines within a System, which is more efficient and usually what you want.  A given shader can simply ignore the variables it doesn't need.
    + `GraphicsPipeline` performs a specific chain of operations, using `Shader` program(s).  In the graphics context, each pipeline typically handles a different type of material or other variation in rendering (textured vs. not, transparent vs. solid, etc).
    + `Vars` has up to 4 (hard limit imposed by WebGPU) `VarGroup`s which are referenced with the `@group(n)` syntax in the WGSL shader, in addition to a special `VertexGroup` specifically for the special Vertex and Index variables.  Each `VarGroup` can have a number of `Var` variables, which occupy sequential `@binding(n)` numbers within each group.
    + `Values` within `Var` manages the specific data values for each variable.  For example, each `Texture` or vertex mesh is stored in its own separate `Value`, with its own `wgpu.Buffer` that is used to transfer data from the CPU to the GPU device.  The `SetCurrent` method selects which `Value` is currently used, for the next `BindPipeline` call that sets all the values to use for a given pipeline run.  Critically, all values must be uploaded to the GPU in advance of a given GPU pass.  For large numbers of `Uniform` and `Storage` values, a `DynamicOffset` can be set so there is just a single `Value` but the specific data used is determined by the `DynamicIndex` within the one big value buffer.
  
* `Texture` manages a WebGPU Texture and associated `TextureView`, along with an optional  `Sampler` that defines how pixels are accessed in a shader.  The Texture can manage any kind of texture object, with different Config methods for the different types.

* `Renderer` is an interface for the final render target, implemented by two types:
    + `Surface` represents the full hardware-managed `Texture`s associated with an actual on-screen Window.  
    + `RenderTexture` is an offscreen render target that renders directly to a Texture, which can then be downloaded from the GPU or used directly as an input to a shader.
    + `Render` is a helper type that is used by both of the above to manage the additional depth texture and multisampling texture target.

* Unlike most game-oriented GPU setups, `gpu` is designed to be used in an event-driven manner where render updates arise from user input or other events, instead of requiring a constant render loop taking place at all times (which can optionally be established too).  The event-driven model is vastly more energy efficient for non-game applications.

## Basic render pass

These are the basic steps for a render pass, using convenient methods on the `sy = GraphicsSystem`, which then manages the rest of the underlying steps.  `pl` here is a `GraphicsPipeline`.

```go
	rp, err := sy.BeginRenderPass()
	if err != nil { // error has already been logged, as all errors are.
		return
	}
	pl.BindPipeline(rp)
	pl.BindDrawIndexed(rp)
	rp.End() // note: could add stuff after End and before EndRenderPass
	sy.EndRenderPass(rp)
```

Note that all errors are logged in the gpu system, because in general GPU-level code should not create errors once it has been debugged.

## Var and Value data

The single most important constraint in thinking about how the GPU works, is that *all resources (data in buffers, textures) must be uploaded to the GPU at the _start_ of the render pass*.

Thus, you must configure all the vars and values prior to a render pass, and if anything changes, these need to be reconfigured.

Then, during the render pass, the `BindPipeline` calls `BindAllGroups` to select which of multiple possible `Value` instances of each `Var` is actually seen by the current GPU commands.  After the initial `BindPipeline` call, you can more selectively call `BindGroup` on an individual group to update the bindings.

Furthermore if you change the `DynamicOffset` for a variable configured with that property, you need to call BindGroup to update the offset into a larger shared value buffer, to determine which value is processed.

The `Var.Values.Current` index determines which Value is used for the BindGroup call, and `SetCurrent*` methods set this for you at various levels of the variable hierarchy.  Likewise, the `Value.DynamicIndex` determines the dynamic offset, and can be set with `SetDynamicIndex*` calls.

`Vars` variables define the `Type` and `Role` of data used in the shaders.  There are 3 major categories of Var roles:

* `Vertex` and `Index` represent mesh points etc that provide input to Vertex shader -- these are handled very differently from the others, and must be located in a `VertexSet` which has a set index of -2.  The offsets into allocated Values are updated *dynamically* for each render Draw command, so you can Bind different Vertex Values as you iterate through objects within a single render pass (again, the underlying vals must be sync'd prior).

* `PushConst` (not yet available in WebGPU) are push constants that can only be 128 bytes total that can be directly copied from CPU ram to the GPU via a command -- it is the most high-performance way to update dynamically changing content, such as view matricies or indexes into other data structures.  Must be located in `PushConstSet` set (index -1).

* `Uniform` (read-only "constants") and `Storage` (read-write) data that contain misc other data, e.g., transformation matricies.  These are the only types that can optionally use the `DynamicOffset` mechanism, which should generally be reserved for cases where there is a large and variable number of values that need to be selected among during a render pass.  The [phong](phong) system stores the object-specific "model matrix" and other object-specific data using this dynamic offset mechanism.

* `Texture` vars that provide the raw `Texture` data, the `TextureView` through which that is accessed, and a `Sampler` that parametrizes how the pixels are mapped onto coordinates in the Fragment shader.  Each texture object is managed as a distinct item in device memory.  

## Naming conventions

* `New*` returns a new object.
* `Config` operates on an existing object and settings, and does everything to get it configured for use.
* `Release` releases allocated WebGPU objects.  The usual Go simplicity of not having to worry about freeing memory does not apply to these objects.

# Compute System

See `examples/compute1` for a very simple compute shader, and [compute.go](vgpu/compute.go) for `Compute*` methods specifically useful for this case.

See [gosl] for a tool that converts Go code into WGSL shader code, so you can effectively run Go on the GPU.

Here's how it works:

* Each WebGPU `Pipeline` holds **1** compute `shader` program, whicGraphicsSystemquivalent to a `kernel` in CUDA. This is the basic unit of computation, accomplishing one parallel sweep of processing across some number of identical data structures.

* You must organize at the outset your `Vars` and `Values` in the `System` to hold the data structures your shaders operate on.  In general, you want to have a single static set of Vars that cover everything you'll need, and different shaders can operate on different subsets of these.  You want to minimize the amount of memory transfer.

* Because the `Queue.Submit` call is by far the most expensive call in WebGPU, you want to minimize those.  This means you want to combine as much of your computation into one big Command sequence, with calls to various different `Pipeline` shaders (which can all be put in one command buffer) that gets submitted *once*, rather than submitting separate commands for each shader.  Ideally this also involves combining memory transfers to / from the GPU in the same command buffer as well.

* TODO: update for webgpu sync mechanisms.  Although rarGraphicsSystemed in graphics, the most important tool for synchronizing commands _within a single command stream_ is the [vkEvent](https://registry.khronos.org/WebGPU/specs/1.3-extensions/man/html/VkEvent.html), which is described a bit in the [Khronos Blog](https://www.khronos.org/blog/understanding-WebGPU-synchronization).  Much of WebGPU discussion centers instead around `Semaphores`, but these are only used for synchronization _between different commands_ --- each of which requires a different `vkQueueSubmit` (and is therefore suboptimal).

* Thus, you should create named events in your compute `System`, and inject calls to set and wait on those events in your command stream.

# Gamma Correction (sRGB vs Linear) and Headless / Offscreen Rendering

It is hard to find this info very clearly stated:

* All internal computation in shaders is done in a *linear* color space.
* Textures are assumed to be sRGB and are automatically converted to linear on upload.
* Other colors that are passed in should be converted from sRGB to linear (the [phong](phong) shader does this for the PerVertex case).
* The `Surface` automatically converts from Linear to sRGB for actual rendering.
* A `RenderTexture` for offscreen / headless rendering *must* use `wgpu.TextureFormatRGBA8UnormSrgb` for the format, in order to get back an image that is automatically converted back to sRGB format.

# Limits

See https://web3dsurvey.com/webgpu for a browser of limits across different platforms, _for the web platform_.  Note that the native version typically will have higher limits for many things across these same platforms, but because we want to maintain full interoperability across web and native, it is the lower web limits that constrain.

* https://web3dsurvey.com/webgpu/limits/maxBindGroups only 4!
* https://web3dsurvey.com/webgpu/limits/maxBindingsPerBindGroup 640 low end: plenty of room for all your variables, you just have to put them in relatively few top-level groups.
* https://web3dsurvey.com/webgpu/limits/maxDynamicUniformBuffersPerPipelineLayout 8: should be plenty.
* https://web3dsurvey.com/webgpu/limits/maxVertexBuffers 8: can't stuff too many vars into the vertex group, but typically not a problem.

# WebGPU Links

* https://webgpu.rocks/
* https://gpuweb.github.io/gpuweb/wgsl/
* https://www.w3.org/TR/webgpu
* https://web3dsurvey.com/webgpu
* https://toji.dev/webgpu-best-practices/ -- very helpful tutorial info
* https://sotrh.github.io/learn-wgpu/beginner/tutorial5-textures/

