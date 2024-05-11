# Tree views

Cogent Core provides interactive tree views that allow you to display a nested tree structure. Users can use context menus and drag-and-drop to add, remove, and move tree nodes.

You can make a tree view and add tree view child nodes directly to it:

```Go
tv := views.NewTreeView(parent).SetText("Root")
views.NewTreeView(tv)
c2 := views.NewTreeView(tv)
views.NewTreeView(c2)
```

You can make a tree view represent another [[tree.Node]] tree:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
views.NewTreeView(parent).SyncTree(n)
```

You can detect when the user changes the value of the tree value:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
views.NewTreeView(parent).SyncTree(n).OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, "Tree view changed")
})
```

You can prevent the user from changing the tree:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
views.NewTreeView(parent).SyncTree(n).SetReadOnly(true)
```

When you use [[views.NewValue]] with a [[tree.Node]] tree node value, it will create a button that opens an interactive inspector of that node:

```Go
n := tree.NewNodeBase()
tree.NewNodeBase(n)
c2 := tree.NewNodeBase(n)
tree.NewNodeBase(c2)
views.NewValue(parent, n)
```
