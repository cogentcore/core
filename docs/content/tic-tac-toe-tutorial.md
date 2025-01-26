+++
Name = "Tic-tac-toe tutorial"
Categories = ["Tutorials"]
+++

This [[tutorials|tutorial]] shows how to make a simple **tic-tac-toe** game.

First, we will make a 3x3 grid of action [[button]]s with [[styles#border]]s:

```Go
grid := core.NewFrame(b)
grid.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 3
})
for range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction)
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Radius.Zero()
        s.Min.Set(units.Em(2))
    })
}
```

Then, we will make a `squares` array that keeps track of the value of each square, and we will make it so that [[events#click]]ing on a button sets its value in the array to either X or O based on an alternating variable `current`. We also add an [[update]]r to update the text of each button based on its value in the array. Also, we add a reset button that clears all of the squares.

```Go
current := "X"
squares := [9]string{}
grid := core.NewFrame(b)
grid.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 3
})
for i := range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction)
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Radius.Zero()
        s.Min.Set(units.Em(2))
    })
    bt.OnClick(func(e events.Event) {
        // don't set squares that already have a value
        if squares[i] != "" {
            return
        }
        squares[i] = current
        if current == "X" {
            current = "O"
        } else {
            current = "X"
        }
        bt.Update()
    })
    bt.Updater(func() {
        bt.SetText(squares[i])
    })
}
core.NewButton(b).SetText("Reset").OnClick(func(e events.Event) {
    squares = [9]string{}
    current = "X"
    grid.Update()
})
```

Finally, we will add status [[text]] that updates according to the current state of the game. This includes checking if there is a winner and displaying it if there is one.

```Go
current := "X"
squares := [9]string{}
status := core.NewText(b)
status.Updater(func() {
    sets := [][3]int{ // possible sets of three that result in a win
        {0, 1, 2}, {3, 4, 5}, {6, 7, 8}, {0, 3, 6}, {1, 4, 7}, {2, 5, 8}, {0, 4, 8}, {2, 4, 6},
    }
    // check if someone has won
    for _, set := range sets {
        if squares[set[0]] != "" && squares[set[0]] == squares[set[1]] && squares[set[0]] == squares[set[2]] {
            status.SetText(squares[set[0]]+" wins!")
            current = ""
            return
        }
    }
    status.SetText("Next player: "+current)
})
grid := core.NewFrame(b)
grid.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 3
})
for i := range 9 {
    bt := core.NewButton(grid).SetType(core.ButtonAction)
    bt.Styler(func(s *styles.Style) {
        s.Border.Width.Set(units.Dp(1))
        s.Border.Radius.Zero()
        s.Min.Set(units.Em(2))
    })
    bt.OnClick(func(e events.Event) {
        // don't set squares if they already have a value or the game is over
        if squares[i] != "" || current == "" {
            return
        }
        squares[i] = current
        if current == "X" {
            current = "O"
        } else {
            current = "X"
        }
        bt.Update()
        status.Update()
    })
    bt.Updater(func() {
        bt.SetText(squares[i])
    })
}
core.NewButton(b).SetText("Reset").OnClick(func(e events.Event) {
    squares = [9]string{}
    current = "X"
    grid.Update()
    status.Update()
})
```
