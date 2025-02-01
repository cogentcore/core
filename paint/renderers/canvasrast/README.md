# Canvas rasterizer

This is the rasterizer from https://github.com/tdewolff/canvas, Copyright (c) 2015 Taco de Wolff, under an MIT License.

First, the original canvas impl used https://pkg.go.dev/golang.org/x/image/vector for rasterizing, which it turns out is _extremely_ slow relative to rasterx/scanx (like 500 - 1000x slower!).

Second, even when using the scanx rasterizer, it is slower than rasterx because the stroking component is much faster on rasterx. Canvas does the stroking via path-based operations, whereas rasterx does it in some more direct way that ends up being faster (no idea what that way is!)

See: https://github.com/cogentcore/core/discussions/1453

At this point, this package will be marked as unused.

# TODO

* arcs-clip join is not working like it does on srwiley: TestShapes4 for example. not about the enum.
* gradients not working, but are in srwiley: probably need the bbox updates

