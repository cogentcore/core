package main

// +build ignore

import (
	"fmt"
	"github.com/skelterjohn/geom"
	"github.com/skelterjohn/go.uik"
	"github.com/skelterjohn/go.uik/layouts"
	"github.com/skelterjohn/go.uik/widgets"
	"github.com/skelterjohn/go.wde"
	_ "github.com/skelterjohn/go.wde/glfw3"
	"image/color"
)

func main() {
	go uikplay()
	wde.Run()
}

func uikplay() {

	w, err := uik.NewWindow(nil, 480, 320)
	if err != nil {
		fmt.Println(err)
		return
	}
	w.W.SetTitle("go.uik")

	gcfg, err := layouts.ParseGridConfig(`
{
	"Components": {
		"radio": {
			"GridX": 0,
			"GridY": 0,
			"AnchorLeft": true,
			"AnchorTop": true
		},
		"label": {
			"GridX": 1,
			"GridY": 0,
			"AnchorRight": true,
			"AnchorTop": true
		}
	}
}
		`)
	if err != nil {
		fmt.Println(err)
		return
	}

	ge := layouts.NewGridEngine(gcfg)
	g := layouts.NewLayouter(ge)

	rg := widgets.NewRadio([]string{"bread", "cake", "beheadings"})
	ge.AddName("radio", &rg.Block)

	l := widgets.NewLabel(geom.Coord{100, 30}, widgets.LabelConfig{"text", 14, color.Black})
	ge.AddName("label", &layouts.NewPadBox(layouts.PadConfig{Right: 10}, &l.Block).Block)

	selLis := make(widgets.SelectionListener, 1)
	rg.AddSelectionListener <- selLis

	w.SetPane(&g.Block)

	w.Show()

	done := make(chan interface{}, 1)
	isDone := func(e interface{}) (accept, done bool) {
		_, accept = e.(uik.CloseEvent)
		done = accept
		return
	}
	w.Block.Subscribe <- uik.Subscription{isDone, done}

loop:
	for {
		select {
		case sel := <-selLis:
			l.SetConfig(widgets.LabelConfig{
				Text:     fmt.Sprintf("Clicked option %d, %q", sel.Index, sel.Option),
				FontSize: 14,
				Color:    color.Black,
			})
		case <-done:
			break loop
		}
	}

	w.W.Close()

	wde.Stop()
}
