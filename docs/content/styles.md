**Styles** contains explanations of common [[style]] properties. You can also see the API documentation for an [exhaustive list](https://pkg.go.dev/cogentcore.org/core/styles#Style) of style properties. You can experiment with style properties in the [[style playground]].

## Color

Many style properties involve [[color]]s, which can be specified in several ways as documented on that linked page.

You can set the content color of [[text]] or an [[icon]]:

```Go
tx := core.NewText(b).SetText("Success")
tx.Styler(func(s *styles.Style) {
    s.Color = colors.Scheme.Success.Base
})
```

### Background

You can set the background color of a [[widget]]:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Background = gradient.NewLinear().AddStop(colors.Yellow, 0).AddStop(colors.Orange, 0.5).AddStop(colors.Red, 1)
    s.Min.Set(units.Em(5))
})
```

## Border

You can add a border to a widget:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Border.Width.Set(units.Dp(4))
    s.Border.Color.Set(colors.Scheme.Outline)
    s.Min.Set(units.Em(5))
})
```

You can make a dotted or dashed border:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Border.Style.Set(styles.BorderDotted)
    s.Border.Width.Set(units.Dp(4))
    s.Border.Color.Set(colors.Scheme.Warn.Base)
    s.Min.Set(units.Em(5))
})
```

You can specify different border properties for different sides of a widget (see the documentation for [[doc:styles.Sides.Set]]):

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Border.Width.Set(units.Dp(4))
    s.Border.Color.Set(colors.Scheme.Warn.Base, colors.Scheme.Error.Base)
    s.Min.Set(units.Em(5))
})
```

### Border radius

You can make a widget have a curved rendering boundary, whether or not it has a [[#border]]:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Border.Radius = styles.BorderRadiusLarge
    s.Background = colors.Scheme.Error.Base
    s.Min.Set(units.Em(5))
})
```

## Layout

There are many layout properties that customize the positioning and sizing of widgets. See the [[layout]] page for a low-level explanation of the layout process.

### Size

You can control the size of a widget through three properties: `Min`, `Max`, and `Grow`.

Min specifies the minimum size that a widget must receive:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Min.Set(units.Em(5))
    s.Background = colors.Scheme.InverseSurface
})
```

Min (and Max and Grow) can be specified for each dimension:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Min.Set(units.Em(10), units.Em(3))
    s.Background = colors.Scheme.InverseSurface
})
```

#### Grow

Grow makes a widget fill the available space up to Max:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Grow.Set(1, 1)
    s.Min.Set(units.Em(5))
    s.Background = colors.Scheme.InverseSurface
})
```

Max puts a constraint on the amount a widget can Grow:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Grow.Set(1, 1)
    s.Min.Set(units.Em(5))
    s.Max.Set(units.Em(10))
    s.Background = colors.Scheme.InverseSurface
})
```

In the example above, notice that the [[frame]] has a size of 10em in the X direction, but only 5em in the Y direction. That is because the widget has room to grow in the X direction and thus reaches the Max, but there are plenty of other widgets competing for space in the Y direction, so it stays at its Min.
