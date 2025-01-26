+++
Categories = ["Tutorials"]
+++

This [[tutorials|tutorial]] makes a simple **hello world** example [[app]]:

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

## Extending hello world

We can expand our hello world example app by displaying a message when a user clicks on the button:

```Go
core.NewButton(b).SetText("Hello, World!").OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "Hello!")
})
```

We can customize the message based on the user's name:

```Go
tf := core.NewTextField(b).SetPlaceholder("Name")
core.NewButton(b).SetText("Hello, World!").OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "Hello, "+tf.Text()+"!")
})
```
