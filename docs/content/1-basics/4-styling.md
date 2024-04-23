# Styling

Cogent Core provides a versatile styling system that allows you to easily customize the appearance of widgets at any level.

You can change any style properties of a widget:

```Go
core.NewText(parent).SetText("Bold text").Style(func(s *styles.Style) {
    s.Font.Weight = styles.WeightBold
})
```

You can change the colors of a widget using Cogent Core's dynamic color scheme system:

```Go
core.NewButton(parent).SetText("Success button").Style(func(s *styles.Style) {
    s.Background = colors.C(colors.Scheme.Success.Base)
    s.Color = colors.C(colors.Scheme.Success.On)
})
```

You can change the size of a widget using Cogent Core's flexible unit system:

```Go
core.NewBox(parent).Style(func(s *styles.Style) {
    s.Min.Set(units.Dp(50))
    s.Background = colors.C(colors.Scheme.Primary.Base)
})
```

Throughout the documentation for different widgets, you will learn how to use various other styling properties. This is just an introduction to the basic structure of styling.
