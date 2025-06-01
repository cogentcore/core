+++
Categories = ["Resources"]
+++

The [[doc:cursors]] package contains standard cursors that can be set to indicate the kinds of actions available on a widget. The cursor changes when the mouse enters the space of the widget.

Set the cursor in a styler:
```Go
core.NewText(b).SetText("Mouse over to see the cursor").Styler(func(s *styles.Style) {
    s.Cursor = cursors.Help
})
```

This code shows all the standard cursors available:

```Go
curs:= core.NewFrame(b)
curs.Styler(func(s *styles.Style) {
    s.Display = styles.Grid
    s.Columns = 4
	s.Gap.Set(units.Dp(18))
})
curs.Maker(func(p *tree.Plan) {
	for _, c := range cursors.CursorN.Values() {
		nm := strcase.ToCamel(c.String())
		fnm := filepath.Join("svg", c.String() + ".svg")
		tree.AddAt(p, nm, func(w *core.Frame) {
            w.Styler(func(s *styles.Style) {
                s.Direction = styles.Column
				s.Cursor = c.(cursors.Cursor)
            })
			sv := core.NewSVG(w)
			sv.OpenFS(cursors.Cursors, fnm)
			sv.Styler(func(s *styles.Style) {
				s.Min.Set(units.Em(4))
				s.Cursor = c.(cursors.Cursor)
			})
            core.NewText(w).SetText(nm).Styler(func(s *styles.Style) {
                s.SetTextWrap(false)
				s.CenterAll()
			})
		})
	}
})
```

