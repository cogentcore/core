The **basics** are a simple overview of the key [[concepts]] of Cogent Core. We recommend you read the basics before the [[tutorials]] and [[install|install instructions]].

## Hello world

This code makes a simple hello world example app using Cogent Core:

```Go
package main

import "cogentcore.org/core/core"

func main() {
    b := core.NewBody()
    core.NewButton(b).SetText("Hello, World!")
    b.RunMainWindow()
}
```

Notice how you can see the result of the code above, a [[button]] with the [[text]] "Hello, World!". Not only can you see the result of the code, you can edit the code live. Try changing "Hello, World!" to "Click me!" and you will see the button update accordingly.

Even though Cogent Core is written in Go, a compiled language, it uses the interpreter [yaegi](https://github.com/cogentcore/yaegi) to provide interactive editing. You can edit almost all of the examples on this website and see the result immediately. You can also use the [[playground]] to experiment interactively with Cogent Core.

## Apps

*Main article: [[Apps]]*

The first call in every **app** is [[doc:core.NewBody]]. This creates and returns a new [[doc:core.Body]], which is a container in which app content is placed. This takes an optional name, which is used for the title of the app/window/tab.

After calling NewBody, you add content to the body that was returned, which is typically given the local variable name `b` for body.

Then, after adding content to your body, you can create and start a window from it using [[doc:core.Body.RunMainWindow]].

Therefore, the standard structure of a Cogent Core app looks like this:

```Go
package main

import "cogentcore.org/core/core"

func main() {
	b := core.NewBody("App Name")
	// Add app content here
	b.RunMainWindow()
}
```

For most of the code examples on this website, we will omit the outer structure of the app so that you can focus on the app content.

## Widgets

*Main article: [[Widgets]]*

All app content is organized into **widgets**, which are reusable app components that render, store information, and handle [[#events|events]]. All widgets satisfy the [[doc:core.Widget]] interface.

Widgets are typically created by calling the `core.New{WidgetName}` function (for example: [[doc:core.NewButton]]). All of these `New` functions take a parent in which the widget is added. This allows you to create nested widget structures and [[layout]]s that position and size widgets in different ways. For elements at the root level of your app, the parent is `b`, the app body. However, if your widget is located in a some other container, you would pass that as the parent.

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

## Events

*Main article: [[Events]]*

**Events** are user actions that you can process. To handle an event, simply call the `On{EventType}` method on any [[#widgets|widget]]. For example:

```Go
core.NewButton(b).SetText("Click me!").OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "Button clicked")
})
```

The [[doc:events.Event]] object passed to the function can be used for things such as obtaining detailed event information. For example, you can determine the exact position of a click event:

```Go
core.NewButton(b).SetText("Click me!").OnClick(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprint("Button clicked at ", e.Pos()))
})
```
