# Tree views

Cogent Core provides interactive tree views that allow you to display a nested tree structure. Users can use context menus and drag-and-drop to add, remove, and move tree nodes.

You can make a tree view and add nodes directly to it:

```Go
tv := giv.NewTreeView(parent).SetText("Root")
giv.NewTreeView(tv, "Child 1")
n2 := giv.NewTreeView(tv, "Child 2")
giv.NewTreeView(n2, "Nested child")
```
