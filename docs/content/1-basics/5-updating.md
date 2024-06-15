# Updating

Cogent Core provides several ways to dynamically update the content of an app.

The simplest way to update a widget is to call [[core.WidgetBase.Update]] after changing any of its properties:

```Go
count := 0
text := core.NewText(parent).SetText("0")
core.NewButton(parent).SetText("Increment").OnClick(func(e events.Event) {
    count++
    text.SetText(strconv.Itoa(count)).Update()
})
```
