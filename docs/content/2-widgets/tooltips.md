# Tooltips

Cogent Core provides tooltips that give the user additional information about a widget. Users can view tooltips by hovering over a widget for 500 milliseconds or pressing down on a widget for 500 milliseconds, so tooltips work on all platforms.

You can set the tooltip of any widget:

```Go
gi.NewButton(parent).SetIcon(icons.Add).SetTooltip("Add a new item to the list")
```
