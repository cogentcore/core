# Hello World

This page teaches you how to make a simple Hello World example app with Cogent Core.

1. Run `mkdir hello && cd hello && go mod init hello && touch main.go` to make a new Go project
2. Open `main.go` using an editor of your choice
3. Add the following code to your editor:

```Go
package main

import "cogentcore.org/core/core"

func main() {
	b := core.NewBody("Hello")
	core.NewButton(b).SetText("Hello, World!")
	b.RunMainWindow()
}
```

4. Update your dependencies by running `go mod tidy`
5. Build and run the code by running `core run`. This should create a window with a button that says "Hello, World!", similar to the one below:

![Hello World App](hello-world.png)
