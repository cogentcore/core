# App creation and structure

The first call in every Cogent Core app is [[gi.NewBody]]. This creates and returns a new [[gi.Body]], which is a container in which app content is placed.

After calling [[gi.NewBody]], you add content to the [[gi.Body]] that was returned, which is typically given the local variable name `b` for body.

Then, after adding content to your body, you can create and start a window from it using [[gi.Body.RunMainWindow]].

Therefore, the standard structure of a Cogent Core app looks like this:

```go
package main

import "cogentcore.org/core/gi"

func main() {
	b := gi.NewBody("App Name")
	// Add app content here
	b.RunMainWindow()
}
```
