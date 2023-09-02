// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package filecat categorizes file types -- it is the single, consolidated
// place where mimetypes and filetypes etc are managed in GoGi / GoKi.
//
// This whole space is a bit of a heterogenous mess -- most file types
// we care about are not registered on the official iana registry, which
// seems mainly to have paid registrations in application/ category,
// and doesn't have any of the main programming languages etc.
//
// The official Go std library support depends on different platform
// libraries and mac doesn't have one, so it has very limited support
//
// So we sucked it up and made a full list of all the major file types
// that we really care about and also provide a broader category-level organization
// that is useful for functionally organizing / operating on files.
//
// As fallback, we are using both of the go packages developed by others:
// github.com/gabriel-vasile/mimetype
// github.com/h2non/filetype
// (which constrained our package name to not be either of those two)
package filecat

import (
	"goki.dev/ki/v2/kit"
)

// filecat.Cat is a functional category for files -- a broad functional
// categorization that can help decide what to do with the file.
//
// It is computed in part from the mime type, but some types require
// other information.
//
// No single categorization scheme is perfect, so any given use
// may require examination of the full mime type etc, but this
// provides a useful broad-scope categorization of file types.
type Cat int32

const (
	// Unknown is an unknown file category
	Unknown Cat = iota

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

	CatN
)

//go:generate stringer -type=Cat

var KiT_Cat = kit.Enums.AddEnum(CatN, kit.NotBitFlag, nil)

func (kf Cat) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(kf) }
func (kf *Cat) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(kf, b) }

// map keys require text marshaling:
func (ev Cat) MarshalText() ([]byte, error)  { return kit.EnumMarshalText(ev) }
func (ev *Cat) UnmarshalText(b []byte) error { return kit.EnumUnmarshalText(ev, b) }

// filecat.CatFromMime returns the file category based on the mime type -- not all
// Cats can be inferred from file types
func CatFromMime(mime string) Cat {
	if mime == "" {
		return Unknown
	}
	mime = MimeNoChar(mime)
	if mt, has := AvailMimes[mime]; has {
		return mt.Cat // must be set!
	}
	// try from type:
	ms := MimeTop(mime)
	if ms == "" {
		return Unknown
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
	return Unknown
}
