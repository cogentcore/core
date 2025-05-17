# composer

The Composer manages the final rendering to the platform-specific window that you actually see as a user. It maintains a list of `Source` elements that provide platform-specific rendering logic, with one such source for each active Scene in a GUI (e.g., a dialog Scene could be on top of a main window Scene).

See the [render docs](https://cogentcore.org/core/render) for more info.

## Sources:
* `Painter`: Paths, Images, Text (`render.Item`s) rendered onto target "canvas" (web canvas or image.RGBA)
* `xyzRender`: GPU texture
* `video`: image.RGBA source but critically needs fast GPU `Transform` call to blit into target
* `sprites`: image.RGBA onto special top-level compositor
* `scrim`: uniform fast draw (supported by GPUDraw)

## Desktop Destinations:
* `GPUDrawer`: for final rendering
* `image.RGBA`: for intermediate icon, etc, and testing.
* `SVG`, `PDF` etc.

## JS Web Destinations:
* stack of canvases
* `image.RGBA` for icon, etc -- uses _Go_ renderx
* `SVG`, `PDF` etc.

## Renderer

* Maps Painter -> image (rasterx) or Painter -> canvas, or SVG, PDF
* htmlcanvas must get its canvas from a system-level "Drawer" system.

