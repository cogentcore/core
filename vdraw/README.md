# vDraw: vGPU version of Go image/draw compositing functionality

[![GoDocs for ReComp](https://pkg.go.dev/badge/goki.dev/vgpu.svg)](https://pkg.go.dev/goki.dev/vgpu/vdraw)

This package uses [Alpha Compositing](https://en.wikipedia.org/wiki/Alpha_compositing) to render rectangular regions onto a render target, using vGPU, consistent with the [image/draw](https://pkg.go.dev/image/draw) package in Go.  Although "draw" is not a great name for this functionality, it is hard to come up with a better one that isn't crazy long, so we're adopting it -- at least it is familiar.

The GoGi GUI, and probably other 2D-based GUI frameworks, uses a strategy of rendering to various rectangular sub-regions (in GoGi these are `gi.Viewport` objects) that are updated separately, and then the final result can be composited together into a single overall image that can be pasted onto the final window surface that the user sees.  Furthermore, in Gi3D, the 3D Scene is rendered to a framebuffer, which is likewise composited into the final surface window.

This package supports these rectangular image region composition operations, via a simple render pipeline that just renders a rectangular shape with a texture.  There is also a simple fill pipeline that renders a single color into a rectangle.

The max number of images that can be pre-loaded and used per set, per render pass is only 16 -- see MaxImages.  Therefore, we use 4 sets of 16.  The last set is configured for 3D images with 128 layers to be used as sprites.

The fill color is uploaded as a push constant and thus is not subject to any limitations.

# Implementation: Var map

```
Set: -2
    Role: Vertex
        Var: 0:	Pos	Float32Vec2[4]	(size: 8)	Vals: 1
    Role: Index
        Var: 1:	Index	Uint16[6]	(size: 2)	Vals: 1
Set: -1
    Role: Push
        Var: 0:	Mtxs	Struct	(size: 128)	Vals: 0
Set: 0
    Role: TextureRole
        Var: 0:	Tex	ImageRGBA32	(size: 4)	Vals: 1
```

