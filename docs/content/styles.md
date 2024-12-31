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
