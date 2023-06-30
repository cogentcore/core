// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"image"
	"log"

	omouse "github.com/goki/gi/oswin/mouse"
	otouch "github.com/goki/gi/oswin/touch"
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
					app.winPtr = a.Window()
					log.Println("on start, window uintptr:", app.winPtr)
					err := app.newWindow(nil)
					if err != nil {
						log.Fatalln("error creating window in lifecycle cross on:", err)
					}
					app.window.window = a.Window()
					log.Println("set window pointer to", app.window.window)
				case lifecycle.CrossOff:
					log.Println("on stop")
					// todo: on stop
				}
			case size.Event:
				log.Println("size event", e.Size())
				app.window.SetSize(e.Size())
			case paint.Event:
				log.Println("paint event")
				// app.onPaint()
				app.window.sendWindowEvent(window.Paint)
				a.Publish()
			case touch.Event:
				log.Println("touch event", e)
				// app.window.sendWindowEvent(window.Paint)
				app.window.SendEmptyEvent()
				app.window.touchEvent(e)
				app.window.SendEmptyEvent()
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
	oevent := &otouch.Event{
		Where:    image.Point{X: int(event.X), Y: int(event.Y)},
		Sequence: otouch.Sequence(event.Sequence),
		Action:   otouch.Actions(event.Type), // otouch.Actions and touch.Type have the same enum constant values
	}
	oevent.Init()
	log.Println("oswin touch event", oevent.EventBase, oevent.Where, oevent.Sequence, oevent.Action)
	w.Send(oevent)
	action := omouse.Press
	if event.Type == touch.TypeEnd {
		action = omouse.Release
	}
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
