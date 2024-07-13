# Func buttons

Cogent Core provides interactive func buttons, which are [buttons](../basic/buttons) bound to functions. The text and tooltip of a func button are automatically set based on the bound function, and when you click on a func button, it calls the function bound to it.

You can make a func button with any function:

```Go
core.NewFuncButton(parent).SetFunc(func() {
    core.MessageSnackbar(parent, "Function called")
})
```

Notice how the text of the func button above is set to `Anonymous function`. That is because the bound function in that case is an anonymous function. You can also see that the tooltip of the function has been set to give more information about the function.

You can always override the text and tooltip of a func button as long as you do so after you call [[core.FuncButton.SetFunc]]:

```Go
core.NewFuncButton(parent).SetFunc(func() {
    core.MessageSnackbar(parent, "Function called")
}).SetText("Run").SetTooltip("Click me!")
```

When the bound function takes arguments, the user will be prompted for those arguments in a dialog:

```Go
core.NewFuncButton(parent).SetFunc(func(name string, age int) {
    core.MessageSnackbar(parent, name+" is "+strconv.Itoa(age)+" years old")
})
```

When the bound function returns values, you can set [[core.FuncButton.ShowReturn]] to true for the user to be shown those values:

```Go
core.NewFuncButton(parent).SetShowReturn(true).SetFunc(func() (string, int) {
    return "Gopher", 35
})
```

You can prompt the user for confirmation before calling the function:

```Go
core.NewFuncButton(parent).SetConfirm(true).SetFunc(func() {
    core.MessageSnackbar(parent, "Function called")
})
```

You may have noticed in all of the examples so far that the names and tooltips for the func buttons are not particularly useful, and the names of the arguments are missing. To solve this, you can use named functions added to [[types]], which gives information about all of those things. For example, here is a func button for [[core.SettingsWindow]]:

```Go
core.NewFuncButton(parent).SetFunc(core.SettingsWindow).SetConfirm(true)
```

The process for adding a function to [[types]] is similar to the process for adding a struct described in [forms](../collections/forms):

```go
// Add this once per package:
//go:generate core generate

// Add types:add for every function you want the documentation for:

// This comment will be displayed as the tooltip for a func button.
func DoSomething() { //types:add

}
```
