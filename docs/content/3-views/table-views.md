# Table views

Cogent Core provides interactive table views that allow you to display a slice of structs to users as a table and have them edit it. Users can use context menus and drag-and-drop to add, remove, and copy rows. Also, users can sort the table by clicking on columns.

You can make a table view from any pointer to a slice of structs:

```Go
type language struct {
    Name   string
    Rating int
}
giv.NewTableView(parent).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can detect when the user changes the value of the table:

```Go
type language struct {
    Name   string
    Rating int
}
sl := []language{{"Go", 10}, {"Python", 5}}
giv.NewTableView(parent).SetSlice(&sl).OnChange(func(e events.Event) {
    gi.MessageSnackbar(parent, fmt.Sprintf("Languages: %v", sl))
})
```

You can hide certain fields from the user:

```Go
type language struct {
    Name   string
    Rating int `view:"-"`
}
giv.NewTableView(parent).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can also use the `tableview` struct tag, which overrides the `view` struct tag. This allows you to have a struct field displayed in a struct view but not a table view, or vise versa:

```Go
type language struct {
    Name   string
    Rating int `view:"-" tableview:"+"`
}
giv.NewTableView(parent).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```
