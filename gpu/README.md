# GPU for Graphics and Compute

The `gpu` package manages all the details of [WebGPU](https://www.w3.org/TR/webgpu/) to provide a higher-level interface where you can specify the data variables and values, shader pipelines, and other parameters that tell the GPU what to do, without having to worry about all the lower-level implementational details.  It maps directly onto the underlying WebGPU structure, and does not decrease performance in any way. It supports both graphics and compute functionality.

See [Cogent docs gpu](https://cogentcore.org/core/gpu) for full documentation. This README just has some extra detailed bits and pointers to sub-packages:

* [phong](phong) is a Blinn-Phong lighting model implementation on top of `gpu`, which then serves as the basis for the higherlevel [xyz](https://github.com/cogentcore/core/tree/main/xyz) 3D scenegraph system.

* [shape](shape) generates standard 3D shapes (sphere, cylinder, box, etc), with all the normals and texture coordinates.  You can compose shape elements into more complex groups of shapes, programmatically. It separates the calculation of the number of vertex and index elements from actually setting those elements, so you can allocate everything in one pass, and then configure the shape data in a second pass, consistent with the most efficient memory model provided by gpu.  It only has a dependency on the [math32](../math32) package and could be used for anything.

* [gpudraw](gpudraw) implements GPU-accelerated texture-based versions of the Go [image/draw](https://pkg.go.dev/image/draw) api.  This is used for compositing images in the `core` GUI to construct the final rendered scene, and for drawing that scene on the actual hardware window.

## Platforms

* On desktop (mac, windows, linux), [glfw](https://github.com/go-gl/glfw) is used for initializing the GPU.
* Mobile (android, ios)...
  - When developing for Android on macOS, it is critical to set `Emulated Performance` -> `Graphics` to `Software` in the `Android Virtual Device Manager (AVD)`; otherwise, the app will crash on startup. This is because macOS does not support direct access to the underlying hardware GPU in the Android Emulator. You can see more information how to do this [in the Android developer documentation](https://developer.android.com/studio/run/emulator-acceleration). Please note that this issue will not affect end-users of your app, only you while you develop it. Also, due to the typically bad performance of the emulated device GPU on macOS, it is recommended that you use a more modern emulated device than the default Pixel 3a. Finally, you should always test your app on a real mobile device if possible to see what it is actually like.

## Coordinate System

See https://github.com/cogentcore/core/issues/1121 for detailed discussion of coordinate systems.

## Gamma Correction (sRGB vs Linear) and Headless / Offscreen Rendering

It is hard to find this info very clearly stated:

* All internal computation in shaders is done in a *linear* color space.
* Textures are assumed to be sRGB and are automatically converted to linear on upload.
* Other colors that are passed in should be converted from sRGB to linear (the [phong](phong) shader does this for the PerVertex case).
* The `Surface` automatically converts from Linear to sRGB for actual rendering.
* A `RenderTexture` for offscreen / headless rendering *must* use `wgpu.TextureFormatRGBA8UnormSrgb` for the format, in order to get back an image that is automatically converted back to sRGB format.

## WebGPU Links

* https://google.github.io/tour-of-wgsl/ -- much more concise and clear vs. reading the spec!
* https://webgpu.rocks/
* https://gpuweb.github.io/gpuweb/wgsl/
* https://www.w3.org/TR/webgpu
* https://web3dsurvey.com/webgpu
* https://toji.dev/webgpu-best-practices/ -- very helpful tutorial info
* https://sotrh.github.io/learn-wgpu/beginner/tutorial5-textures/
* https://webgpu.github.io/webgpu-samples/

