The **basics** are a simple overview of the key [[concepts]] of Cogent Core. You can interactively run and edit the examples on this website directly, or you can [[install]] Cogent Core on your system and experiment locally. You can also use the [[playground]] to develop interactively. After you finish the basics, we recommend you read the [[tutorials]] and explore the [[widgets]].

## Hello world

This code makes a simple **hello world** example app:

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

## Apps

*Main page: [[App]]*

The first call in every **app** is [[doc:core.NewBody]]. This creates and returns a new [[doc:core.Body]], which is a container in which app content is placed. This takes an optional name, which is used for the title of the app/window/tab.

After calling NewBody, you add content to the body that was returned, which is typically given the local variable name `b` for body.

Then, after adding content to your body, you can create and start a window from it using [[doc:core.Body.RunMainWindow]].

Therefore, the standard structure of an app looks like this:

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

*Main page: [[Widget]]*

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

*Main page: [[Event]]*

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

## Styling

*Main page: [[Styling]]*

**Styling** allows you to easily customize the appearance of [[#widgets|widgets]] at any level.

You can change any style properties of a widget:

```Go
core.NewText(b).SetText("Bold text").Styler(func(s *styles.Style) {
    s.Font.Weight = styles.WeightBold
})
```

You can change the [[color]]s of a widget using Cogent Core's dynamic [[color#scheme|color scheme]] system:

```Go
core.NewButton(b).SetText("Success button").Styler(func(s *styles.Style) {
    s.Background = colors.Scheme.Success.Base
    s.Color = colors.Scheme.Success.On
})
```

You can change the size of a widget using Cogent Core's flexible [[unit]] system:

```Go
core.NewFrame(b).Styler(func(s *styles.Style) {
    s.Min.Set(units.Dp(50))
    s.Background = colors.Scheme.Primary.Base
})
```

## Updating

*Main page: [[Updating]]*

There are several ways to dynamically **update** the content of an app.

The simplest way to update a [[#widgets|widget]] is to call [[doc:core.WidgetBase.Update]] after changing any of its properties:

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

## Value binding

*Main page: [[Value binding]]*

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

## Plans

*Main page: [[Plan]]*

The previous two sections cover how to update the properties of a [[#widgets|widget]], but what if you want to update the structure of a widget? To answer that question, Cogent Core provides **plans**, a mechanism for specifying what the children of a widget should be, which is then used to automatically update the actual children to reflect that.

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

## Async

*Main page: [[Async]]*

Most of the time, updating happens synchronously through [[#events|event]] handlers, [[#styling|stylers]], [[#updating|updaters]], and [[#plans|makers]]. However, sometimes you need to update content **asynchronously** from another goroutine. When you do so, you just need to protect any updates you make to [[#widgets|widgets]] with [[doc:core.WidgetBase.AsyncLock]] and [[doc:core.WidgetBase.AsyncUnlock]].

For example, this code utilizes a goroutine to update [[text]] to the current time every second:

```Go
text := core.NewText(b)
text.Updater(func() {
    text.SetText(time.Now().Format("15:04:05"))
})
go func() {
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        text.AsyncLock()
        text.Update()
        text.AsyncUnlock()
    }
}()
```

## Next steps

Now that you understand the basics, you can apply them in [[tutorials]], explore the [[widgets]], or [[install]] Cogent Core on your system.
