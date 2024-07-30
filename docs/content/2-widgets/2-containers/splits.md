Cogent Core provides customizable splits, which allow you to divide space among widgets and have the user customize how much space each widget gets using draggable handles.

You can make splits without any custom options:

```Go
sp := core.NewSplits(b)
core.NewText(sp).SetText("First")
core.NewText(sp).SetText("Second")
```

You can add as much content as you want to each splits item:

```Go
sp := core.NewSplits(b)
first := core.NewFrame(sp)
core.NewButton(first).SetText("First")
core.NewTextField(first).SetText("First")
second := core.NewFrame(sp)
core.NewButton(second).SetText("Second")
core.NewTextField(second).SetText("Second")
```

You can add as many items as you want to splits:

```Go
sp := core.NewSplits(b)
core.NewText(sp).SetText("First")
core.NewText(sp).SetText("Second")
core.NewText(sp).SetText("Third")
core.NewText(sp).SetText("Fourth")
```

You can change the default amount of space that each widget receives:

```Go
sp := core.NewSplits(b).SetSplits(0.2, 0.8)
core.NewText(sp).SetText("First")
core.NewText(sp).SetText("Second")
```

You can use [[core.Splits.Tiles]] to create 2D arrangements of widgets, without having to nest multiple splits.  This is simpler because it operates on the same list of child widgets, whereas the nesting approach requires moving child widgets around to switch between different arrangements.

```Go
sp := core.NewSplits(b)
core.NewText(sp).SetText("First")
core.NewText(sp).SetText("Second")
core.NewText(sp).SetText("Third")
core.NewText(sp).SetText("Fourth")
sp.SetTiles(core.TileSpan, core.TileSecondLong)
```

You can arrange widgets in a column (by default, split widgets are arranged in a row on wide screens and a column on compact screens):

```Go
sp := core.NewSplits(b)
sp.Styler(func(s *styles.Style) {
    s.Direction = styles.Column
})
core.NewText(sp).SetText("First")
core.NewText(sp).SetText("Second")
```

You can arrange widgets in a row:

```Go
sp := core.NewSplits(b)
sp.Styler(func(s *styles.Style) {
    s.Direction = styles.Row
})
core.NewText(sp).SetText("First")
core.NewText(sp).SetText("Second")
```
