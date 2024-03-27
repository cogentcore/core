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
    sz := pc.Size()
    pc.MoveTo(0, 0)
    pc.LineTo(sz.X, sz.Y)
    pc.StrokeStyle.Color = colors.C(colors.Scheme.Error.Base)
    pc.Stroke()
})
```

You can change the width of lines:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    sz := pc.Size()
    pc.MoveTo(0, 0)
    pc.LineTo(sz.X, sz.Y)
    pc.StrokeStyle.Width.Dp(8)
    pc.StrokeStyle.Color = colors.C(colors.Scheme.Error.Base)
    pc.Stroke()
})
```

You can draw circles:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    sz := pc.Size()
    pc.DrawCircle(sz.X/2, sz.Y/2, sz.X/2)
    pc.FillStyle.Color = colors.C(colors.Scheme.Success.Base)
    pc.Fill()
})
```

You can draw ellipses:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    sz := pc.Size()
    pc.DrawEllipse(sz.X/2, sz.Y/2, sz.X/2, sz.Y/4)
    pc.FillStyle.Color = colors.C(colors.Scheme.Success.Base)
    pc.Fill()
})
```

You can draw elliptical arcs:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    sz := pc.Size()
    pc.DrawEllipticalArc(sz.X/2, sz.Y/2, sz.X/2, sz.Y/4, mat32.Pi, 2*mat32.Pi)
    pc.FillStyle.Color = colors.C(colors.Scheme.Success.Base)
    pc.Fill()
})
```

You can draw regular polygons:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    sz := pc.Size()
    pc.DrawRegularPolygon(6, sz.X/2, sz.Y/2, sz.X/2, mat32.Pi)
    pc.FillStyle.Color = colors.C(colors.Scheme.Success.Base)
    pc.Fill()
})
```

You can draw quadratic arcs:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    sz := pc.Size()
    pc.MoveTo(0, 0)
    pc.QuadraticTo(sz.X/2, sz.Y/4, sz.X, sz.Y)
    pc.StrokeStyle.Color = colors.C(colors.Scheme.Error.Base)
    pc.Stroke()
})
```

You can draw cubic arcs:

```Go
gi.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    sz := pc.Size()
    pc.MoveTo(0, 0)
    pc.CubicTo(sz.X/2, sz.Y/4, sz.X/4, sz.Y/2, sz.X, sz.Y)
    pc.StrokeStyle.Color = colors.C(colors.Scheme.Error.Base)
    pc.Stroke()
})
```
