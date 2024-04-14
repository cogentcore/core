# Tooltips

Cogent Core provides tooltips that give the user additional information about a widget. Users can view tooltips by hovering over a widget for 500 milliseconds or pressing down on a widget for 500 milliseconds, so tooltips work on all platforms.

You can set the tooltip of any widget:

```Go
core.NewButton(parent).SetIcon(icons.Add).SetTooltip("Add a new item to the list")
```

Some widgets automatically add certain information to their tooltip by implementing the [[core.Widget.WidgetTooltip]] method, like sliders:

```Go
core.NewSlider(parent)
```
