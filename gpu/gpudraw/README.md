# GPUDraw: GPU version of Go image/draw compositing functionality

This package uses [Alpha Compositing](https://en.wikipedia.org/wiki/Alpha_compositing) to render rectangular regions onto a render target, using the [gpu](../gpu) package, consistent with the [image/draw](https://pkg.go.dev/image/draw) and experimental [golang.org/x/image](https://pkg.go.dev/golang.org/x/image@v0.18.0/draw) package in Go.

The Cogent Core GUI, along with other 2D-based GUI frameworks, uses a strategy of rendering to various rectangular sub-regions (in Cogent Core these are `core.Scene` objects) that are updated separately, and then the final result can be composited together into a single overall image that can be pasted onto the final window surface that the user sees.

This package supports these rectangular image region composition operations, via a simple render pipeline that just renders a rectangular shape with a texture.  There is also a simple Fill pipeline that renders a single color into a rectangle.

The main API takes image.Image inputs directly, and follows the `image/draw` API as closely as possible (see [draw.go](draw.go)):

```Go
Copy(dp image.Point, src image.Image, sr image.Rectangle, op draw.Op)

Scale(dr image.Rectangle, src image.Image, sr image.Rectangle, op draw.Op)

Transform(xform math32.Matrix3, src image.Image, sr image.Rectangle, op draw.Op)

// Copy also alls this with image.Uniform image input
Fill(clr color.Color, dr image.Rectangle, op draw.Op)
```

Calls to these functions must be surrounded by `Start` and `End` calls: internally it accumulates everything after the Start call, and actually does the render at the End.

# Use / Used interface

There are also `*Used` versions of the above methods that operate on the current image that has been setup using the `Use*` methods.  These allow the system to use GPU-internal texture sources for greater efficiency where relevant.  These methods also support flipping the Y axis and other more advanced features (e.g., rotation in the `ScaleUsed` case).

# Implementation: Var map

```
Group: -2 Vertex
    Role: Vertex
        Var: 0:	Pos	Float32Vector2	(size: 8)	Values: 1
    Role: Index
        Var: 0:	Index	Uint16	(size: 2)	Values: 1
Group: 0 Matrix
    Role: Uniform
        Var: 0:	Matrix	Struct	(size: 128)	Values: 1
Group: 1 Texture
    Role: SampledTexture
        Var: 0:	TexSampler	TextureRGBA32	(size: 4)	Values: 16
```

