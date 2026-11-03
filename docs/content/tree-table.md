+++
Categories = ["Widgets"]
+++

A **tree table** is a [[widget]] that allows you to display a tree of structs to users as a table and have them edit it ([[doc:core.TreeTable]]). It combines a [[table]] and a [[tree]]: the fields are the columns and the items are the rows, as in a table, and the first column shows the tree, with the items indented under their parents and branches that open and close them. Users can use [[context menu]]s and [[events#drag-and-drop]] to add, remove, and move items, including into and out of other items.

For a slice of structs without children, use a [[table]] instead. For a tree of nodes without columns, use a [[tree]].

## Properties

You can make a tree table from a pointer to a slice of pointers to structs, where the children of an item are in its field named `Children`:

```Go
type task struct {
    Name     string
    Done     bool
    Children []*task
}
tasks := []*task{
    {Name: "Work", Children: []*task{{Name: "Email", Done: true}, {Name: "Report"}}},
    {Name: "Home", Children: []*task{{Name: "Dishes"}}},
}
core.NewTreeTable(b).SetItems(&tasks)
```

The children can also be in a field with a `treetable:"children"` struct tag, which can be a `[]any` of pointers to the same struct type, as needed for struct types made with `reflect.StructOf`. A bool field with a `treetable:"closed"` struct tag records whether each item is closed, so that it is saved with the items.

You can prevent users from editing a tree table, in which case the left and right arrow keys close and open the selected item:

```Go
type task struct {
    Name     string
    Done     bool
    Children []*task
}
tasks := []*task{
    {Name: "Work", Children: []*task{{Name: "Email", Done: true}, {Name: "Report"}}},
    {Name: "Home", Children: []*task{{Name: "Dishes"}}},
}
core.NewTreeTable(b).SetItems(&tasks).SetReadOnly(true)
```

## Events

You can detect when a user [[events#change]]s the items of a tree table:

```Go
type task struct {
    Name     string
    Done     bool
    Children []*task
}
tasks := []*task{
    {Name: "Work", Children: []*task{{Name: "Email", Done: true}, {Name: "Report"}}},
}
core.NewTreeTable(b).SetItems(&tasks).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Tasks under work: %d", len(tasks[0].Children)))
})
```
