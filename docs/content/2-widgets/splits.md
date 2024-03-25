# Splits

Cogent Core provides customizable splits, which allow you to divide space among widgets and have the user customize how much space each widget gets using draggable handles.

You can make splits without any custom options:

```Go
sp := gi.NewSplits(parent)
gi.NewLabel(sp).SetText("First")
gi.NewLabel(sp).SetText("Second")
```

You can add as many items as you want to splits:

```Go
sp := gi.NewSplits(parent)
gi.NewLabel(sp).SetText("First")
gi.NewLabel(sp).SetText("Second")
gi.NewLabel(sp).SetText("Third")
gi.NewLabel(sp).SetText("Fourth")
```

You can change the default amount of space that each widget receives:

```Go
sp := gi.NewSplits(parent).SetSplits(0.2, 0.8)
gi.NewLabel(sp).SetText("First")
gi.NewLabel(sp).SetText("Second")
```

You can arrange widgets in a column (by default, split widgets are arranged in a row on wide screens and a column on compact screens):

```Go
sp := gi.NewSplits(parent)
sp.Style(func(s *styles.Style) {
    s.Direction = styles.Column
})
gi.NewLabel(sp).SetText("First")
gi.NewLabel(sp).SetText("Second")
```

You can arrange widgets in a row:

```Go
sp := gi.NewSplits(parent)
sp.Style(func(s *styles.Style) {
    s.Direction = styles.Row
})
gi.NewLabel(sp).SetText("First")
gi.NewLabel(sp).SetText("Second")
```
