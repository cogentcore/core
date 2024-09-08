Cogent Core provides interactive trees that allow you to display a nested tree structure. Users can use context menus and drag-and-drop to add, remove, and move tree nodes. See [file trees](../other/file-trees) for trees designed to display file structures.

You can make a tree and add tree child nodes directly to it:

```Go
tr := core.NewTree(b).SetText("Root")
core.NewTree(tr)
c2 := core.NewTree(tr)
core.NewTree(c2)
```

You can make a tree represent another [[tree.Node]] tree:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
core.NewTree(b).SyncTree(n)
```

You can detect when the user changes the value of a tree:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
core.NewTree(b).SyncTree(n).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, "Tree changed")
})
```

You can prevent the user from changing a tree:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
core.NewTree(b).SyncTree(n).SetReadOnly(true)
```

You can add an initialization function that is called automatically with each tree node:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
tr := core.NewTree(b)
tr.SetTreeInit(func(tr *core.Tree) {
    tr.AddContextMenu(func(m *core.Scene) {
        core.NewButton(m).SetText("My button")
    })
})
tr.SyncTree(n)
```

You can make a button that opens an interactive inspector of a tree:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
core.NewTreeButton(b).SetTree(n)
```
