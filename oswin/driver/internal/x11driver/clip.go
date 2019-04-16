// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd
// +build !3d

package x11driver

import (
	"log"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/goki/gi/oswin/mimedata"
)

// implements clipboard support for X11
// https://github.com/jtanx/libclipboard/blob/master/src/clipboard_x11.c
// https://www.uninformativ.de/blog/postings/2017-04-02/0/POSTING-en.html
// Qt source: qtbase/src/plugins/platforms/xcb/qxcbwindow.cpp

// favor CLIPBOARD for our writing and check it first -- standard explicit
// cut/paste one PRIMARY is for mouse-selected text that usually pasted with
// middle-mouse-button

type clipImpl struct {
	lastWrite mimedata.Mimes
}

var theClip = clipImpl{}

// ClipTransSize determines how much data in bytes to transfer per call, 1MB
var ClipTransSize = uint32(1048576)

// ClipTimeOut determines how long to wait before timing out waiting for the
// SelectionNotifyEvent
var ClipTimeOut = 1 * time.Second

func (ci *clipImpl) IsEmpty() bool {
	selown, err := xproto.GetSelectionOwner(theApp.xc, theApp.atomClipboardSel).Reply()
	if err != nil {
		log.Printf("X11 Clipboard Read error: %v\n", err)
		return false
	}
	if selown.Owner == xproto.AtomNone {
		selown, err = xproto.GetSelectionOwner(theApp.xc, theApp.atomPrimarySel).Reply()
		if err != nil {
			log.Printf("X11 Clipboard Read error: %v\n", err)
			return false
		}
	}
	if selown.Owner == xproto.AtomNone { // nothing there..
		return true
	}
	return false
}

func (ci *clipImpl) Read(types []string) mimedata.Mimes {
	if types == nil {
		return nil
	}
	// check for an owner on either the CLIPBOARD or PRIMARY selections
	// owner info is not actually used under the XCB/XGB protocol -- just to see
	// that it exists..
	useSel := theApp.atomClipboardSel
	selown, err := xproto.GetSelectionOwner(theApp.xc, theApp.atomClipboardSel).Reply()
	if err != nil {
		log.Printf("X11 Clipboard Read error: %v\n", err)
		return nil
	}
	if selown.Owner == xproto.AtomNone {
		useSel = theApp.atomPrimarySel
		selown, err = xproto.GetSelectionOwner(theApp.xc, theApp.atomPrimarySel).Reply()
		if err != nil {
			log.Printf("X11 Clipboard Read error: %v\n", err)
			return nil
		}
	}
	if selown.Owner == xproto.AtomNone { // nothing there..
		return nil
	}

	if selown.Owner == theApp.window32 { // we are the owner -- just send our data
		return ci.lastWrite
	}

	wantText := mimedata.IsText(types[0])

	if wantText {
		// this is the main call requesting the selection -- there are no apparent
		// docs for the xcb version of this call, in terms of the "Property" arg,
		// but example from jtanx just uses the name of the selection again, so...
		xproto.ConvertSelection(theApp.xc, theApp.window32, useSel, theApp.atomUTF8String, useSel, xproto.TimeCurrentTime)

		// var ptyp xproto.Atom
		b := make([]byte, 0, 1024)

		select {
		case ev := <-theApp.selNotifyChan:
			bytesAfter := uint32(1)
			bufsz := uint32(0) // current buffer size
			for bytesAfter > 0 {
				// last two args are offset and amount to transfer, in 32bit "long" sizes
				prop, err := xproto.GetProperty(theApp.xc, true, theApp.window32, ev.Property, xproto.AtomAny, bufsz/4, ClipTransSize/4).Reply()
				if err != nil {
					log.Printf("X11 Clipboard Read Property error: %v\n", err)
					return nil
				}
				bytesAfter = prop.BytesAfter
				sz := len(prop.Value)
				if sz > 0 {
					b = append(b, prop.Value...)
					bufsz += uint32(sz)
				}
				// ptyp = prop.Type
			}
		case <-time.After(ClipTimeOut):
			log.Printf("X11 Clipboard Read: unexpected timeout on receipt of SelectionNotifyEvent\n")
			return nil
		}

		isMulti, mediaType, boundary, body := mimedata.IsMultipart(b)
		if isMulti {
			return mimedata.FromMultipart(body, boundary)
		} else {
			if mediaType != "" { // found a mime type encoding
				return mimedata.NewMime(mediaType, b)
			} else {
				// we can't really figure out type, so just assume..
				return mimedata.NewMime(types[0], b)
			}
		}
	} else {
		// todo: deal with image formats etc
	}
	return nil
}

func (ci *clipImpl) Write(data mimedata.Mimes) error {
	// we just advertise ourselves as clipboard owners and save the data until
	// someone requests it..
	ci.lastWrite = data
	useSel := theApp.atomClipboardSel
	xproto.SetSelectionOwner(theApp.xc, theApp.window32, useSel, xproto.TimeCurrentTime)
	return nil
}

func (ci *clipImpl) SendLastWrite(ev xproto.SelectionRequestEvent) {
	reply := xproto.SelectionNotifyEvent{
		Time:      ev.Time,
		Requestor: ev.Requestor,
		Selection: ev.Selection,
		Target:    ev.Target,
		Property:  xproto.AtomNone,
	}

	mask := xproto.EventMaskNoEvent
	if ci.lastWrite != nil {
		reply.Property = ev.Property
		if reply.Property == xproto.AtomNone {
			reply.Property = reply.Target
		}
		switch reply.Target {
		case theApp.atomTargets: // requesting to know what targets we support
			mask = xproto.EventMaskPropertyChange
			targs := make([]byte, 4*3)
			bi := 0
			xgb.Put32(targs[bi:], uint32(theApp.atomUTF8String))
			bi += 4
			xgb.Put32(targs[bi:], uint32(theApp.atomTimestamp))
			bi += 4
			xgb.Put32(targs[bi:], uint32(theApp.atomTargets))
			xproto.ChangeProperty(theApp.xc, xproto.PropModeReplace, reply.Requestor,
				reply.Property, xproto.AtomAtom, 32, 3, targs)
		case theApp.atomTimestamp:
			mask = xproto.EventMaskPropertyChange
			targs := make([]byte, 4*1)
			xgb.Put32(targs, uint32(xproto.TimeCurrentTime))
			xproto.ChangeProperty(theApp.xc, xproto.PropModeReplace, reply.Requestor,
				reply.Property, xproto.AtomInteger, 32, 1, targs)
		case theApp.atomUTF8String:
			mask = xproto.EventMaskPropertyChange
			if len(ci.lastWrite) > 1 {
				mpd := ci.lastWrite.ToMultipart()
				xproto.ChangeProperty(theApp.xc, xproto.PropModeReplace, reply.Requestor,
					reply.Property, reply.Target, 8, uint32(len(mpd)), mpd)
			} else {
				d := ci.lastWrite[0]
				xproto.ChangeProperty(theApp.xc, xproto.PropModeReplace, reply.Requestor,
					reply.Property, reply.Target, 8, uint32(len(d.Data)), d.Data)
			}
		}
	}
	xproto.SendEvent(theApp.xc, false, reply.Requestor, uint32(mask), string(reply.Bytes()))
}

func (ci *clipImpl) Clear() {
	ci.lastWrite = nil
	xproto.SetSelectionOwner(theApp.xc, xproto.AtomNone, theApp.atomClipboardSel, xproto.TimeCurrentTime)
}
