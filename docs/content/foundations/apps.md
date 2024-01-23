# App creation and structure

The first call in every Cogent Core app is [[gi.NewAppBody]]. This creates a new [[gi.App]] object, which holds information about the app like its name, icon, and description. It also creates and returns a new [[gi.Body]], which is a container in which app content is placed.

After calling [[gi.NewAppBody]], you add content to the [[gi.Body]] that was returned, which is typically given the local variable name `b` for body.

Then, after adding content to your body, you can create a window from it using [[gi.Body.NewWindow]]. This returns a [[gi.Stage]] object that you can run with [[gi.Stage.Run]]. Then, to run the main loop that prevents your app from quitting until all windows are closed, you can run [[gi.Stage.Wait]], which must be called once and only once for every app.

Therefore, the standard structure of a Cogent Core app looks like this:

```go
package main

import "cogentcore.org/core/gi"

func main() {
	b := gi.NewAppBody("App Name")
	// add app content
	b.NewWindow().Run().Wait()
}
```