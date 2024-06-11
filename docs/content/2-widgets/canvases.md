# Canvases

Cogent Core provides customizable canvases that allow you to arbitrarily draw anything that you want. All canvas coordinates are on a normalized 0-1 scale.

You can set the function used to draw the canvas:

```Go
core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.FillBox(math32.Vector2{}, math32.Vec2(1, 1), colors.C(colors.Scheme.Primary.Base))
})
```

You can draw lines:

```Go
core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.MoveTo(0, 0)
    pc.LineTo(1, 1)
    pc.StrokeStyle.Color = colors.C(colors.Scheme.Error.Base)
    pc.Stroke()
})
```

You can change the width of lines:

```Go
core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.MoveTo(0, 0)
    pc.LineTo(1, 1)
    pc.StrokeStyle.Color = colors.C(colors.Scheme.Error.Base)
    pc.StrokeStyle.Width.Dp(8)
    pc.ToDots()
    pc.Stroke()
})
```

You can draw circles:

```Go
core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.DrawCircle(0.5, 0.5, 0.5)
    pc.FillStyle.Color = colors.C(colors.Scheme.Success.Base)
    pc.Fill()
})
```

You can draw ellipses:

```Go
core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.DrawEllipse(0.5, 0.5, 0.5, 0.25)
    pc.FillStyle.Color = colors.C(colors.Scheme.Success.Base)
    pc.Fill()
})
```

You can draw elliptical arcs:

```Go
core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.DrawEllipticalArc(0.5, 0.5, 0.5, 0.25, math32.Pi, 2*math32.Pi)
    pc.FillStyle.Color = colors.C(colors.Scheme.Success.Base)
    pc.Fill()
})
```

You can draw regular polygons:

```Go
core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.DrawRegularPolygon(6, 0.5, 0.5, 0.5, math32.Pi)
    pc.FillStyle.Color = colors.C(colors.Scheme.Success.Base)
    pc.Fill()
})
```

You can draw quadratic arcs:

```Go
core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.MoveTo(0, 0)
    pc.QuadraticTo(0.5, 0.25, 1, 1)
    pc.StrokeStyle.Color = colors.C(colors.Scheme.Error.Base)
    pc.Stroke()
})
```

You can draw cubic arcs:

```Go
core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.MoveTo(0, 0)
    pc.CubicTo(0.5, 0.25, 0.25, 0.5, 1, 1)
    pc.StrokeStyle.Color = colors.C(colors.Scheme.Error.Base)
    pc.Stroke()
})
```

You can change the size of the canvas:

```Go
c := core.NewCanvas(parent).SetDraw(func(pc *paint.Context) {
    pc.FillBox(math32.Vector2{}, math32.Vec2(1, 1), colors.C(colors.Scheme.Warn.Base))
})
c.Styler(func(s *styles.Style) {
    s.Min.Set(units.Dp(128))
})
```
