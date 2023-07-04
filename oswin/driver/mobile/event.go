// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android || ios

package mobile

import (
	"image"
	"log"

	"github.com/goki/gi/oswin"
	okey "github.com/goki/gi/oswin/key"
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
		app.mobapp = a
		for e := range a.Events() {
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					app.RunOnMain(func() {
						err := app.newWindow(nil, a.Window())
						if err != nil {
							log.Fatalln("error creating window in lifecycle cross on:", err)
						}
					})
				case lifecycle.CrossOff:
					log.Println("on stop")
					// we need to set the size of the window to 0 so that it detects a size difference
					// and lets the size event go through when we come back later
					app.window.SetSize(image.Point{})
					app.window.sendWindowEvent(window.Minimize)

					app.RunOnMain(app.destroyVk)
				}
				switch e.Crosses(lifecycle.StageFocused) {
				case lifecycle.CrossOn:
					app.window.focus(true)
				case lifecycle.CrossOff:
					app.window.focus(false)
				}
				switch e.Crosses(lifecycle.StageAlive) {
				case lifecycle.CrossOff:
					app.RunOnMain(app.fullDestroyVk)
					app.stopMain()
				}
			case size.Event:
				log.Println("size event", e.Size())
				app.window.size = e
				app.mu.Lock()
				app.getScreen()
				app.mu.Unlock()
				oswin.InitScreenLogicalDPIFunc()
				app.window.LogDPI = app.screens[0].LogicalDPI
				app.window.sendWindowEvent(window.ScreenUpdate)
			case paint.Event:
				app.mu.Lock()
				log.Println("paint event")
				app.window.sendWindowEvent(window.Paint)
				a.Publish()
				app.mu.Unlock()
			case touch.Event:
				log.Println("touch event", e)
				app.window.touchEvent(e)
			case mouse.Event:
				log.Println("mouse event", e)
			case key.Event:
				log.Println("key event", e)
				app.window.keyEvent(e)
			}
		}
	})
}

func (w *windowImpl) touchEvent(event touch.Event) {
	// TODO: decide whether to implement touch
	// oevent := &otouch.Event{
	// 	Where:    image.Point{X: int(event.X), Y: int(event.Y)},
	// 	Sequence: otouch.Sequence(event.Sequence),
	// 	Action:   otouch.Actions(event.Type), // otouch.Actions and touch.Type have the same enum constant values
	// }
	// oevent.Init()
	// log.Println("oswin touch event", oevent.EventBase, oevent.Where, oevent.Sequence, oevent.Action)
	// w.Send(oevent)

	if event.Type == touch.TypeMove {
		pos := image.Point{X: int(event.X), Y: int(event.Y)}
		oevent := &omouse.DragEvent{
			MoveEvent: omouse.MoveEvent{
				Event: omouse.Event{
					Where:  pos,
					Button: omouse.Left,
					Action: omouse.Drag,
				},
				From: w.lastMouseMovePos,
			},
			Start: w.lastMouseButtonPos,
		}
		w.lastMouseMovePos = pos
		oevent.Init()
		log.Printf("oswin mouse move event %#v", oevent)
		w.Send(oevent)
		return
	}

	action := omouse.Press
	if event.Type == touch.TypeEnd {
		action = omouse.Release
	}

	pos := image.Point{X: int(event.X), Y: int(event.Y)}

	oevent := &omouse.Event{
		Where:  pos,
		Button: omouse.Left,
		Action: action,
	}
	w.lastMouseButtonPos = pos
	oevent.Init()
	log.Printf("oswin mouse event %#v", oevent)
	w.Send(oevent)
}

func (w *windowImpl) keyEvent(event key.Event) {
	if event.Direction != key.DirRelease {
		return
	}
	oevent := &okey.ChordEvent{
		Event: okey.Event{
			Code:      okey.Codes(event.Code),
			Rune:      event.Rune,
			Modifiers: int32(event.Modifiers),
			Action:    okey.Actions(event.Direction),
		},
	}
	oevent.Init()
	log.Printf("gi event: %#v\n", oevent)
	w.Send(oevent)
}

// for sending window.Event's
func (w *windowImpl) sendWindowEvent(act window.Actions) {
	winEv := window.Event{
		Action: act,
	}
	winEv.Init()
	log.Printf("Sent window event %#v\n", winEv)
	w.Send(&winEv)
}
