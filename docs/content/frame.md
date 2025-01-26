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

## Styles

See the [[styles]] page for documentation on important [[style]] properties for frames, especially the [[styles#layout]] styles.

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
