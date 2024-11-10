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