// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"cogentcore.org/core/units"
)

// A Legend gives a description of the meaning of different
// data elements of the plot.  Each legend entry has a name
// and a thumbnail, where the thumbnail shows a small
// sample of the display style of the corresponding data.
type Legend struct {
	// TextStyle is the style given to the legend entry texts.
	TextStyle TextStyle

	// Padding is the amount of padding to add
	// between each entry in the legend.  If Padding
	// is zero then entries are spaced based on the
	// font size.
	Padding units.Value

	// Top and Left specify the location of the legend.
	// If Top is true the legend is located along the top
	// edge of the plot, otherwise it is located along
	// the bottom edge.  If Left is true then the legend
	// is located along the left edge of the plot, and the
	// text is positioned after the icons, otherwise it is
	// located along the right edge and the text is
	// positioned before the icons.
	Top, Left bool

	// XOffs and YOffs are added to the legend's final position.
	XOffs, YOffs units.Value

	// YPosition specifies the vertical position of a legend entry.
	// Valid values are [-1,+1], with +1 being the top of the
	// entry vertical space, and -1 the bottom.
	YPosition float64

	// ThumbnailWidth is the width of legend thumbnails.
	ThumbnailWidth units.Value

	// Entries are all of the LegendEntries described by this legend.
	Entries []LegendEntry
}

func (lg *Legend) Defaults() {
	lg.TextStyle.Defaults()
	lg.YPosition = 0.9
	lg.ThumbnailWidth.Pt(20)
}

// Draw draws the legend to the given draw.Canvas.
func (lg *Legend) Draw(pt *Plot) {
	/*
		iconx := c.Min.X
		sty := lg.TextStyle
		em := sty.Rectangle(" ")
		textx := iconx + lg.ThumbnailWidth + em.Max.X
		if !lg.Left {
			iconx = c.Max.X - lg.ThumbnailWidth
			textx = iconx - em.Max.X
			sty.XAlign--
		}
		textx += lg.XOffs
		iconx += lg.XOffs

		descent := sty.FontExtents().Descent
		enth := lg.entryHeight()
		y := c.Max.Y - enth - descent
		if !lg.Top {
			y = c.Min.Y + (enth+lg.Padding)*(vg.Length(len(lg.entries))-1)
		}
		y += lg.YOffs

		icon := &draw.Canvas{
			Canvas: c.Canvas,
			Rectangle: vg.Rectangle{
				Min: vg.Point{X: iconx, Y: y},
				Max: vg.Point{X: iconx + lg.ThumbnailWidth, Y: y + enth},
			},
		}

		if lg.YPosition < draw.PosBottom || draw.PosTop < lg.YPosition {
			panic("plot: invalid vertical offset for the legend's entries")
		}
		yoff := vg.Length(lg.YPosition-draw.PosBottom) / 2
		yoff += descent

		for _, e := range lg.entries {
			for _, t := range e.thumbs {
				t.Thumbnail(icon)
			}
			yoffs := (enth - descent - sty.Rectangle(e.text).Max.Y) / 2
			yoffs += yoff
			c.FillText(sty, vg.Point{X: textx, Y: icon.Min.Y + yoffs}, e.text)
			icon.Min.Y -= enth + lg.Padding
			icon.Max.Y -= enth + lg.Padding
		}
	*/
}

/*
// Rectangle returns the extent of the Legend.
func (lg *Legend) Rectangle(c draw.Canvas) vg.Rectangle {
	var width, height vg.Length
	sty := lg.TextStyle
	entryHeight := lg.entryHeight()
	for i, e := range lg.entries {
		width = vg.Length(math.Max(float64(width), float64(lg.ThumbnailWidth+sty.Rectangle(" "+e.text).Max.X)))
		height += entryHeight
		if i != 0 {
			height += lg.Padding
		}
	}
	var r vg.Rectangle
	if lg.Left {
		r.Max.X = c.Max.X
		r.Min.X = c.Max.X - width
	} else {
		r.Max.X = c.Min.X + width
		r.Min.X = c.Min.X
	}
	if lg.Top {
		r.Max.Y = c.Max.Y
		r.Min.Y = c.Max.Y - height
	} else {
		r.Max.Y = c.Min.Y + height
		r.Min.Y = c.Min.Y
	}
	return r
}

// entryHeight returns the height of the tallest legend
// entry text.
func (lg *Legend) entryHeight() (height vg.Length) {
	for _, e := range lg.entries {
		if h := lg.TextStyle.Rectangle(e.text).Max.Y; h > height {
			height = h
		}
	}
	return
}
*/

// Add adds an entry to the legend with the given name.
// The entry's thumbnail is drawn as the composite of all of the
// thumbnails.
func (lg *Legend) Add(name string, thumbs ...Thumbnailer) {
	lg.Entries = append(lg.Entries, LegendEntry{Text: name, Thumbs: thumbs})
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
