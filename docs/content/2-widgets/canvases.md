# Canvases

Cogent Core provides customizable canvases that allow you to arbitrarily draw anything that you want.

You can set the function used to draw the canvas:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.FillBoxColor(mat32.Vec2{}, pc.Size(), colors.Scheme.Primary.Base)
})
```
