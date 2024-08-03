# GPUDraw: GPU version of Go image/draw compositing functionality

This package uses [Alpha Compositing](https://en.wikipedia.org/wiki/Alpha_compositing) to render rectangular regions onto a render target, using the [gpu](../gpu) package, consistent with the [image/draw](https://pkg.go.dev/image/draw) package in Go.

The Cogent Core GUI, along with other 2D-based GUI frameworks, uses a strategy of rendering to various rectangular sub-regions (in Cogent Core these are `core.Scene` objects) that are updated separately, and then the final result can be composited together into a single overall image that can be pasted onto the final window surface that the user sees.

This package supports these rectangular image region composition operations, via a simple render pipeline that just renders a rectangular shape with a texture.  There is also a simple fill pipeline that renders a single color into a rectangle.

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

