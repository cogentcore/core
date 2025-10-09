# XYZ

`xyz` is a 3D graphics framework written in Go. It is a separate standalone package that renders to an offscreen WebGPU framebuffer, which can easily be converted into a Go `image.RGBA`.  The [xyzcore](xyzcore) package provides an integration of xyz in Cogent Core, for dynamic and efficient 3D rendering within 2D GUI windows.

See [Cogent docs xyz](https://cogentcore.org/core/xyz) for full documentation. This README just has some extra detailed bits and pointers to sub-packages:

* The [physics](physics) sub-package provides a physics engine for simulating 3D virtual worlds, using xyz.

