+++
Categories = ["Resources"]
+++

The **[[doc:cursors]]** package contains standard cursors that can be set to indicate the kinds of actions available on a [[widget]]. The cursor changes when the mouse enters the space of a widget.

You can set a cursor in [[style]]r:

```Go
tx := core.NewText(b).SetText("Hover to see the cursor")
tx.Styler(func(s *styles.Style) {
    s.Cursor = cursors.Help
})
```

## All cursors

These are all of the standard cursors available:

{collapsed="true"}
```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
	s.Wrap = true
	s.Grow.Set(1, 0)
})
fr.Maker(func(p *tree.Plan) {
	for _, c := range cursors.CursorValues() {
		name := c.String()
		fname := "svg/" + name + ".svg"
		tree.AddAt(p, name, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Cursor = c
				s.Direction = styles.Column
				s.Align.Items = styles.Center
			})
			sv := core.NewSVG(w)
			sv.OpenFS(cursors.Cursors, fname)
			sv.Styler(func(s *styles.Style) {
				s.Min.Set(units.Em(4))
			})
			tx := core.NewText(w).SetText(strcase.ToCamel(name))
			tx.Styler(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
		})
	}
})
```
