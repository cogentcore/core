# Widgets

All app content in Cogent Core is organized into widgets, which are reusable app components that render, store information, and handle events. All widgets satisfy the [[gi.Widget]] interface.

Widgets are typically created by calling the `gi.New{WidgetName}` function (for example: [[gi.NewButton]]). All of these `New` functions take a parent to which the widget is added. This allows you to create nested widget structures and layouts that position and size widgets in different ways. For elements at the root level of your app, the parent is `b`, the app body. However, if your widget is located in a some other container, you would pass that as the parent.

Many widgets define attributes that you can set, like the text of a button. These attributes can be set using the `Set{AttributeName}` method (for example: [[gi.Button.SetText]]). These `Set` methods always return the original object so that you can chain multiple `Set` calls together on one line. You can also always access the attributes of a widget directly by accessing its fields.

Here is an example of using `New` and `Set` functions to construct and configure a widget:

```Go
gi.NewButton(parent).SetText("Click me!").SetIcon(icons.Add)
```