## Updating order

The `Update()` method on [[doc:core.WidgetBase]] is the general-purpose update method, which should be called whenever you have finished setting field values on a widget (or any other such configuration changes), and want to have the GUI display reflect those changes.  In general, setting field values or other methods that configure a widget _should have no side-effects directly_, and all the implications thereof should be handled in the context of the `Update()` method.

It performs these steps in order:

* calls `UpdateWidget()` then `Style()` on itself and each of its children down the full depth of the tree under it.

* calls `NeedsLayout()` to trigger a new layout of the widgets.

The `UpdateWidget()` method first updates the value represented by the widget if any `Bind` value has been set, and then runs the `Updaters` functions, including any `Maker` functions as described in [basics/plans](../basics/plans), which makes a [[doc:tree.Plan]] to update the child elements within the Widget.

Some other important things to keep in mind:

* `Update()` is automatically called on all widgets when the [[doc:core.Scene]] is first shown in the GUI, so it doesn't need to be called manually prior to that.

* `Update()` can be called prior to GUI rendering, and in general _must_ be called prior to accessing any children of a given Widget, as they will not in general exist prior to the Update call.

* `Restyle()` can be called to specifically call `Style()` and `NeedsRender()` on a widget (and its children), when its `State` or other style-relevant data has changed.  This is much faster and more efficient than a full `Update()` call.

## How to make a new Widget type

To better understand the Update logic, it is useful to see how the code is organized to configure a given Widget.  We will look at elements of the [[doc:core.Button]] widget as a familiar example.

### The `Init()` method has everything

In general, all of the configuration code is specified in the `Init()` method, which specifies [styling](styling), [event](events) handling, and make / update functions, typically in that order:

```go
func (bt *Button) Init() {
    bt.Frame.Init() // MUST call parent type's Init!
    bt.Styler(func(s *styles.Style) {
        ...
    })
    bt.OnClick(func(e events.Event) {
        ...
    })
    bt.Maker(func(p *tree.Plan) {
        ...
    })
}    
```

### Styler

The `Styler` function should start with a `SetAbilities` call that specifies what kinds of things this Widget can do, and then it sets style properties based on the current `State` flags and other field values etc.  See [styling](styling) for more information.

### Maker of Plans

The primary maker function for a Widget can start with any general-purpose code needed to coordinate relevant state prior to adding child widgets, which will be run every time the `Update()` function is called.

```go
    bt.Maker(func(p *tree.Plan) {
        if bt.HasMenu() {
            if bt.Type == ButtonMenu {
                if bt.Indicator == "" {
                    bt.Indicator = icons.KeyboardArrowRight
                } ...
```

Next, children are added with appropriate logic determining whether they are needed or not:

```go
        ...
        if bt.Icon.IsSet() {
            tree.AddAt(p, "icon", func(w *Icon) {
                w.Styler(func(s *styles.Style) {
                    s.Font.Size.Dp(18)
                })
                w.Updater(func() {
                    w.SetIcon(bt.Icon)
                })
            })
            if bt.Text != "" {
                tree.AddAt(p, "space", func(w *Space) {})
            }
        }
        ...
```

The [[doc:tree.AddAt]] adds a child with the given name to the overall [[doc:tree.Plan]] for the widget's children, and the first closure function indicates what to do when that icon is actually made for the first time.  This code is run _after_ the standard `Init()` method for the [[doc:core.Icon]] type itself, so it is providing any _additional_ styling and functionality over and above the defaults for that type.  You can connect event handlers here, etc.  Critically, this code is only ever run _once_, and like the `Init()` method itself, it should largely be setting closures that will actually be run _later_ when the widget is actually made.

The `Updater` closure added here will be called _every time Update() is called_ on the Icon, and it ensures that this icon is always updated to reflect the `Icon` field _on the parent Button_ object.  This is how you establish connections between properties on different widgets to ensure everything is consistent: the button's Icon field is the definitive source setting for what the icon should be.

In general, it is ideal to be able to specify _all_ of the dynamic updating logic for the children within the Add and Updater functions.  Indeed, this ability to put all the logic in one place is a major advantage of this system.  Nevertheless, sometimes you also need to provide additional methods that access the `ChildByName` using its unique name provided to `AddAt`, and update its properties.

Note that you should _never_ call functions like `Styler`, `Maker`, `OnClick` etc in a situation where they might be called multiple times, because that would end up adding _multiple copies_ of the given closure functions to the list of such functions to be run at the appropriate time, which, aside from being inefficient, could lead to bad effects.

The [[doc:tree.Add]] function works just like `AddAt` except it automatically generates a unique name, based on the point in the source code where it is called.  This is convenient for [[doc:core.Toolbar]] Makers, where you are often adding multiple buttons and don't really care about the names because you will not be referring back to these children elsewhere.

### Adding more Init functions to children

In a Widget that embeds another widget type and extends its functionality, you can add additional `Init` closures that override or extend the basic initialization function that was specified in the `AddAt` call that creates the child widget in the first place.  This is done using the [[doc:tree.AddInit]] function, e.g., in the case of a [[doc:core.Spinner]] that embeds [[doc:core.TextField]] and modifies the properties of the leading and trailing icons thereof:

```go
	sp.Maker(func(p *tree.Plan) {
		if sp.IsReadOnly() {
			return
		}
		tree.AddInit(p, "lead-icon", func(w *Button) {...})
		tree.AddInit(p, "trail-icon", func(w *Button) {...})
	})
```

### Adding individual children and Makers for sub-children

If a widget does not require dynamic configuration of its children (i.e., it always has the same children), you can save a step of indentation by using the [[doc:tree.AddChildAt]] version of [[doc:tree.AddAt]], without enclosing everything in an outer `Maker` function.  These `Child` versions of all the basic `tree.Add*` functions simply create a separate `Maker` function wrapper for you.  There is a list of Maker functions that are called to create the overall [[doc:tree.Plan]] for the widget, so multiple such functions can be defined, and they are called in the order added.

These `Child` versions are essential when you need to specify the children of children (and beyond), to configure an entire complex structure.  Here's an example from the [Cogent Mail](https://github.com/cogentcore/cogent/mail) app:

```go
	tree.AddChildAt(a, "splits", func(w *core.Splits) {
		tree.AddChildAt(w, "mbox", func(w *core.Tree) {
			w.SetText("Mailboxes")
		})
		tree.AddChildAt(w, "list", func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
		})
		tree.AddChildAt(w, "mail", func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
			tree.AddChildAt(w, "msv", func(w *core.Form) {
				w.SetReadOnly(true)
			})
			tree.AddChildAt(w, "mb", func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
				})
			})
		})
		w.SetSplits(0.1, 0.2, 0.7)
	})
```

The resulting code nicely captures the tree structure of the overall GUI, and this can be useful even when there aren't any dynamic elements to it.



