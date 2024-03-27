# Canvases

Cogent Core provides customizable canvases that allow you to arbitrarily draw anything that you want.

You can set the function used to draw the canvas:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.FillBoxColor(mat32.Vec2{}, pc.Size(), colors.Scheme.Primary.Base)
})
```

You can draw lines:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.MoveTo(0, 0)
    pc.LineTo(pc.Size().X, pc.Size().Y)
    pc.StrokeStyle.Color = colors.C(colors.Scheme.Error.Base)
    pc.Stroke()
})
```

You can draw circles:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.DrawCircle(pc.Size().X/2, pc.Size().Y/2, pc.Size().X/2)
    pc.FillStyle.Color = colors.C(colors.Scheme.Success.Base)
    pc.Fill()
})
```
