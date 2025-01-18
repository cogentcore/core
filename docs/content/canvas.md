+++
Categories = ["Widgets"]
+++

A **canvas** is a [[widget]] that allows you to manually render anything. All canvas coordinates are on a normalized 0-1 scale.

If you want to render SVG files, use an [[SVG]] widget instead. For images, use an [[image]] widget. For videos, use a [[video]] widget. For HTML, see [[HTML]].

## Draw

You can set the function used to draw a canvas:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.FillBox(math32.Vector2{}, math32.Vec2(1, 1), colors.Scheme.Primary.Base)
})
```

You can draw lines:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.MoveTo(0, 0)
    pc.LineTo(1, 1)
    pc.StrokeStyle.Color = colors.Scheme.Error.Base
    pc.Stroke()
})
```

You can change the width of lines:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.MoveTo(0, 0)
    pc.LineTo(1, 1)
    pc.StrokeStyle.Color = colors.Scheme.Error.Base
    pc.StrokeStyle.Width.Dp(8)
    pc.ToDots()
    pc.Stroke()
})
```

You can draw circles:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.DrawCircle(0.5, 0.5, 0.5)
    pc.FillStyle.Color = colors.Scheme.Success.Base
    pc.Fill()
})
```

You can combine any number of canvas rendering operations:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.DrawCircle(0.6, 0.6, 0.15)
    pc.FillStyle.Color = colors.Scheme.Warn.Base
    pc.Fill()

    pc.MoveTo(0.7, 0.2)
    pc.LineTo(0.2, 0.7)
    pc.StrokeStyle.Color = colors.Scheme.Primary.Base
    pc.StrokeStyle.Width.Dp(16)
    pc.ToDots()
    pc.Stroke()
})
```

You can animate a canvas:

```Go
t := 0
c := core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.DrawCircle(0.5, 0.5, float32(t%60)/120)
    pc.FillStyle.Color = colors.Scheme.Success.Base
    pc.Fill()
})
go func() {
    for range time.Tick(time.Second/60) {
        t++
        c.NeedsRender()
    }
}()
```

You can draw ellipses:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.DrawEllipse(0.5, 0.5, 0.5, 0.25)
    pc.FillStyle.Color = colors.Scheme.Success.Base
    pc.Fill()
})
```

You can draw elliptical arcs:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.DrawEllipticalArc(0.5, 0.5, 0.5, 0.25, math32.Pi, 2*math32.Pi)
    pc.FillStyle.Color = colors.Scheme.Success.Base
    pc.Fill()
})
```

You can draw regular polygons:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.DrawRegularPolygon(6, 0.5, 0.5, 0.5, math32.Pi)
    pc.FillStyle.Color = colors.Scheme.Success.Base
    pc.Fill()
})
```

You can draw quadratic arcs:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.MoveTo(0, 0)
    pc.QuadraticTo(0.5, 0.25, 1, 1)
    pc.StrokeStyle.Color = colors.Scheme.Error.Base
    pc.Stroke()
})
```

You can draw cubic arcs:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.MoveTo(0, 0)
    pc.CubicTo(0.5, 0.25, 0.25, 0.5, 1, 1)
    pc.StrokeStyle.Color = colors.Scheme.Error.Base
    pc.Stroke()
})
```

## Styles

You can change the size of a canvas:

```Go
c := core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.FillBox(math32.Vector2{}, math32.Vec2(1, 1), colors.Scheme.Warn.Base)
})
c.Styler(func(s *styles.Style) {
    s.Min.Set(units.Dp(128))
})
```

You can make a canvas [[styles#grow]] to fill the available space:

```Go
c := core.NewCanvas(b).SetDraw(func(pc *paint.Context) {
    pc.FillBox(math32.Vector2{}, math32.Vec2(1, 1), colors.Scheme.Primary.Base)
})
c.Styler(func(s *styles.Style) {
    s.Grow.Set(1, 1)
})
```
