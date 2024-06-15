# Todo list

This tutorial shows how to make a todo list app using Cogent Core. You should read the [basics](../basics) section if you haven't yet before starting this.

We will represent todo list items using an `item` struct type:

```go
type item struct {
    Done bool
    Task string
}
```
