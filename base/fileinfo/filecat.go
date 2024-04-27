// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileinfo

// Categories is a functional category for files; a broad functional
// categorization that can help decide what to do with the file.
//
// It is computed in part from the mime type, but some types require
// other information.
//
// No single categorization scheme is perfect, so any given use
// may require examination of the full mime type etc, but this
// provides a useful broad-scope categorization of file types.
type Categories int32 //enums:enum

const (
	// UnknownCategory is an unknown file category
	UnknownCategory Categories = iota

	// Folder is a folder / directory
	Folder

	// Archive is a collection of files, e.g., zip tar
	Archive

	// Backup is a backup file (# ~ etc)
	Backup

	// Code is a programming language file
	Code

	// Doc is an editable word processing file including latex, markdown, html, css, etc
	Doc

	// Sheet is a spreadsheet file (.xls etc)
	Sheet

	// Data is some kind of data format (csv, json, database, etc)
	Data

	// Text is some other kind of text file
	Text

	// Image is an image (jpeg, png, svg, etc) *including* PDF
	Image

	// Model is a 3D model
	Model

	// Audio is an audio file
	Audio

	// Video is a video file
	Video

	// Font is a font file
	Font

	// Exe is a binary executable file (scripts go in Code)
	Exe

	// Bin is some other type of binary (object files, libraries, etc)
	Bin
)

// CategoryFromMime returns the file category based on the mime type;
// not all Categories can be inferred from file types
func CategoryFromMime(mime string) Categories {
	if mime == "" {
		return UnknownCategory
	}
	mime = MimeNoChar(mime)
	if mt, has := AvailableMimes[mime]; has {
		return mt.Cat // must be set!
	}
	// try from type:
	ms := MimeTop(mime)
	if ms == "" {
		return UnknownCategory
	}
	switch ms {
	case "image":
		return Image
	case "audio":
		return Audio
	case "video":
		return Video
	case "font":
		return Font
	case "model":
		return Model
	}
	if ms == "text" {
		return Text
	}
	return UnknownCategory
}
