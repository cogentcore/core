+++
Name = "GPU"
Categories = ["Architecture"]
+++

The [[doc:gpu]] package provides a higher-level interface to [WebGPU](https://www.w3.org/TR/webgpu/), which is available on all non-web platforms and is gaining wider browser support: [WebGPU](https://caniuse.com/webgpu).

The GPU (_graphical processing unit_) is a hardware-accelerated graphics and compute processor.

The `gpu` package manages all the details of WebGPU to provide a higher-level interface where you can specify the data variables and values, shader pipelines, and other parameters that tell the GPU what to do, without having to worry about all the lower-level implementational details. It maps directly onto the underlying WebGPU structure, and does not decrease performance in any way. It supports both graphics and compute functionality.

The [[xyz]] package provides an even higher-level 3D graphics abstraction on top of `gpu`, which is what most users will typically want to use.

The main gpu code is in the top-level `gpu` package, with the following sub-packages available:

* [[doc:gpu/phong]] is a Blinn-Phong lighting model implementation on top of `gpu`, which then serves as the basis for the higher-level [[xyz]] 3D scenegraph system.

* [[doc:gpu/shape]] generates standard 3D shapes (sphere, cylinder, box, etc), with all the normals and texture coordinates. You can compose shape elements into more complex groups of shapes, programmatically. It separates the calculation of the number of vertex and index elements from actually setting those elements, so you can allocate everything in one pass, and then configure the shape data in a second pass, consistent with the most efficient memory model provided by gpu. It only has a dependency on the [math32](../math32) package and could be used for anything.

* [[doc:gpu/gpudraw]] implements GPU-accelerated texture-based versions of the Go [image/draw](https://pkg.go.dev/image/draw) api. This is used by the `Composer` framework for compositing images in the `core` GUI to construct the final rendered scene, and for drawing that scene on the actual hardware window (see [[render]] for details).

We maintain a separate [webgpu](https://github.com/cogentcore/webgpu) package that provides a Go and JS wrapper around the rust-based [wgpu](https://github.com/gfx-rs/wgpu) and [wgpu-native](https://github.com/gfx-rs/wgpu-native) packages that actually implement WebGPU itself on the desktop and mobile. This "native" version is just as performant as the much more difficult-to-use [Vulkan](https://www.vulkan.org/) framework, which we used to use.

## Selecting a GPU Device

For systems with multiple GPU devices, by default the discrete device is selected, and if multiple of those are present, the one with the most RAM is used. To see what is available and their properties, use:

```
$ go run cogentcore.org/core/gpu/cmd/webgpuinfo@main
```

(you can `install` that tool for later use as well)

There are different rules and ordering of adapters for graphics vs. compute usage.

## Graphics usage

The `GPU_DEVICE` environment variable selects a particular device by number or name (`deviceName`). The order of the devices are as presented by the WebGPU system, and shown in the `webgpuinfo` listing.

## Compute usage

For compute usage, if there are multiple discrete devices, then they are ordered from 0 to n-1 for device numbering, so that the logical process of selecting among different devices is straightforward.  The `gpu.SelectAdapter` variable can be set to directly set an adapter by logical index, or the `GPU_COMPUTE_DEVICE` environment variable.

## Types

* `GPU` represents the hardware `Adapter` and maintains global settings, info about the hardware.

* `Device` is a *logical* device and associated `Queue` info. Each such device can function in parallel.

There are many distinct mechanisms for graphics vs. compute functionality, so we review the Graphics system first, then the Compute.

## Graphics System

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

The single most important constraint in thinking about how the GPU works, is that _all resources (data in buffers, textures) must be uploaded to the GPU at the _start_ of the render pass_.

Thus, you must configure all the vars and values prior to a render pass, and if anything changes, these need to be reconfigured.

Then, during the render pass, the `BindPipeline` calls `BindAllGroups` to select which of multiple possible `Value` instances of each `Var` is actually seen by the current GPU commands.  After the initial `BindPipeline` call, you can more selectively call `BindGroup` on an individual group to update the bindings.

Furthermore if you change the `DynamicOffset` for a variable configured with that property, you need to call BindGroup to update the offset into a larger shared value buffer, to determine which value is processed.

The `Var.Values.Current` index determines which Value is used for the BindGroup call, and `SetCurrent*` methods set this for you at various levels of the variable hierarchy.  Likewise, the `Value.DynamicIndex` determines the dynamic offset, and can be set with `SetDynamicIndex*` calls.

`Vars` variables define the `Type` and `Role` of data used in the shaders.  There are 3 major categories of Var roles:

* `Vertex` and `Index` represent mesh points etc that provide input to Vertex shader -- these are handled very differently from the others, and must be located in a `VertexSet` which has a set index of -2.  The offsets into allocated Values are updated *dynamically* for each render Draw command, so you can Bind different Vertex Values as you iterate through objects within a single render pass (again, the underlying vals must be sync'd prior).

* `PushConst` (not yet available in WebGPU) are push constants that can only be 128 bytes total that can be directly copied from CPU ram to the GPU via a command -- it is the most high-performance way to update dynamically changing content, such as view matricies or indexes into other data structures.  Must be located in `PushConstSet` set (index -1).

* `Uniform` (read-only "constants") and `Storage` (read-write) data that contain misc other data, e.g., transformation matricies.  These are the only types that can optionally use the `DynamicOffset` mechanism, which should generally be reserved for cases where there is a large and variable number of values that need to be selected among during a render pass.  The [phong](phong) system stores the object-specific "model matrix" and other object-specific data using this dynamic offset mechanism.

* `Texture` vars that provide the raw `Texture` data, the `TextureView` through which that is accessed, and a `Sampler` that parametrizes how the pixels are mapped onto coordinates in the Fragment shader.  Each texture object is managed as a distinct item in device memory.  

## Coordinate System

The world and "normalized display coordinate" (NDC) system for `gpu` is the following right-handed framework:

```
    ^
 Y+ | 
    |
    +-------->
   /      X+
  / Z+
 v
```

Which is consistent with the [standard cartesian coordinate system](https://en.wikipedia.org/wiki/Cartesian_coordinate_system), where everything is rotated 90 degrees along the X axis, so that Y+ now points into the depth plane, and Z+ points upward:

```
    ^   ^
 Z+ |  / Y+
    | / 
    +-------->
   /      X+
  / Y-
 v
```

You can think of this as having vertical "stacks" of standard X-Y coordinates, stacked up along the Z axis, like a big book of graph paper. The advantage of our "Y+ up" system is that the X-Y 2D cartesian plane then maps directly onto the actual 2D screen that the user is looking at, with Z being the "extra" depth axis.  Given the primacy and universal standard way of understanding the 2D plane, this consistency seems like a nice advantage.

In this coordinate system, the standard _front face winding order_ is clockwise (CW), so the default is set to: `pl.SetFrontFace(wgpu.FrontFaceCW)` in the `GraphicsPipeline`.  

The above coordinate system is consistent with OpenGL, but other 3D rendering frameworks, including the default in WebGPU, have other systems, as documented here: https://github.com/gpuweb/gpuweb/issues/416. WebGPU is consistent with DirectX and Metal (by design), and is a _left handed_ coordinate system (using `FrontFaceCCW` by default), which conflicts with the near-universal [right-hand-rule](https://en.wikipedia.org/wiki/Right-hand_rule) used in physics and engineering.  Vulkan has its own peculiar coordinate system, with the "up" Y direction being _negative_, which turns it into a right-handed system, but one that doesn't make a lot of intuitive sense.

## Compute system

See `examples/compute1` for a very simple compute shader.

See the `gosl` system in [Cogent Lab](https://github.com/cogentcore/lab) for a tool that converts Go code into WGSL shader code, so you can effectively run Go on the GPU.

Here's how it works:

* Each WebGPU `Pipeline` holds **1** compute `shader` program, which is equivalent to a `kernel` in CUDA. This is the basic unit of computation, accomplishing one parallel sweep of processing across some number of identical data structures.

* The `Vars` and `Values` in the `System` hold all the data structures your shaders operate on, and must be configured and data uploaded before running.  In general, it is best to have a single static set of Vars that cover everything you'll need, and different shaders can operate on different subsets of these, minimizing the amount of memory transfer.

* Because the `Queue.Submit` call is by far the most expensive call in WebGPU, it should be minimized. This means combining as much of your computation into one big Command sequence, with calls to various different `Pipeline` shaders (which can all be put in one command buffer) that gets submitted _once_, rather than submitting separate commands for each shader.  Ideally this also involves combining memory transfers to / from the GPU in the same command buffer as well.

* There are no explicit sync mechanisms on the command, CPU side WebGPU (they only exist in the WGSL shaders), but it is designed so that shader compute is automatically properly synced with prior and subsequent memory transfer commands, so it automatically does the right thing for most use cases.

* Compute is particularly taxing on memory transfer in general, and overall the best strategy is to rely on the optimized `WriteBuffer` command to transfer from CPU to GPU, and then use a staging buffer to read data back from the GPU. E.g., see [this reddit post](https://www.reddit.com/r/wgpu/comments/13zqe1u/can_someone_please_explain_to_me_the_whole_buffer/).  Critically, the write commands are queued and any staging buffers are managed internally, so it shouldn't be much slower than manually doing all the staging. For reading, we have to implement everything ourselves, and here it is critical to batch the `ReadSync` calls for all relevant values, so they all happen at once. Use ad-hoc `ValueGroup`s to organize these batched read operations efficiently for the different groups of values that need to be read back in the different compute stages.

* For large numbers of items to compute, there is a strong constraint that only 65_536 (2^16) workgroups can be submitted, _per dimension_ at a time. For unstructured 1D indexing, we typically use `[64,1,1]` for the workgroup size (which must be hard-coded into the shader and coordinated with the Go side code), which gives 64 * 65_536 = 4_194_304 max items. For more than that number, more than 1 needs to be used for the second dimension. The NumWorkgroups* functions return appropriate sizes with a minimum remainder. See [examples/compute](examples/compute) for the logic needed to get the overall global index from the workgroup sizes.

## Naming conventions

* `New*` returns a new object.
* `Config` operates on an existing object and settings, and does everything to get it configured for use.
* `Release` releases allocated WebGPU objects.  The usual Go simplicity of not having to worry about freeing memory does not apply to these objects.

## Limits

See https://web3dsurvey.com/webgpu for a browser of limits across different platforms, _for the web platform_.  Note that the native version typically will have higher limits for many things across these same platforms, but because we want to maintain full interoperability across web and native, it is the lower web limits that constrain.

* https://web3dsurvey.com/webgpu/limits/maxBindGroups only 4!
* https://web3dsurvey.com/webgpu/limits/maxBindingsPerBindGroup 640 low end: plenty of room for all your variables, you just have to put them in relatively few top-level groups.
* https://web3dsurvey.com/webgpu/limits/maxDynamicUniformBuffersPerPipelineLayout 8: should be plenty.
* https://web3dsurvey.com/webgpu/limits/maxVertexBuffers 8: can't stuff too many vars into the vertex group, but typically not a problem.

