# Value binding

Cogent Core provides a powerful value binding mechanism that allows you to link the value of a variable and the value of a widget, ensuring that they automatically stay up-to-date.

For example, the example in the [previous updating page](updating) can also be written as:

```Go
count := 0
text := core.Bind(&count, core.NewText(parent))
core.NewButton(parent).SetText("Increment").OnClick(func(e events.Event) {
    count++
    text.Update()
})
```

That code uses [[core.Bind]] to bind the value of the variable `count` to the text of the widget `text`, meaning that the text will be updated from the variable whenever [[core.WidgetBase.Update]] is called.
