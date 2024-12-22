+++
Categories = ["Concepts"]
+++

**States** are flags that describe the current transient state of a [[widget]] based on the [[event]]s it has received. States are stored in the [[style]] object. They are similar to [CSS pseudo-classes](https://developer.mozilla.org/en-US/docs/Web/CSS/Pseudo-classes).

You can style an element based on its state:

```Go
bt := core.NewButton(b).SetText("Hover")
bt.Styler(func(s *styles.Style) {
    if s.Is(states.Hovered) {
        s.Background = colors.Scheme.Success.Base
    }
})
```
