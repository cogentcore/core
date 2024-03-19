# Text fields

Cogent Core provides highly customizable text fields with powerful selection, navigation, and editing functionality built in, including undo and redo, copy and paste, and word-based navigation, selection, and deletion.

You can make a text field without any custom options:

```Go
gi.NewTextField(parent)
```

You can set the placeholder of a text field and add label text before it:

```Go
gi.NewLabel(parent).SetText("Name:")
gi.NewTextField(parent).SetPlaceholder("Jane Doe")
```

You can set the starting text of a text field:

```Go
gi.NewTextField(parent).SetText("Hello, world!")
```

Text field content can overflow onto multiple lines:

```Go
gi.NewTextField(parent).SetText("This is a really long sentence that demonstrates how text field content can overflow onto multiple lines")
```

You can make a text field outlined instead of filled:

```Go
gi.NewTextField(parent).SetType(gi.TextFieldOutlined)
```

You can make a text field designed for password input:

```Go
gi.NewTextField(parent).SetTypePassword()
```

You can add a clear button to a text field:

```Go
gi.NewTextField(parent).AddClearButton()
```

You can set any custom leading and trailing icons you want:

```Go
gi.NewTextField(parent).SetLeadingIcon(icons.Euro).SetTrailingIcon(icons.OpenInNew, func(e events.Event) {
    gi.MessageSnackbar(parent, "Opening shopping cart")
})
```

You can add a validation function that ensures the value of a text field is valid:

```Go
tf := gi.NewTextField(parent)
tf.SetValidator(func() error {
    if !strings.Contains(tf.Text(), "Go") {
        return errors.New("Must contain Go")
    }
    return nil
})
```
