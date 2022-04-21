# eGPU: emergent GPU hardware support

This is a work in progress exploring using Vulkan to access the GPU.  Ultimately, it should replace the opengl implementation of the `oswin/gpu` interfaces in `GoGi`, to provide a more future-proof graphics backend.  As such, the code is being organized around that structure.

The Go vulkan bindings are from here: https://github.com/vulkan-go/vulkan, and the initial boilerplate code for various things is from https://github.com/vulkan-go/asche and https://github.com/vulkan-go/demos

Key docs for all major Vulkan types: https://gpuopen.com/learn/understanding-vulkan-objects/

For compute engine use, we are following this tutorial and associated linked ones:

https://bakedbits.dev/posts/vulkan-compute-example/

# Links

* https://asawicki.info/news_1740_vulkan_memory_types_on_pc_and_how_to_use_them
* https://github.com/Glavnokoman/vuh
