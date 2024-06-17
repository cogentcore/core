# Func buttons

Cogent Core provides interactive func buttons, which are [buttons](../basic/buttons) bound to functions. The text, icon, and tooltip of a func button are automatically set based on the bound function, and when you click on a func button, it calls the function bound to it.

You can make a func button with any function:

```Go
core.NewFuncButton(parent).SetFunc(func() {
    core.MessageSnackbar(parent, "Function called")
})
```
