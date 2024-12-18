+++
Categories = ["Tutorials"]
+++

This [[tutorials|tutorial]] shows how to make a **sign in** page that leads to a home page using [[pages]].

We will represent user information using a struct:

```Go
type user struct {
    Username string
    Password string
}
u := &user{}
```

We can create a sign in page to enter user information in a [[form]]:

```Go
type user struct {
    Username string
    Password string
}
u := &user{}
pg := core.NewPages(b)
pg.AddPage("sign-in", func(pg *core.Pages) {
    core.NewForm(pg).SetStruct(u)
    core.NewButton(pg).SetText("Sign in")
})
```

We can create a home page that opens after a user [[event#click|clicks]] the sign in [[button]]:

```Go
type user struct {
    Username string
    Password string
}
u := &user{}
pg := core.NewPages(b)
pg.AddPage("sign-in", func(pg *core.Pages) {
    core.NewForm(pg).SetStruct(u)
    core.NewButton(pg).SetText("Sign in").OnClick(func(e events.Event) {
        pg.Open("home")
    })
})
pg.AddPage("home", func(pg *core.Pages) {
    core.NewText(pg).SetText("Welcome, "+u.Username+"!").SetType(core.TextHeadlineSmall)
    core.NewButton(pg).SetText("Sign out").OnClick(func(e events.Event) {
        *u = user{}
        pg.Open("sign-in")
    })
})
```
