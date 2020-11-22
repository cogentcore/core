// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"image"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

// TextLink represents a hyperlink within rendered text
type TextLink struct {
	Label     string   `desc:"text label for the link"`
	URL       string   `desc:"full URL for the link"`
	Props     ki.Props `desc:"properties defined for the link"`
	StartSpan int      `desc:"span index where link starts"`
	StartIdx  int      `desc:"index in StartSpan where link starts"`
	EndSpan   int      `desc:"span index where link ends (can be same as EndSpan)"`
	EndIdx    int      `desc:"index in EndSpan where link ends (index of last rune in label)"`
	Widget    ki.Ki    `desc:"the widget that owns this text link -- only set prior to passing off to handler function"`
}

// Bounds returns the bounds of the link
func (tl *TextLink) Bounds(tr *Text, pos mat32.Vec2) image.Rectangle {
	stsp := &tr.Spans[tl.StartSpan]
	tpos := pos.Add(stsp.RelPos)
	sr := &(stsp.Render[tl.StartIdx])
	sp := tpos.Add(sr.RelPos)
	sp.Y -= sr.Size.Y
	ep := sp
	if tl.EndSpan == tl.StartSpan {
		er := &(stsp.Render[tl.EndIdx])
		ep = tpos.Add(er.RelPos)
		ep.X += er.Size.X
	} else {
		er := &(stsp.Render[len(stsp.Render)-1])
		ep = tpos.Add(er.RelPos)
		ep.X += er.Size.X
	}
	return image.Rectangle{Min: sp.ToPointFloor(), Max: ep.ToPointCeil()}
}

// TextLinkHandlerFunc is a function that handles TextLink links -- returns
// true if the link was handled, false if not (in which case it might be
// passed along to someone else)
type TextLinkHandlerFunc func(tl TextLink) bool

// TextLinkHandler is used to handle TextLink links, if non-nil -- set this to
// your own handler to get first crack at all the text link clicks -- if this
// function returns false (or is nil) then the URL is sent to URLHandler (the
// default one just calls oswin.TheApp.OpenURL)
var TextLinkHandler TextLinkHandlerFunc

// URLHandlerFunc is a function that handles URL links -- returns
// true if the link was handled, false if not (in which case it might be
// passed along to someone else).
type URLHandlerFunc func(url string) bool

// URLHandler is used to handle URL links, if non-nil -- set this to your own
// handler to process URL's, depending on TextLinkHandler -- the default
// version of this function just calls oswin.TheApp.OpenURL -- setting this to
// nil will prevent any links from being open that way, and your own function
// will have full responsibility for links if set (i.e., the return value is ignored)
var URLHandler = func(url string) bool {
	if oswin.TheApp != nil {
		oswin.TheApp.OpenURL(url)
		return true
	}
	return false
}
