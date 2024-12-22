+++
Categories = ["Concepts"]
+++

**Abilities** are flags that specify what [[states]] and [[event]]s a [[widget]] can have. Abilities are stored in the [[style]] object.

You can set abilities (try hovering and pressing on the frame below):

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.SetAbilities(true, abilities.Hoverable, abilities.Activatable)
    s.Background = colors.Scheme.SurfaceContainerHigh
    s.Min.Set(units.Em(5))
})
```
