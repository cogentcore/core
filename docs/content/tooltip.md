+++
Categories = ["Widgets"]
+++

**Tooltips** give the user additional information about a [[widget]]. Users can view tooltips by [[events#long hover]]ing over a widget for 500 milliseconds or [[events#long press]]ing down on a widget for 500 milliseconds, so tooltips work on all platforms.

You can set the tooltip of any widget:

```Go
core.NewButton(b).SetIcon(icons.Add).SetTooltip("Add a new item to the list")
```

Some widgets automatically add certain information to their tooltip by implementing the [[doc:core.Widget.WidgetTooltip]] method, like [[slider]]s:

```Go
core.NewSlider(b)
```
