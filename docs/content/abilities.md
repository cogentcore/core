+++
Categories = ["Concepts"]
+++

**Abilities** are flags that specify what [[states]] and [[event]]s a [[widget]] can have. Abilities are stored in the [[style]] object.

You can add abilities (try hovering and clicking on the frame below):

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.SetAbilities(true, abilities.Hoverable, abilities.Activatable)
    s.Background = colors.Scheme.SurfaceContainerHigh
    s.Min.Set(units.Em(5))
})
```

You can remove abilities:

```Go
bt := core.NewButton(b).SetText("Not clickable")
bt.Styler(func(s *styles.Style) {
    s.SetAbilities(false, abilities.Activatable, abilities.DoubleClickable, abilities.TripleClickable)
})
bt.OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "This will never happen")
})
```

A [[states#disabled]] widget effectively has no abilities:

```Go
bt := core.NewButton(b).SetText("Disabled")
bt.SetEnabled(false)
```
