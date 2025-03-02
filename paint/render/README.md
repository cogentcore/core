# render API

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

## Drawer, render.Scene

We need a new abstraction at this level.  Drawer is not the right one.

It must be at the system level.


