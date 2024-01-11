# xe

xe provides an easy way to execute commands, handling errors and parsing full command strings.

```Go
    err := xe.Run("git", xe.Args("commit -am"))
```

This version allows a full command string to be used:

```Go
    err := xe.RunSh("git commit -am")
```


