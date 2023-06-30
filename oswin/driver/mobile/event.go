// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android || ios

package mobile

import (
	"image"
	"log"

	"github.com/goki/gi/oswin"
	omouse "github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/oswin/window"
	mapp "github.com/goki/mobile/app"
	"github.com/goki/mobile/event/key"
	"github.com/goki/mobile/event/lifecycle"
	"github.com/goki/mobile/event/mouse"
	"github.com/goki/mobile/event/paint"
	"github.com/goki/mobile/event/size"
	"github.com/goki/mobile/event/touch"
)

// eventLoop starts running the mobile app event loop
func (app *appImpl) eventLoop() {
	mapp.Main(func(a mapp.App) {
		for e := range a.Events() {
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					err := app.newWindow(nil, a.Window())
					if err != nil {
						log.Fatalln("error creating window in lifecycle cross on:", err)
					}
				case lifecycle.CrossOff:
					log.Println("on stop")
					// todo: on stop
				}
				switch e.Crosses(lifecycle.StageFocused) {
				case lifecycle.CrossOn:
					app.window.focus(true)
				case lifecycle.CrossOff:
					app.window.focus(false)
				}
			case size.Event:
				log.Println("size event", e.Size())
				app.window.size = e
				app.window.SetSize(e.Size())
				app.mu.Lock()
				app.getScreen()
				app.mu.Unlock()
				oswin.InitScreenLogicalDPIFunc()
				app.window.LogDPI = app.screens[0].LogicalDPI
				app.window.sendWindowEvent(window.Resize)
				app.window.sendWindowEvent(window.Paint)
			case paint.Event:
				log.Println("paint event")
				// app.onPaint()
				app.window.sendWindowEvent(window.Paint)
				a.Publish()
			case touch.Event:
				log.Println("touch event", e)
				// app.window.sendWindowEvent(window.Paint)
				app.window.touchEvent(e)
				a.Publish()
				// todo: on touch
			case mouse.Event:
				log.Println("mouse event", e)
			case key.Event:
				log.Println("key event", e)
			}
		}
	})
}

func (w *windowImpl) touchEvent(event touch.Event) {
	// oevent := &otouch.Event{
	// 	Where:    image.Point{X: int(event.X), Y: int(event.Y)},
	// 	Sequence: otouch.Sequence(event.Sequence),
	// 	Action:   otouch.Actions(event.Type), // otouch.Actions and touch.Type have the same enum constant values
	// }
	// oevent.Init()
	// log.Println("oswin touch event", oevent.EventBase, oevent.Where, oevent.Sequence, oevent.Action)
	// w.Send(oevent)
	action := omouse.Press
	if event.Type == touch.TypeEnd {
		action = omouse.Release
	}

	// ommvevent := &omouse.MoveEvent{
	// 	From: image.Point{X: int(0), Y: int(0)},
	// }
	// ommvevent.Where = image.Point{X: int(event.X), Y: int(event.Y)}
	// ommvevent.Action = omouse.Move
	//
	// ommvevent.Init()
	// log.Println("oswin mouse move event", ommvevent.EventBase, ommvevent.Where, ommvevent.Button, ommvevent.Action)
	// w.Send(ommvevent)

	omevent := &omouse.Event{
		Where:  image.Point{X: int(event.X), Y: int(event.Y)},
		Button: omouse.Left,
		Action: action,
	}
	omevent.Init()
	log.Println("oswin mouse event", omevent.EventBase, omevent.Where, omevent.Button, omevent.Action)
	w.Send(omevent)
}

// for sending window.Event's
func (w *windowImpl) sendWindowEvent(act window.Actions) {
	winEv := window.Event{
		Action: act,
	}
	winEv.Init()
	w.Send(&winEv)
}
