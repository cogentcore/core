+++
Categories = ["Concepts"]
+++

The previous two sections cover how to update the properties of a [[widget]], but what if you want to update the structure of a widget? To answer that question, Cogent Core provides **plans**, a mechanism for specifying what the children of a widget should be, which is then used to automatically update the actual children to reflect that.

For example, this code uses [[doc:tree.Plan]] through [[doc:tree.NodeBase.Maker]] to dynamically update the number of [[button]]s in a [[frame]]:

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

## Plan logic

A `Plan` is a list (slice) of [[doc:tree.PlanItem]]s that specify all of the children for a given widget.

Each item must have a unique name, specified in the `PlanItem`, which is used for updating the children in an efficient way to ensure that the widget actually has the correct children. If the current children have all of the same names as the `Plan` list, then nothing is done. Otherwise, any missing items are inserted, and any extra ones are removed, and everything is put in the correct order according to the `Plan`.

The `Init` function(s) in the `PlanItem` are only run _once_ when a new widget element is made, so they should contain all of the initialization steps such as adding [[styles]] `Styler` functions and [[events]] handling functions.

There are functions in the `tree` package that use generics to make it easy to add plan items. The type of child widget to create is determined by the type in the Init function, for example this code:

```go
tree.AddAt(p, strconv.Itoa(i), func(w *core.Button) {
```

specifies that a `*core.Button` will be created.

* [[doc:tree.AddAt]] adds a new item to a `Plan` with a specified name (which must be unique!), and an `Init` function.
* [[doc:tree.Add]] does `AddAt` with an automatically-generated unique name based on the location in the code where the function is called. This only works for one-off calls, not in a for-loop where multiple elements are made at the same code line, like the above example.

## Maker function

A [[doc:tree.NodeBase.Maker]] function adds items to the Plan for a widget.

By having multiple such functions, each function can handle a different situation, when there is more complex logic that determines what child elements are needed, and how they should be configured.

Functions are called in the order they are added, and there are three [[doc:base/tiered.Tiered]] levels of these function lists, to allow more control over the ordering.

* [[doc:tree.AddChild]] adds a Maker function that calls [[doc:tree.Add]] to add a new child element. This is helpful when adding a few children in the main `Init` function of a given widget, saving the need to nest everything within an explicit `Maker` function.
* [[doc:tree.AddChildAt]] is the `AddAt` version that takes a unique name instead of auto-generating it.

## Styling sub-elements of widgets

The [[doc:tree.AddChildInit]] function can be used to modify the styling of elements within another widget. For example, many standard widgets have an optional [[icon]] element (e.g., [[button]], [[chooser]]). If you want to change the size of that icon, you can do something like this:

```Go
tree.AddChildInit(core.NewButton(b).SetIcon(icons.Download), "icon", func(w *core.Icon) {
    w.Styler(func(s *styles.Style) {
        s.Min.Set(units.Em(2))
    })
})
```

`AddChildInit` adds a new `Init` function to an existing `PlanItem`'s list of such functions. You just need to know the name of the children, which can be found using the [[inspector]] (in general they are lower kebab-case names based on the corresponding `Set` function, e.g., `SetIcon` -> `icon`, etc)

