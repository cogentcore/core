# Keyed lists

Cogent Core provides interactive keyed lists that allow you to display a map value to users and have them edit it. Users can use context menus to add, remove, and move rows.

You can make a keyed list from any map pointer:

```Go
core.NewKeyedList(parent).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
```

You can make a keyed list that fits in one line:

```Go
core.NewKeyedList(parent).SetInline(true).SetMap(&map[string]int{"Go": 1, "C++": 3})
```

You can detect when the user changes the value of the map:

```Go
m := map[string]int{"Go": 1, "C++": 3, "Python": 5}
core.NewKeyedList(parent).SetMap(&m).OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, fmt.Sprintf("Map: %v", m))
})
```

You can prevent the user from editing the map:

```Go
core.NewKeyedList(parent).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5}).SetReadOnly(true)
```

You can make a keyed list with elements of any type:

```Go
core.NewKeyedList(parent).SetMap(&map[string]any{"Go": 1, "C++": "C-like", "Python": true})
```

When you use [[core.NewValue]] with a map value, it will create an inline keyed list if the map has two or fewer elements:

```Go
// core.NewValue(parent, &map[string]int{"Go": 1, "C++": 3})
```

Otherwise, it will create a button that opens a dialog with a normal keyed list:

```Go
// core.NewValue(parent, &map[string]int{"Go": 1, "C++": 3, "Python": 5})
```