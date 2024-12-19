+++
Categories = ["Widgets"]
+++

A **text field** is a [[widget]] that a user can enter text in. Text fields come with powerful selection, navigation, and editing functionality built in, including undo and redo, copy and paste, and word-based navigation, selection, and deletion.

Text fields should mainly be used for relatively short and simple text. For more advanced use cases such as code editing, use a [[text editor]]. For numeric input, use a [[spinner]].

## Properties

You can make a text field without any custom options:

```Go
core.NewTextField(b)
```

You can set the placeholder of a text field and add label [[text]] before it:

```Go
core.NewText(b).SetText("Name:")
core.NewTextField(b).SetPlaceholder("Jane Doe")
```

You can set the text of a text field:

```Go
core.NewTextField(b).SetText("Hello, world!")
```

Text field content can overflow onto multiple lines:

```Go
core.NewTextField(b).SetText("This is a long sentence that demonstrates how text field content can overflow onto multiple lines")
```

You can make a text field outlined instead of filled:

```Go
core.NewTextField(b).SetType(core.TextFieldOutlined)
```

You can make a text field designed for password input:

```Go
core.NewTextField(b).SetTypePassword()
```

You can add a clear [[button]] to a text field:

```Go
core.NewTextField(b).AddClearButton()
```

You can set any custom leading and trailing [[icon]]s you want:

```Go
core.NewTextField(b).SetLeadingIcon(icons.Euro).SetTrailingIcon(icons.OpenInNew, func(e events.Event) {
    core.MessageSnackbar(b, "Opening shopping cart")
})
```

## Events

You can add a validation function that ensures the value of a text field is valid when a user [[event#change|changes]] it:

```Go
tf := core.NewTextField(b)
tf.SetValidator(func() error {
    if !strings.Contains(tf.Text(), "Go") {
        return errors.New("Must contain Go")
    }
    return nil
})
```

You can detect when the user [[event#change|changes]] the content of a text field and then exits it:

```Go
tf := core.NewTextField(b)
tf.OnChange(func(e events.Event) {
    core.MessageSnackbar(b, "OnChange: "+tf.Text())
})
```

You can detect when the user makes any changes to the content of a text field as they type ([[event#input|input]]):

```Go
tf := core.NewTextField(b)
tf.OnInput(func(e events.Event) {
    core.MessageSnackbar(b, "OnInput: "+tf.Text())
})
```
