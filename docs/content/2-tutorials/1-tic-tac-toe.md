# Tic-tac-toe

This tutorial shows how to make a simple tic-tac-toe game using Cogent Core.

First, we will make a 3x3 grid of action buttons with borders:

```Go
grid := core.NewFrame(parent)
grid.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 3
    s.Gap.Zero()
})
for range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction).SetText(" ")
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Color.Set(colors.C(colors.Scheme.Outline))
        s.Border.Radius.Zero()
    })
}
```

Then, we will make it so that clicking on a button sets its text to either X or O based on an alternating variable `current`. We also add a `squares` array that keeps track of the value of each square. This allows us to prevent users from setting a square if it is already set.

```Go
current := "X"
squares := [9]string{}
grid := core.NewFrame(parent)
grid.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 3
    s.Gap.Zero()
})
for i := range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction).SetText(" ")
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Color.Set(colors.C(colors.Scheme.Outline))
        s.Border.Radius.Zero()
    })
    bt.OnClick(func(e events.Event) {
        if squares[i] != "" {
            return
        }
        squares[i] = current
        bt.SetText(current).Update()
        if current == "X" {
            current = "O"
        } else {
            current = "X"
        }
    })
}
```

Finally, we will add status text that updates according to the current state of the game:

```Go
current := "X"
squares := [9]string{}
status := core.NewText(parent)
status.Updater(func() {
    status.SetText("Next player: "+current)
})
grid := core.NewFrame(parent)
grid.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 3
    s.Gap.Zero()
})
for i := range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction).SetText(" ")
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Color.Set(colors.C(colors.Scheme.Outline))
        s.Border.Radius.Zero()
    })
    bt.OnClick(func(e events.Event) {
        if squares[i] != "" {
            return
        }
        squares[i] = current
        bt.SetText(current).Update()
        if current == "X" {
            current = "O"
        } else {
            current = "X"
        }
        status.Update()
    })
}
```
