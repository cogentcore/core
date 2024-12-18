+++
Categories = ["Concepts"]
+++

**Styling** allows you to easily customize the appearance of [[#widgets|widgets]] at any level.

You can change any style properties of a widget:

```Go
core.NewText(b).SetText("Bold text").Styler(func(s *styles.Style) {
    s.Font.Weight = styles.WeightBold
})
```

You can change the [[color]]s of a widget using Cogent Core's dynamic [[color#scheme|color scheme]] system:

```Go
core.NewButton(b).SetText("Success button").Styler(func(s *styles.Style) {
    s.Background = colors.Scheme.Success.Base
    s.Color = colors.Scheme.Success.On
})
```

You can change the size of a widget using Cogent Core's flexible [[unit]] system:

```Go
core.NewFrame(b).Styler(func(s *styles.Style) {
    s.Min.Set(units.Dp(50))
    s.Background = colors.Scheme.Primary.Base
})
```
