+++
Categories = ["Widgets"]
+++

A **body** is a [[frame]] that contains the main [[app]] content. It is contained within a [[scene]], which is the root node of a window or dialog.

The `b` widget in these docs examples represents the body that you typically add widgets to:

```Go
core.NewButton(b).SetText("Click me")
```

This is docs shorthand for:

```Go
package main

import "cogentcore.org/core/core"

func main() {
	b := core.NewBody("App Name")
	core.NewButton(b).SetText("Click me")
	b.RunMainWindow()
}
```

## Methods

Because the body is typically the easiest content container to access, there are various helper methods on it. For example, to create a [[stage]], you can use methods like [[doc:core.Body.RunMainWindow]] as above. See the [[stage]] docs for more info on these methods and how to customize the way a window or dialog is displayed.
