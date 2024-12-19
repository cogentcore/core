+++
Categories = ["Concepts"]
+++

There are several ways to dynamically **update** the content of an [[app]].

The simplest way to update a [[widget]] is to call [[doc:core.WidgetBase.Update]] after changing any of its properties:

```Go
count := 0
text := core.NewText(b).SetText("0")
core.NewButton(b).SetText("Increment").OnClick(func(e events.Event) {
    count++
    text.SetText(strconv.Itoa(count)).Update()
})
```

You can also register a [[doc:tree.NodeBase.Updater]] that will get called when the widget is updated. This can allow you to more closely couple widgets with their updating logic:

```Go
count := 0
text := core.NewText(b)
text.Updater(func() {
    text.SetText(strconv.Itoa(count))
})
core.NewButton(b).SetText("Increment").OnClick(func(e events.Event) {
    count++
    text.Update()
})
```
