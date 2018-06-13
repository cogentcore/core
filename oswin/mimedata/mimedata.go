// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mimedata defines MIME data support used in clipboard and
// drag-and-drop functions in the GoGi GUI.  mimedata.Data struct contains format
// and []byte data, and multiple representations of the same data are encoded
// in mimedata.Mimes which is just a []mimedata.Data slice
package mimedata

// Data represents one element of MIME data as a type string and byte slice
type Data struct {
	// MIME Type string representing the data, e.g., text/plain, text/html, text/xml, text/uri-list, image/jpg, png etc
	Type string

	// Data for the item
	Data []byte
}

// various standard MIME types -- see http://www.iana.org/assignments/media-types/media-types.xhtml for a big list
var (
	// for ALL text formats, Go standard utf8 encoding is assumed -- standard for most?
	TextAny   = "text/*"
	TextPlain = "text/plain"
	TextHTML  = "text/html"
	TextURL   = "text/uri-list"
	TextCSS   = "text/css"
	TextCSV   = "text/csv"
	// .ics calendar entry
	TextCalendar = "text/calendar"
	// text version of XML is for human-readable xml
	TextXML = "text/xml"

	ImageAny  = "image/*"
	ImageJPEG = "image/jepg"
	ImagePNG  = "image/png"
	ImageGIF  = "image/gif"
	ImageTIFF = "image/tiff"
	ImageSVG  = "image/svg+xml"

	AudioAny  = "audio/*"
	AudioWAV  = "audio/wav"
	AudioMIDI = "audio/midi"
	AudioOGG  = "audio/ogg"
	AudioAAC  = "audio/aac"
	AudioMPEG = "audio/mpeg"
	AudioMP4  = "audio/mp4"

	VideoAny       = "video/*"
	VideoMPEG      = "video/mpeg"
	VideoAVI       = "video/x-msvideo"
	VideoOGG       = "video/ogg"
	VideoMP4       = "video/mp4"
	VideoH264      = "video/h264"
	VideoQuicktime = "video/quicktime"

	FontAny = "font/*"
	FontTTF = "font/ttf"

	AppRTF  = "application/rtf"
	AppPDF  = "application/pdf"
	AppJSON = "application/json"
	// app version of XML is for non-human-readable xml content
	AppXML        = "application/xml"
	AppColor      = "application/x-color"
	AppJavaScript = "application/javascript"
	AppGo         = "application/go"

	// use this as a prefix for any GoGi-specific elements (e.g., AppGoGi + ".treeview")
	AppGoGi = "application/vnd.gogi"
)

// NewTextData returns a Data representation of the string -- good idea to
// always have a text/plain representation of everything on clipboard /
// drag-n-drop
func NewTextData(text string) *Data {
	return &Data{TextPlain, []byte(text)}
}

// Mimes is a slice of mime data, potentially encoding the same data in
// different formats -- this is used for all oswin API's for maximum
// flexibility
type Mimes []*Data

// NewText returns a Mimes representation of the string as a single text/plain Data
func NewText(text string) Mimes {
	md := NewTextData(text)
	mi := make(Mimes, 1)
	mi[0] = md
	return mi
}

// NewTextPlus returns a Mimes representation of an item as a text string plus
// a more specific type
func NewTextPlus(text, typ string, data []byte) Mimes {
	md := NewTextData(text)
	mi := make(Mimes, 2)
	mi[0] = md
	mi[1] = &Data{typ, data}
	return mi
}

// NewMime returns a Mimes representation of one element
func NewMime(typ string, data []byte) Mimes {
	mi := make(Mimes, 1)
	mi[0] = &Data{typ, data}
	return mi
}

// HasType returns true if Mimes has given type of data available
func (mi Mimes) HasType(typ string) bool {
	for _, d := range mi {
		if d.Type == typ {
			return true
		}
	}
	return false
}

// TypeData returns data associated with given MIME type
func (mi Mimes) TypeData(typ string) []byte {
	for _, d := range mi {
		if d.Type == typ {
			return d.Data
		}
	}
	return nil
}

// Text attempts to extract text-format data from a Mimes -- returns "" if not there
func (mi Mimes) Text() string {
	for _, d := range mi {
		if d.Type == TextPlain {
			return string(d.Data)
		}
	}
	return ""
}

// todo: image, etc extractors
