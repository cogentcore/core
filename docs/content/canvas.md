+++
Categories = ["Widgets"]
+++

A **canvas** is a [[widget]] that allows you to manually render anything. All canvas coordinates are on a normalized 0-1 scale.

If you want to render SVG files, use an [[SVG]] widget instead. For images, use an [[image]] widget. For videos, use a [[video]] widget. For HTML, see [[HTML]].

## Draw

You can set the function used to draw a canvas:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.FillBox(math32.Vector2{}, math32.Vec2(1, 1), colors.Scheme.Primary.Base)
})
```

You can draw lines:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.MoveTo(0, 0)
    pc.LineTo(1, 1)
    pc.Stroke.Color = colors.Scheme.Error.Base
    pc.Draw()
})
```

You can change the width of lines:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.MoveTo(0, 0)
    pc.LineTo(1, 1)
    pc.Stroke.Color = colors.Scheme.Error.Base
    pc.Stroke.Width.Dp(8)
    pc.ToDots()
    pc.Draw()
})
```

You can draw circles:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.Circle(0.5, 0.5, 0.5)
    pc.Fill.Color = colors.Scheme.Success.Base
    pc.Draw()
})
```

You can combine any number of canvas rendering operations:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.Circle(0.6, 0.6, 0.15)
    pc.Fill.Color = colors.Scheme.Warn.Base
    pc.Draw()

    pc.MoveTo(0.7, 0.2)
    pc.LineTo(0.2, 0.7)
    pc.Stroke.Color = colors.Scheme.Primary.Base
    pc.Stroke.Width.Dp(16)
    pc.ToDots()
    pc.Draw()
})
```

You can [[animate]] a canvas (see that page for more information):

```Go
t := float32(0)
c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.Circle(0.5, 0.5, 0.5*math32.Sin(t/500))
    pc.Fill.Color = colors.Scheme.Success.Base
    pc.Draw()
})
c.Animate(func(a *core.Animation) {
    t += a.Dt
    c.NeedsRender()
})
```

You can draw ellipses:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.Ellipse(0.5, 0.5, 0.5, 0.25)
    pc.Fill.Color = colors.Scheme.Success.Base
    pc.Draw()
})
```

You can draw elliptical arcs:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.EllipticalArc(0.5, 0.5, 0.5, 0.25, math32.Pi, 2*math32.Pi)
    pc.Fill.Color = colors.Scheme.Success.Base
    pc.Draw()
})
```

You can draw regular polygons:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.RegularPolygon(6, 0.5, 0.5, 0.5, math32.Pi)
    pc.Fill.Color = colors.Scheme.Success.Base
    pc.Draw()
})
```

You can draw quadratic arcs:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.MoveTo(0, 0)
    pc.QuadTo(0.5, 0.25, 1, 1)
    pc.Stroke.Color = colors.Scheme.Error.Base
    pc.Draw()
})
```

You can draw cubic arcs:

```Go
core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.MoveTo(0, 0)
    pc.CubeTo(0.5, 0.25, 0.25, 0.5, 1, 1)
    pc.Stroke.Color = colors.Scheme.Error.Base
    pc.Draw()
})
```

## Styles

You can change the size of a canvas:

```Go
c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.FillBox(math32.Vector2{}, math32.Vec2(1, 1), colors.Scheme.Warn.Base)
})
c.Styler(func(s *styles.Style) {
    s.Min.Set(units.Dp(128))
})
```

You can make a canvas [[styles#grow]] to fill the available space:

```Go
c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.FillBox(math32.Vector2{}, math32.Vec2(1, 1), colors.Scheme.Primary.Base)
})
c.Styler(func(s *styles.Style) {
    s.Grow.Set(1, 1)
})
```
