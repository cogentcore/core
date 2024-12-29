+++
Categories = ["Tutorials"]
+++

This [[tutorials|tutorial]] shows how to make a **todo list** app.

We will represent todo list items using an `item` struct type:

```Go
type item struct {
    Done bool
    Task string
}
```

We can create a slice of these items and then represent them with a [[table]] widget:

```Go
type item struct {
	Done bool `display:"checkbox"`
	Task string
}
items := []item{{Task: "Code"}, {Task: "Eat"}}
core.NewTable(b).SetSlice(&items)
```

We can add a [[button]] for adding a new todo list item:

```Go
type item struct {
	Done bool `display:"checkbox"`
	Task string
}
items := []item{{Task: "Code"}, {Task: "Eat"}}
var table *core.Table
core.NewButton(b).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
    table.NewAt(0)
})
table = core.NewTable(b).SetSlice(&items)
```
