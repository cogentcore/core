# Apps

The first call in every Cogent Core app is [[core.NewBody]]. This creates and returns a new [[core.Body]], which is a container in which app content is placed.

After calling [[core.NewBody]], you add content to the [[core.Body]] that was returned, which is typically given the local variable name `b` for body.

Then, after adding content to your body, you can create and start a window from it using [[core.Body.RunMainWindow]].

Therefore, the standard structure of a Cogent Core app looks like this:

```go
package main

import "cogentcore.org/core/gi"

func main() {
	b := core.NewBody("App Name")
	// Add app content here
	b.RunMainWindow()
}
```
