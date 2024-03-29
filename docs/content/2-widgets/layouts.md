# Layouts

Cogent Core provides customizable layouts that can position content in many different ways. Layouts are similar to frames, but frames also render a container in addition to laying out elements. Frames support everything that layouts support. Also, layouts only grow in one direction by default, whereas frames grow in both directions by default.

You can make a layout and place elements inside of it:

```Go
ly := gi.NewLayout(parent)
gi.NewButton(ly).SetText("First")
gi.NewButton(ly).SetText("Second")
gi.NewButton(ly).SetText("Third")
```

You can position elements in a column instead of in a row:

```Go
ly := gi.NewLayout(parent)
ly.Style(func(s *styles.Style) {
    s.Direction = styles.Column
})
gi.NewButton(ly).SetText("First")
gi.NewButton(ly).SetText("Second")
gi.NewButton(ly).SetText("Third")
```

You can change the space between elements in a layout:

```Go
ly := gi.NewLayout(parent)
ly.Style(func(s *styles.Style) {
    s.Gap.Set(units.Em(2))
})
gi.NewButton(ly).SetText("First")
gi.NewButton(ly).SetText("Second")
gi.NewButton(ly).SetText("Third")
```

You can limit the size of a layout:

```Go
ly := gi.NewLayout(parent)
ly.Style(func(s *styles.Style) {
    s.Max.X.Em(10)
})
gi.NewButton(ly).SetText("First")
gi.NewButton(ly).SetText("Second")
gi.NewButton(ly).SetText("Third")
```

You can make a layout add scroll bars when it overflows:

```Go
ly := gi.NewLayout(parent)
ly.Style(func(s *styles.Style) {
    s.Overflow.X = styles.OverflowAuto
    s.Max.X.Em(10)
})
gi.NewButton(ly).SetText("First")
gi.NewButton(ly).SetText("Second")
gi.NewButton(ly).SetText("Third")
```

You can make a layout wrap when it overflows:

```Go
ly := gi.NewLayout(parent)
ly.Style(func(s *styles.Style) {
    s.Wrap = true
    s.Max.X.Em(10)
})
gi.NewButton(ly).SetText("First")
gi.NewButton(ly).SetText("Second")
gi.NewButton(ly).SetText("Third")
```

You can position elements in a grid:

```Go
ly := gi.NewLayout(parent)
ly.Style(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 2
})
gi.NewButton(ly).SetText("First")
gi.NewButton(ly).SetText("Second")
gi.NewButton(ly).SetText("Third")
gi.NewButton(ly).SetText("Fourth")
```
