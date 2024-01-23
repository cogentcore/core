# Hello World

Create a simple Hello World example app with Goki.

1. Run `mkdir myapp && cd myapp && go mod init myapp && touch main.go` to make a new Go project
2. Open `main.go` using an editor of your choice
3. Add the following code to your editor:

```Go
package main

import "cogentcore.org/core/gi"

func main() {
	b := gi.NewAppBody("Hello")
	gi.NewLabel(b).SetText("Hello, World!")
	b.StartMainWindow()
}
```

4. Update your dependencies by running `go mod tidy`
5. Build and run the code by running `core run`. This should create a window with text that says "Hello, World!", similar to the one below:

![Hello World App](helloworld.png)
