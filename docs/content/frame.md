+++
Categories = ["Widgets"]
+++

A **frame** is a [[widget]] that can position content in many different ways and [[render]] a container. Frames are the base container type, and almost all [[widgets]] are contained within some kind of frame.

You can make a frame and place elements inside of it:

```Go
fr := core.NewFrame(b)
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

## Styling

You can position elements in a column instead of in a row:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Direction = styles.Column
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can change the space between elements in a frame:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Gap.Set(units.Em(2))
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can limit the size of a frame:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Max.X.Em(10)
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can make a frame add scrollbars when it overflows:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Overflow.X = styles.OverflowAuto
    s.Max.X.Em(10)
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can make a frame wrap when it overflows:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Wrap = true
    s.Max.X.Em(10)
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can position elements in a grid:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 2
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
core.NewButton(fr).SetText("Fourth")
```

You can add a background to a frame:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Background = colors.Scheme.Warn.Container
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can add a gradient background to a frame:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Background = gradient.NewLinear().AddStop(colors.Yellow, 0).AddStop(colors.Orange, 0.5).AddStop(colors.Red, 1)
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can add a border to a frame:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Border.Width.Set(units.Dp(4))
    s.Border.Color.Set(colors.Scheme.Outline)
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can make the corners of a frame rounded:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Border.Radius = styles.BorderRadiusLarge
    s.Border.Width.Set(units.Dp(4))
    s.Border.Color.Set(colors.Scheme.Outline)
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can make a frame grow to fill the available space:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Grow.Set(1, 1)
    s.Border.Width.Set(units.Dp(4))
    s.Border.Color.Set(colors.Scheme.Outline)
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

## Separator

You can add a separator to separate elements inside of a frame:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Direction = styles.Column
})
core.NewButton(fr).SetText("First")
core.NewSeparator(fr)
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

Separators can go in either direction:

```Go
fr := core.NewFrame(b)
core.NewButton(fr).SetText("First")
sp := core.NewSeparator(fr)
sp.Styler(func(s *styles.Style) {
    s.Direction = styles.Column
})
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

## Stretch

You can add a stretch to create a growing gap between elements inside of a frame:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Grow.Set(1, 0)
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewStretch(fr)
core.NewButton(fr).SetText("Third")
```
