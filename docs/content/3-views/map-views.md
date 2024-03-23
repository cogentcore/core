# Map views

Cogent Core provides interactive map views that allow you to display a map value to users and have them edit it. Users can use context menus to add, remove, and copy rows.

You can make a map view from any map pointer:

```Go
giv.NewMapView(parent).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
```

You can detect when the user changes the value of the map:

```Go
m := map[string]int{"Go": 1, "C++": 3, "Python": 5}
giv.NewMapView(parent).SetMap(&m).OnChange(func(e events.Event) {
    gi.MessageSnackbar(parent, fmt.Sprintf("Map: %v", m))
})
```

You can prevent the user from editing the map:

```Go
giv.NewMapView(parent).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5}).SetReadOnly(true)
```

You can make a map view that fits in one line:

```Go
giv.NewMapViewInline(parent).SetMap(&map[string]int{"Go": 1, "C++": 3})
```

Inline map views support everything that normal map views do, including everything documented above.

When you use [[giv.NewValue]] with a map value, it will create an inline map view if the map has two or fewer elements:

```Go
giv.NewValue(parent, &map[string]int{"Go": 1, "C++": 3})
```

Otherwise, it will create a button that opens a dialog with a normal map view:

```Go
giv.NewValue(parent, &map[string]int{"Go": 1, "C++": 3, "Python": 5})
```
