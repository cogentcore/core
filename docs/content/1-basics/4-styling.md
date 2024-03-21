# Styling

Cogent Core provides a versatile styling system that allows you to easily customize the appearance of widgets at any level.

You can change any style properties of a widget:

```Go
gi.NewLabel(parent).SetText("Bold text").Style(func(s *styles.Style) {
    s.Font.Weight = styles.WeightBold
})
```

You can change the colors of a widget using Cogent Core's dynamic color scheme system:

```Go
gi.NewButton(parent).SetText("Success button").Style(func(s *styles.Style) {
    s.Background = colors.C(colors.Scheme.Success.Base)
    s.Color = colors.C(colors.Scheme.Success.On)
})
```

You can use Cogent Core's flexible unit system to specify sizing properties of a widget in one of many different units. The most common units are `dp` (density-independent pixels, or 1/160th of 1 inch), and `em` (the font size of the element).

```Go
gi.NewLabel(parent).SetText("Big text").Style(func(s *styles.Style) {
    s.Font.Size.Dp(21)
})
```

Throughout the documentation for different widgets, you will learn how to use various other styling properties.
