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
tree := ki.NewRoot[*ki.Node]("Root")
ki.New[*ki.Node](tree, "Child 1")
c2 := ki.New[*ki.Node](tree, "Child 2")
ki.New[*ki.Node](c2, "Nested child")
giv.NewTreeView(parent).SyncTree(tree)
```

You can detect when the user changes the value of the tree value:

```Go
tree := ki.NewRoot[*ki.Node]("Root")
ki.New[*ki.Node](tree, "Child 1")
c2 := ki.New[*ki.Node](tree, "Child 2")
ki.New[*ki.Node](c2, "Nested child")
giv.NewTreeView(parent).SyncTree(tree).OnChange(func(e events.Event) {
    gi.MessageSnackbar(parent, "Tree view changed")
})
```

You can prevent the user from changing the tree:

```Go
tree := ki.NewRoot[*ki.Node]("Root")
ki.New[*ki.Node](tree, "Child 1")
c2 := ki.New[*ki.Node](tree, "Child 2")
ki.New[*ki.Node](c2, "Nested child")
giv.NewTreeView(parent).SyncTree(tree).SetReadOnly(true)
```
