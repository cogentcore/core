+++
Categories = ["Concepts"]
+++

All app content is organized into **widgets**, which are reusable app components that [[render]], store information, and handle [[event]]s. See [[widgets]] for a list of widget types.

All widgets satisfy the [[doc:core.Widget]] interface. Widgets are typically created by calling the `core.New{WidgetName}` function (for example: [[doc:core.NewButton]]). All of these `New` functions take a parent in which the widget is added. This allows you to create nested widget structures and [[layout]]s that position and size widgets in different ways. For elements at the root level of your app, the parent is `b`, the [[app]] body. However, if your widget is located in a some other container, you would pass that as the parent.

Many widgets define attributes that you can set, like the text of a [[button]]. These attributes can be set using the `Set{AttributeName}` method (for example: [[doc:core.Button.SetText]]). These `Set` methods always return the original object so that you can chain multiple `Set` calls together on one line. You can also always access the attributes of a widget by directly accessing its fields.

Here is an example of using `New` and `Set` functions to construct and configure a widget:

```Go
core.NewButton(b).SetText("Click me!").SetIcon(icons.Add)
```

You can always assign a widget to a variable and then get information from it or make further calls on it at any point. For example:

```Go
bt := core.NewButton(b).SetText("Click me!")
// Later...
bt.SetText("New text")
```
