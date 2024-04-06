# Tree views

Cogent Core provides interactive tree views that allow you to display a nested tree structure. Users can use context menus and drag-and-drop to add, remove, and move tree nodes.

You can make a tree view and add tree view child nodes directly to it:

```Go
tv := giv.NewTreeView(parent).SetText("Root")
giv.NewTreeView(tv, "Child 1")
c2 := giv.NewTreeView(tv, "Child 2")
giv.NewTreeView(c2, "Nested child")
```

You can make a tree view represent another [[ki.Ki]] tree:

```Go
n := ki.NewRoot[*ki.NodeBase]("Root")
ki.New[*ki.NodeBase](n, "Child 1")
c2 := ki.New[*ki.NodeBase](n, "Child 2")
ki.New[*ki.NodeBase](c2, "Nested child")
giv.NewTreeView(parent).SyncTree(n)
```

You can detect when the user changes the value of the tree value:

```Go
n := ki.NewRoot[*ki.NodeBase]("Root")
ki.New[*ki.NodeBase](n, "Child 1")
c2 := ki.New[*ki.NodeBase](n, "Child 2")
ki.New[*ki.NodeBase](c2, "Nested child")
giv.NewTreeView(parent).SyncTree(n).OnChange(func(e events.Event) {
    gi.MessageSnackbar(parent, "Tree view changed")
})
```

You can prevent the user from changing the tree:

```Go
n := ki.NewRoot[*ki.NodeBase]("Root")
ki.New[*ki.NodeBase](n, "Child 1")
c2 := ki.New[*ki.NodeBase](n, "Child 2")
ki.New[*ki.NodeBase](c2, "Nested child")
giv.NewTreeView(parent).SyncTree(n).SetReadOnly(true)
```

When you use [[giv.NewValue]] with a [[ki.Ki]] tree node value, it will create a button that opens an interactive inspector of that node:

```Go
n := ki.NewRoot[*ki.NodeBase]("Root")
ki.New[*ki.NodeBase](n, "Child 1")
c2 := ki.New[*ki.NodeBase](n, "Child 2")
ki.New[*ki.NodeBase](c2, "Nested child")
giv.NewValue(parent, n)
```
