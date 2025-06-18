+++
Categories = ["Concepts"]
+++

[[Update]] and [[bind]] cover how to update the properties of a [[widget]], but what if you want to update the structure of a widget? To answer that question, Cogent Core provides **plans**, a mechanism for specifying what the children of a widget should be, which is then used to automatically update the actual children to reflect that.

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

Plans are a powerful tool that are critical for some widgets such as those that need to dynamically manage hundreds of children in a convenient and performant way. They aren't always necessary, but you will find them being used a lot in complicated apps, and you will see more examples of them in the rest of this documentation. Read more of this page for details.

## Naming

Each item in a plan must have a unique name, which is used for updating the children in an efficient way. If the current children have all of the same names as the plan, then nothing is done. Otherwise, any missing items are inserted, and any extra ones are removed, and everything is put in the correct order according to the plan.

There are two ways to accomplish unique naming. The first is [[doc:tree.AddAt]], where you pass a unique name to the function. That is useful for cases like the example above where you have some unique index to use as the name.

However, if you aren't doing a for loop, the easier approach is [[doc:tree.Add]]. Add calls AddAt using an automatically generated unique name based on the location in the code where the function is called. This doesn't work in loops since multiple items are added at the same line of code. Here is an example using Add:

```Go
on := true
sw := core.Bind(&on, core.NewSwitch(b))
buttons := core.NewFrame(b)
buttons.Maker(func(p *tree.Plan) {
    tree.Add(p, func(w *core.Button) {
        w.SetText("First")
    })
    if on {
        tree.Add(p, func(w *core.Button) {
            w.SetText("Extra")
        })
    }
})
sw.OnChange(func(e events.Event) {
    buttons.Update()
})
```

## Maker

The examples above have used [[doc:tree.NodeBase.Maker]], which adds a maker function that can use logic like if statements and for loops to determine what elements should be added. However, sometimes you know that certain elements will always be added, in which case you can use a helper function to avoid unnecessary complexity and code nesting.

[[doc:tree.AddChild]] and [[doc:tree.AddChildAt]] are the same as Add and AddAt respectively, except that they add the maker function for you. For example, here is the same example as above, but with the first button taken out of the maker using AddChild:

```Go
on := true
sw := core.Bind(&on, core.NewSwitch(b))
buttons := core.NewFrame(b)
tree.AddChild(buttons, func(w *core.Button) {
    w.SetText("First")
})
buttons.Maker(func(p *tree.Plan) {
    if on {
        tree.Add(p, func(w *core.Button) {
            w.SetText("Extra")
        })
    }
})
sw.OnChange(func(e events.Event) {
    buttons.Update()
})
```

When there are multiple maker functions, they are called in the order they are added (FIFO). There are also functions like [[doc:tree.NodeBase.FirstMaker]] and [[doc:tree.NodeBase.FinalMaker]] to allow more control over the ordering when necessary.

## Init functions

The anonymous function that you pass to [[doc:tree.Add]] etc is the init function, responsible for customizing the child widget. This function is only run one time, when that widget is made, and it contains all of the initialization steps such as adding [[styler]]s and [[event]] handlers. Because it is only run once, the init function needs to add [[update]]rs for any properties that may change over time:

```Go
number := 3
spinner := core.Bind(&number, core.NewSpinner(b))
fr := core.NewFrame(b)
tree.AddChild(fr, func(w *core.Text) {
    w.Updater(func() {
        w.SetText(strconv.Itoa(number))
    })
})
spinner.OnChange(func(e events.Event) {
    fr.Update()
})
```

This is an important point worth repeating: the init function is only run *once*. It is a closure and looks like other closures that are run more than once ([[update]]rs, [[style]]rs, [[event]] handlers etc), but it is only run once, and all dynamic logic must be placed in an updater, styler, or event handler. Many common pitfalls derive from this.

### Generics

Functions like [[doc:tree.Add]] use generics to make it easy to add plan items. The type you specify for the `w` argument in the function is used to determine the type of child widget to create (using generics type parameter inference). In rare cases where the precise type of the widget is not known at compile time, see [[doc:tree.AddNew]] and [[doc:tree.Plan.Add]].

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

