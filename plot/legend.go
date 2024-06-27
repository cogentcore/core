// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/styles/units"
)

// LegendPosition specifies where to put the legend
type LegendPosition struct {
	// Top and Left specify the location of the legend.
	Top, Left bool

	// XOffs and YOffs are added to the legend's final position,
	// relative to the relevant anchor position
	XOffs, YOffs units.Value
}

func (lg *LegendPosition) Defaults() {
	lg.Top = true
}

// A Legend gives a description of the meaning of different
// data elements of the plot.  Each legend entry has a name
// and a thumbnail, where the thumbnail shows a small
// sample of the display style of the corresponding data.
type Legend struct {
	// TextStyle is the style given to the legend entry texts.
	TextStyle TextStyle

	// position of the legend
	Position LegendPosition `display:"inline"`

	// ThumbnailWidth is the width of legend thumbnails.
	ThumbnailWidth units.Value

	// Fill specifies the background fill color for the legend box,
	// if non-nil.
	Fill image.Image

	// Entries are all of the LegendEntries described by this legend.
	Entries []LegendEntry
}

func (lg *Legend) Defaults() {
	lg.TextStyle.Defaults()
	lg.TextStyle.Padding.Dp(2)
	lg.TextStyle.Font.Size.Dp(20)
	lg.Position.Defaults()
	lg.ThumbnailWidth.Pt(20)
	lg.Fill = gradient.ApplyOpacity(colors.Scheme.Surface, 0.75)
}

// Add adds an entry to the legend with the given name.
// The entry's thumbnail is drawn as the composite of all of the
// thumbnails.
func (lg *Legend) Add(name string, thumbs ...Thumbnailer) {
	lg.Entries = append(lg.Entries, LegendEntry{Text: name, Thumbs: thumbs})
}

// LegendForPlotter returns the legend Text for given plotter,
// if it exists as a Thumbnailer in the legend entries.
// Otherwise returns empty string.
func (lg *Legend) LegendForPlotter(plt Plotter) string {
	for _, e := range lg.Entries {
		for _, tn := range e.Thumbs {
			if tp, isp := tn.(Plotter); isp && tp == plt {
				return e.Text
			}
		}
	}
	return ""
}

// Thumbnailer wraps the Thumbnail method, which
// draws the small image in a legend representing the
// style of data.
type Thumbnailer interface {
	// Thumbnail draws an thumbnail representing
	// a legend entry.  The thumbnail will usually show
	// a smaller representation of the style used
	// to plot the corresponding data.
	Thumbnail(pt *Plot)
}

// A LegendEntry represents a single line of a legend, it
// has a name and an icon.
type LegendEntry struct {
	// text is the text associated with this entry.
	Text string

	// thumbs is a slice of all of the thumbnails styles
	Thumbs []Thumbnailer
}
