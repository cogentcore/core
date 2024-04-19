# Text fields

Cogent Core provides highly customizable text fields with powerful selection, navigation, and editing functionality built in, including undo and redo, copy and paste, and word-based navigation, selection, and deletion.

You can make a text field without any custom options:

```Go
core.NewTextField(parent)
```

You can set the placeholder of a text field and add label text before it:

```Go
core.NewLabel(parent).SetText("Name:")
core.NewTextField(parent).SetPlaceholder("Jane Doe")
```

You can set the text of a text field:

```Go
core.NewTextField(parent).SetText("Hello, world!")
```

Text field content can overflow onto multiple lines:

```Go
core.NewTextField(parent).SetText("This is a long sentence that demonstrates how text field content can overflow onto multiple lines")
```

You can make a text field outlined instead of filled:

```Go
core.NewTextField(parent).SetType(core.TextFieldOutlined)
```

You can make a text field designed for password input:

```Go
core.NewTextField(parent).SetTypePassword()
```

You can add a clear button to a text field:

```Go
core.NewTextField(parent).AddClearButton()
```

You can set any custom leading and trailing icons you want:

```Go
core.NewTextField(parent).SetLeadingIcon(icons.Euro).SetTrailingIcon(icons.OpenInNew, func(e events.Event) {
    core.MessageSnackbar(parent, "Opening shopping cart")
})
```

You can add a validation function that ensures the value of a text field is valid:

```Go
tf := core.NewTextField(parent)
tf.SetValidator(func() error {
    if !strings.Contains(tf.Text(), "Go") {
        return errors.New("Must contain Go")
    }
    return nil
})
```

You can detect when the user changes the content of the text field and then exits it:

```Go
tf := core.NewTextField(parent)
tf.OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, "OnChange: "+tf.Text())
})
```

You can detect when the user makes any change to the content of the text field as they type:

```Go
tf := core.NewTextField(parent)
tf.OnInput(func(e events.Event) {
    core.MessageSnackbar(parent, "OnInput: "+tf.Text())
})
```
