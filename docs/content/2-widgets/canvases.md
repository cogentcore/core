# Canvases

Cogent Core provides customizable canvases that allow you to arbitrarily draw anything that you want.

You can set the function used to draw the canvas:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.FillBoxColor(mat32.V2(0, 0), mat32.V2FromPoint(pc.Image.Rect.Max), colors.Scheme.Primary.Base)
})
```
