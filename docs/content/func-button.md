+++
Categories = ["Widgets"]
+++

A **func button** is a [[button]] that can be [[bind|bound]] to a function. The [[text]] and [[tooltip]] of a func button are automatically set based on the bound function, and when you [[events#click]] on a func button, it calls the function bound to it.

## Properties

You can make a func button with any function:

```Go
core.NewFuncButton(b).SetFunc(func() {
    core.MessageSnackbar(b, "Function called")
})
```

Notice how the text of the func button above is set to `Anonymous function`. That is because the bound function in that case is an anonymous function. You will see [[#generate|later on this page]] how to get more meaningful information using named functions.

You can always override the text and tooltip of a func button as long as you do so after you call [[doc:core.FuncButton.SetFunc]]:

```Go
fb := core.NewFuncButton(b).SetFunc(func() {
    core.MessageSnackbar(b, "Function called")
})
fb.SetText("Run").SetTooltip("Click me!")
```

When the bound function takes arguments, the user will be prompted for those arguments in a [[dialog]]:

```Go
core.NewFuncButton(b).SetFunc(func(name string, age int) {
    core.MessageSnackbar(b, name+" is "+strconv.Itoa(age)+" years old")
})
```

When the bound function returns values, you can set [[doc:core.FuncButton.ShowReturn]] to true for the user to be shown those values:

```Go
core.NewFuncButton(b).SetShowReturn(true).SetFunc(func() (string, int) {
    return "Gopher", 35
})
```

You can prompt the user for confirmation before calling the function:

```Go
core.NewFuncButton(b).SetConfirm(true).SetFunc(func() {
    core.MessageSnackbar(b, "Function called")
})
```

## Generate

You may have noticed in all of the examples so far that the names and tooltips for the func buttons are not particularly useful, and the names of the arguments are missing. To solve this, you can use named functions added to [[generate#types]], which gives information about all of those things. For example, here is a func button for [[doc:core.SettingsWindow]]:

```Go
core.NewFuncButton(b).SetFunc(core.SettingsWindow).SetConfirm(true)
```

The process for adding a function to [[generate#types]] uses [[generate]] as shown below:

```go
// Add this once per package:
//go:generate core generate

// Add types:add for every function you want the documentation for:

// This comment will be displayed as the tooltip for a func button.
func DoSomething() { //types:add

}
```
