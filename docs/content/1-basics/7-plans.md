The previous two pages cover how to update the properties of a widget, but what if you want to update the structure of a widget? To answer that question, Cogent Core provides [[tree.Plan]], a mechanism for specifying what the children of a widget should be, which is then used to automatically update the actual children to reflect that.

For example, this code uses [[tree.Plan]] through [[core.WidgetBase.Maker]] to dynamically update the number of buttons in a frame:

```Go
number := 3
spinner := core.Bind(&number, core.NewSpinner(b)).SetMin(0)
buttons := core.NewFrame(b)
buttons.Maker(func(p *tree.Plan) {
    for i := range number {
        tree.AddAt(p, strconv.Itoa(i), func(w *core.Button) {
            w.SetText(strconv.Itoa(i))
        })
    }
})
spinner.OnChange(func(e events.Event) {
    buttons.Update()
})
```

Plans are a powerful tool that are critical for some widgets such as those that need to dynamically manage hundreds of children in a convenient and performant way. They aren't always necessary, but you will find them being used a lot in complicated apps, and you will see more examples of them in the rest of this documentation.
