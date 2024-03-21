# Frames

Cogent Core provides customizable frames that can position content in many different ways and render a container. Frames are similar to layouts, but frames also render a container in addition to laying out elements. Frames support everything that layouts support. Also, frames grow in both directions by default, whereas layouts only grow in one direction by default.

You can make a frame and place elements inside of it:

```Go
fr := gi.NewFrame(parent)
gi.NewButton(fr).SetText("First")
gi.NewButton(fr).SetText("Second")
gi.NewButton(fr).SetText("Third")
```

You can add a background to a frame:

```Go
fr := gi.NewFrame(parent)
fr.Style(func(s *styles.Style) {
    s.Background = colors.C(colors.Scheme.Warn.Container)
})
gi.NewButton(fr).SetText("First")
gi.NewButton(fr).SetText("Second")
gi.NewButton(fr).SetText("Third")
```

You can add a border to a frame:

```Go
fr := gi.NewFrame(parent)
fr.Style(func(s *styles.Style) {
    s.Border.Width.Set(units.Dp(4))
    s.Border.Color.Set(colors.C(colors.Scheme.Outline))
})
gi.NewButton(fr).SetText("First")
gi.NewButton(fr).SetText("Second")
gi.NewButton(fr).SetText("Third")
```

You can make the corners of a frame rounded:

```Go
fr := gi.NewFrame(parent)
fr.Style(func(s *styles.Style) {
    s.Border.Radius = styles.BorderRadiusLarge
    s.Border.Width.Set(units.Dp(4))
    s.Border.Color.Set(colors.C(colors.Scheme.Outline))
})
gi.NewButton(fr).SetText("First")
gi.NewButton(fr).SetText("Second")
gi.NewButton(fr).SetText("Third")
```

Frames grow to fill the available space by default, but you can disable that:

```Go
fr := gi.NewFrame(parent)
fr.Style(func(s *styles.Style) {
    s.Grow.Set(0, 0)
    s.Border.Width.Set(units.Dp(4))
    s.Border.Color.Set(colors.C(colors.Scheme.Outline))
})
gi.NewButton(fr).SetText("First")
gi.NewButton(fr).SetText("Second")
gi.NewButton(fr).SetText("Third")
```
