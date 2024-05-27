package core

import (
	"strconv"
	"testing"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

const (
	nButtons = 100
	nUpdates = 100
)

func BenchmarkBuildDeclarative(b *testing.B) {
	for range b.N {
		for range nUpdates {
			b := NewBody()
			for i := range nButtons {
				NewButton(b).SetText(strconv.Itoa(i)).SetIcon(icons.Download).Style(func(s *styles.Style) {
					s.Min.Set(units.Em(5))
					s.CenterAll()
				})
			}
			for i := range nButtons {
				bt := b.Child(i)
				bt.CopyFieldsFrom(bt)
			}
		}
	}
}

func BenchmarkBuildPlan(bn *testing.B) {
	b := NewBody()
	for range bn.N {
		for range nUpdates {
			p := &Plan{}
			for i := range nButtons {
				AddAt(p, strconv.Itoa(i), func(w *Button) {
					w.SetText(strconv.Itoa(i)).SetIcon(icons.Download).Style(func(s *styles.Style) {
						s.Min.Set(units.Em(5))
						s.CenterAll()
					})
				})
			}
			p.Build(b)
		}
	}
}
