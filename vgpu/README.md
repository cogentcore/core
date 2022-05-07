# vGPU: emergent GPU hardware support

[![GoDocs for vGPU](https://pkg.go.dev/badge/github.com/goki/vgpu.svg)](https://pkg.go.dev/github.com/goki/vgpu/vgpu)


Note: this is the sub-readme docs for the vgpu library itself, whereas overall repo project README can provide higher-level intro -- this is more for the "tech nitty-gritty".

This is a work in progress exploring using Vulkan to access the GPU.  Ultimately, it should replace the opengl implementation of the `oswin/gpu` interfaces in `GoGi`, to provide a more future-proof graphics backend.  As such, the code is being organized around that structure.

The Go vulkan bindings are from here: https://github.com/vulkan-go/vulkan, and the initial boilerplate code for various things is from https://github.com/vulkan-go/asche and https://github.com/vulkan-go/demos

Key docs for all major Vulkan types: https://gpuopen.com/learn/understanding-vulkan-objects/

For compute engine use, we are following this tutorial and associated linked ones:
https://bakedbits.dev/posts/vulkan-compute-example/

# TODO

* verify that Mem.Config works when called repeatedly.

* multisampling

* Figure out when you need to call the Update* binding vs. just the dynamic binding - in principle for pure Uniform / Storage updates, don't need the Update guy.

* Full Phong package -- shouldn't have to write that separately.

# Links

* https://developer.nvidia.com/blog/vulkan-dos-donts/ -- lots of key tips there

* https://asawicki.info/news_1740_vulkan_memory_types_on_pc_and_how_to_use_them
* https://github.com/Glavnokoman/vuh
* https://arm-software.github.io/vulkan-sdk/basic_compute.html
* https://vkguide.dev/docs/chapter-4/storage_buffers/

* https://stackoverflow.com/questions/67831583/vanilla-vulkan-compute-shader-not-writing-to-output-buffer -- how to write to output buffer in compute shader.

* https://www.reddit.com/r/vulkan/comments/rtpdvu/interleaved_vs_separate_vertex_buffers/ -- separate is actually better for most cases, and is *vastly* simpler.

* https://www.lei.chat/posts/hlsl-for-vulkan-resources/ -- key for HLSL resource bindings!
