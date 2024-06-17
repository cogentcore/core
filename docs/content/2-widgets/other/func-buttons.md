# Func buttons

Cogent Core provides interactive func buttons, which are [buttons](../basic/buttons) bound to functions. The text, icon, and tooltip of a func button are automatically set based on the bound function, and when you click on a func button, it calls the function bound to it.

You can make a func button with any function:

```Go
core.NewFuncButton(parent).SetFunc(func() {
    core.MessageSnackbar(parent, "Function called")
})
```

Notice how the text of the func button above is set to `Main init`. That is because the bound function in that case is an anonymous function defined in `main.init`. You can also see that the tooltip of the function has been set to give more information about the function.

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
