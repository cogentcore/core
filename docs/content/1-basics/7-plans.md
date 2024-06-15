# Plans

The previous two pages cover how to update the properties of a widget, but what if you want to update the structure of a widget? To answer that question, Cogent Core provides [[core.Plan]]: a mechanism for specifying what the children of a widget *should* be, and then automatically updating the actual children to reflect that.

For example, this code uses [[core.Plan]] through [[core.WidgetBase.Maker]] to dynamically update the number of buttons in a frame:

```Go
number := 3
spinner := core.Bind(&number, core.NewSpinner(parent))
buttons := core.NewFrame(parent)
buttons.Maker(func(p *core.Plan) {
    for i := range number {
        core.AddAt(p, strconv.Itoa(i), func(w *core.Button) {
            w.SetText(strconv.Itoa(i))
        })
    }
})
spinner.OnChange(func(e events.Event) {
    buttons.Update()
})
```
