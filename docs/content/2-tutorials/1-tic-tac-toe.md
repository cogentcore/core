# Tic-tac-toe

This tutorial shows how to make a simple tic-tac-toe game using Cogent Core.

First, we will make a 3x3 grid of action buttons with borders and blank icons:

```Go
grid := core.NewFrame(parent)
grid.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 3
    s.Gap.Zero()
})
for range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction).SetIcon(icons.Blank)
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Color.Set(colors.C(colors.Scheme.Outline))
        s.Border.Radius.Zero()
    })
}
```

Then, we will make it so that clicking on a button sets its icon to either close (X) or circle (O) based on an alternating variable `isX`. We also add a `squares` map that keeps track of the state of each square: true for X, false for O, and unspecified for empty. This allows us to prevent users from setting a square if it is already set.

```Go
isX := true
squares := map[int]bool{}
grid := core.NewFrame(parent)
grid.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 3
    s.Gap.Zero()
})
for i := range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction).SetIcon(icons.Blank)
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Color.Set(colors.C(colors.Scheme.Outline))
        s.Border.Radius.Zero()
    })
    bt.OnClick(func(e events.Event) {
        if _, set := squares[i]; set {
            return
        }
        squares[i] = isX
        if isX {
            bt.SetIcon(icons.Close)
        } else {
            bt.SetIcon(icons.Circle)
        }
        bt.Update()
        isX = !isX
    })
}
```

Finally, we will add status text that updates according to the current state of the game:

```Go
isX := true
squares := map[int]bool{}
status := core.NewText(parent)
status.Updater(func() {
    if isX {
        status.SetText("Next player: X")
    } else {
        status.SetText("Next player: O")
    }
})
grid := core.NewFrame(parent)
grid.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 3
    s.Gap.Zero()
})
for i := range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction).SetIcon(icons.Blank)
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Color.Set(colors.C(colors.Scheme.Outline))
        s.Border.Radius.Zero()
    })
    bt.OnClick(func(e events.Event) {
        if _, set := squares[i]; set {
            return
        }
        squares[i] = isX
        if isX {
            bt.SetIcon(icons.Close)
        } else {
            bt.SetIcon(icons.Circle)
        }
        isX = !isX
        bt.Update()
        status.Update()
    })
}
```
