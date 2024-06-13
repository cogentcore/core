# Trees

Cogent Core provides interactive trees that allow you to display a nested tree structure. Users can use context menus and drag-and-drop to add, remove, and move tree nodes.

You can make a tree and add tree child nodes directly to it:

```Go
tv := core.NewTree(parent).SetText("Root")
core.NewTree(tv)
c2 := core.NewTree(tv)
core.NewTree(c2)
```

You can make a tree represent another [[tree.Node]] tree:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
core.NewTree(parent).SyncTree(n)
```

You can detect when the user changes the value of the tree value:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
core.NewTree(parent).SyncTree(n).OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, "Tree changed")
})
```

You can prevent the user from changing the tree:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
core.NewTree(parent).SyncTree(n).SetReadOnly(true)
```

When you use [[core.NewValue]] with a [[tree.Node]] tree node value, it will create a button that opens an interactive inspector of that node:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
// core.NewValue(parent, n)
```
