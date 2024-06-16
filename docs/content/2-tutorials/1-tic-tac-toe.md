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

Then, we will make it so that clicking on a button sets its icon to either close (X) or circle (O) based on an alternating variable `isX`, if the icon of the button has not already been set:

```Go
isX := true
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
    bt.OnClick(func(e events.Event) {
        if bt.Icon != icons.Blank {
            return
        }
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
for range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction).SetIcon(icons.Blank)
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Color.Set(colors.C(colors.Scheme.Outline))
        s.Border.Radius.Zero()
    })
    bt.OnClick(func(e events.Event) {
        if bt.Icon != icons.Blank {
            return
        }
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
