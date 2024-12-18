+++
Categories = ["Concepts"]
+++

**Value binding** allows you to link the value of a variable and the value of a [[#widgets|widget]], ensuring that they automatically stay up-to-date.

For example, the example in [[#updating]] above can also be written as:

```Go
count := 0
text := core.Bind(&count, core.NewText(b))
core.NewButton(b).SetText("Increment").OnClick(func(e events.Event) {
    count++
    text.Update()
})
```

That code uses [[doc:core.Bind]] to bind the value of the variable `count` to the [[text]] of the widget `text`, meaning that the text will be updated from the variable whenever [[doc:core.WidgetBase.Update]] is called.

You can use value binding with more than just text widgets; most widgets implement the [[doc:core.Value]] interface and thus support value binding. For example, this code uses value binding with a [[switch]] and a corresponding bool value:

```Go
on := true
core.Bind(&on, core.NewSwitch(b)).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, "The switch is now "+strconv.FormatBool(on))
})
```

Note that value binding goes both ways: not only is the value of the widget updated in [[doc:core.WidgetBase.Update]], the value of the bound variable is updated before [[doc:core.WidgetBase.OnChange]]. This two-way updating makes value binding very useful for creating interactive widgets that represent some underlying value.
