# Values

Cogent Core provides the giv value system, which allows you to instantly turn Go values into interactive widgets bound to the original values with just a single simple line of code.

You can create a [[giv.Value]] from any value:

```Go
giv.NewValue(parent, colors.Orange)
```

You can detect when the user changes a value:

```Go
t := time.Now()
giv.NewValue(parent, &t).OnChange(func(e events.Event) {
    gi.MessageSnackbar(parent, "The time is "+t.Format(time.DateTime))
})
```

You can customize a value using tags:

```Go
giv.NewValue(parent, 70, `view:"slider"`)
```

Cogent Core provides interactive widget values for many types, including most elementary types, like strings, integers, and floats; composite types, like maps, slices, and structs; and widely used other types, like colors, times, and durations. More values are documented in the documentation pages for certain views, like map and slice views.
