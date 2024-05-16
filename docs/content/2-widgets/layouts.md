# Layouts

Cogent Core provides customizable layouts that can position content in many different ways. Layouts are similar to frames, but frames also render a container in addition to laying out elements. Frames support everything that layouts support. Also, layouts only grow in one direction by default, whereas frames grow in both directions by default.

You can make a layout and place elements inside of it:

```Go
ly := core.NewFrame(parent)
core.NewButton(ly).SetText("First")
core.NewButton(ly).SetText("Second")
core.NewButton(ly).SetText("Third")
```

You can position elements in a column instead of in a row:

```Go
ly := core.NewFrame(parent)
ly.Style(func(s *styles.Style) {
    s.Direction = styles.Column
})
core.NewButton(ly).SetText("First")
core.NewButton(ly).SetText("Second")
core.NewButton(ly).SetText("Third")
```

You can change the space between elements in a layout:

```Go
ly := core.NewFrame(parent)
ly.Style(func(s *styles.Style) {
    s.Gap.Set(units.Em(2))
})
core.NewButton(ly).SetText("First")
core.NewButton(ly).SetText("Second")
core.NewButton(ly).SetText("Third")
```

You can limit the size of a layout:

```Go
ly := core.NewFrame(parent)
ly.Style(func(s *styles.Style) {
    s.Max.X.Em(10)
})
core.NewButton(ly).SetText("First")
core.NewButton(ly).SetText("Second")
core.NewButton(ly).SetText("Third")
```

You can make a layout add scroll bars when it overflows:

```Go
ly := core.NewFrame(parent)
ly.Style(func(s *styles.Style) {
    s.Overflow.X = styles.OverflowAuto
    s.Max.X.Em(10)
})
core.NewButton(ly).SetText("First")
core.NewButton(ly).SetText("Second")
core.NewButton(ly).SetText("Third")
```

You can make a layout wrap when it overflows:

```Go
ly := core.NewFrame(parent)
ly.Style(func(s *styles.Style) {
    s.Wrap = true
    s.Max.X.Em(10)
})
core.NewButton(ly).SetText("First")
core.NewButton(ly).SetText("Second")
core.NewButton(ly).SetText("Third")
```

You can position elements in a grid:

```Go
ly := core.NewFrame(parent)
ly.Style(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 2
})
core.NewButton(ly).SetText("First")
core.NewButton(ly).SetText("Second")
core.NewButton(ly).SetText("Third")
core.NewButton(ly).SetText("Fourth")
```
