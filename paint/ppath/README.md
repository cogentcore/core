# paint/ppath

This is adapted from https://github.com/tdewolff/canvas, Copyright (c) 2015 Taco de Wolff, under an MIT License.

The canvas Path type provides significant powerful functionality, and we are grateful for being able to appropriate that into our framework. Because the rest of our code is based on `float32` instead of `float64`, including the xyz 3D framework and our extensive `math32` library, and because only `float32` is GPU compatible (and we are planning a WebGPU Renderer), we converted the canvas code to use `float32` and our `math32` library.

In addition, we simplified the `Path` type to just be a `[]float32` directly, which allows many of the methods to not have a pointer receiver if they don't modify the Path, making that distinction clearer.

We also organized the code into this sub-package so it is clearer what aspects are specifically about the Path vs. other aspects of the overall canvas system.

Because of the extensive tests, we can be reasonably assured that the conversion has not introduced any bugs, and we will monitor canvas for upstream changes.

# Basic usage

In the big picture, a `Path` defines a shape (outline), and depending on the additional styling parameters, this can end up being filled and / or just the line drawn. But the Path itself is only concerned with the line trajectory as a mathematical entity.

The `Path` has methods for each basic command: `MoveTo`, `LineTo`, `QuadTo`, `CubeTo`, `ArcTo`, and `Close`. These are the basic primitives from which everything else is constructed.

`shapes.go` defines helper functions that return a `Path` for various familiar geometric shapes.

Once a Path has been defined, the actual rendering process involves optionally filling closed paths and then drawing the line using a specific stroke width, which is where the `Stroke` method comes into play (which handles the intersection logic, via the `Settle` method), also adding the additional `Join` and `Cap` elements that are typically used for rendering, returning a new path. The canvas code is in `renderers/rasterizer` showing the full process, using the `golang.org/x/image/vector` rasterizer.

The rasterizer is what actually turns the path lines into discrete pixels in an image. It uses the Flatten* methods to turn curves into discrete straight line segments, so everything is just a simple set of lines in the end, which can actually be drawn. The rasterizer also does antialiasing so the results look smooth.

# Import logic

`path` is a foundational package that should not import any other packages such as paint, styles, etc. These other higher-level packages import path.

