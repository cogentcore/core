+++
Categories = ["Widgets"]
+++

**Splits** allow you to divide space among [[widget]]s and have the user customize how much space each widget gets using draggable handles.

## Properties

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

You can use [[doc:core.Splits.Tiles]] to create 2D arrangements of widgets without having to nest multiple splits:

```Go
sp := core.NewSplits(b)
core.NewText(sp).SetText("First")
core.NewText(sp).SetText("Second")
core.NewText(sp).SetText("Third")
core.NewText(sp).SetText("Fourth")
sp.SetTiles(core.TileSpan, core.TileSecondLong)
```

## Styles

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
