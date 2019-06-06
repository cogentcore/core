// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filecat

import (
	"fmt"
	"log"
	"strings"

	"github.com/goki/ki/kit"
)

// filecat.Supported are file types that are specifically supported by GoGi
// and can be processed in one way or another, plus various others
// that we SHOULD be able to process at some point
type Supported int

// SupportedMimes maps from the support type into the MimeType info for each
// supported file type -- the supported MimeType may be just one of
// multiple that correspond to the supported type -- it should be first in list
// and have extensions defined
var SupportedMimes map[Supported]MimeType

// MimeString gives the string representation of the canonical mime type
// associated with given supported mime type.
func MimeString(sup Supported) string {
	mt, has := SupportedMimes[sup]
	if !has {
		log.Printf("filecat.MimeString called with unrecognized 'Supported' type: %v\n", sup)
		return ""
	}
	return mt.Mime
}

// SupportedCat returns the Cat category for given supported file type
func SupportedCat(sup Supported) Cat {
	if sup == NoSupport {
		return Unknown
	}
	mt, has := SupportedMimes[sup]
	if !has {
		log.Printf("filecat.SupportedCat called with unrecognized 'Supported' type: %v\n", sup)
		return Unknown
	}
	return mt.Cat
}

// IsMatch returns true if given file type matches target type,
// which could be any of the Any options
func IsMatch(targ, typ Supported) bool {
	if targ == Any {
		return true
	}
	if targ == AnySupported {
		return typ != NoSupport
	}
	if targ == typ {
		return true
	}
	cat := SupportedCat(typ)
	switch targ {
	case AnyFolder:
		return cat == Folder
	case AnyArchive:
		return cat == Archive
	case AnyBackup:
		return cat == Backup
	case AnyCode:
		return cat == Code
	case AnyDoc:
		return cat == Doc
	case AnySheet:
		return cat == Sheet
	case AnyData:
		return cat == Data
	case AnyText:
		return cat == Text
	case AnyImage:
		return cat == Image
	case AnyModel:
		return cat == Model
	case AnyAudio:
		return cat == Audio
	case AnyVideo:
		return cat == Video
	case AnyFont:
		return cat == Font
	case AnyExe:
		return cat == Exe
	case AnyBin:
		return cat == Bin
	}
	return false
}

// IsMatchList returns true if given file type matches any of a list of targets
// if list is empty, then always returns true
func IsMatchList(targs []Supported, typ Supported) bool {
	if len(targs) == 0 {
		return true
	}
	for _, trg := range targs {
		if IsMatch(trg, typ) {
			return true
		}
	}
	return false
}

// SupportedByName looks up supported file type by caps or lowercase name
func SupportedByName(name string) (Supported, error) {
	var sup Supported
	err := sup.FromString(name)
	if err != nil {
		if err != nil {
			name = strings.ToLower(name)
			err = kit.Enums.SetEnumIfaceFromAltString(&sup, name) // alts = lowercase
			if err != nil {
				err = fmt.Errorf("filecat.SupportedByName: doesn't look like that is a supported file type: %v", name)
				return sup, err
			}
		}
	}
	return sup, nil
}

// These are the super high-frequency used mime types, to have very quick const level support
const (
	TextPlain = "text/plain"
	DataJson  = "application/json"
	DataXml   = "application/xml"
	DataCsv   = "text/csv"
)

// These are the supported file types, organized by category
const (
	// NoSupport = a non-supported file type
	NoSupport Supported = iota

	// Any is used when selecting a file type, if any type is OK (including NoSupport)
	// see also AnySupported and the Any options for each category
	Any

	// AnySupported is used when selecting a file type, if any Supported
	// file type is OK (excludes NoSupport) -- see Any and Any options for each category
	AnySupported

	// Folder is a folder / directory
	AnyFolder

	// Archive is a collection of files, e.g., zip tar
	AnyArchive
	Multipart
	Tar
	Zip
	GZip
	SevenZ
	Xz
	BZip
	Dmg
	Shar

	// Backup files
	AnyBackup
	Trash

	// Code is a programming language file
	AnyCode
	Ada
	Bash
	Csh
	C // includes C++
	CSharp
	D
	Diff
	Eiffel
	Erlang
	Forth
	Fortran
	FSharp
	Go
	Haskell
	Java
	JavaScript
	Lisp
	Lua
	Makefile
	Mathematica
	Matlab
	ObjC
	OCaml
	Pascal
	Perl
	Php
	Prolog
	Python
	R
	Ruby
	Rust
	Scala
	Tcl

	// Doc is an editable word processing file including latex, markdown, html, css, etc
	AnyDoc
	BibTeX
	TeX
	Texinfo
	Troff

	Html
	Css
	Markdown
	Rtf
	MSWord
	OpenText
	OpenPres
	MSPowerpoint

	EBook
	EPub

	// Sheet is a spreadsheet file (.xls etc)
	AnySheet
	MSExcel
	OpenSheet

	// Data is some kind of data format (csv, json, database, etc)
	AnyData
	Csv
	Json
	Xml
	Protobuf
	Ini
	Tsv
	Uri
	Color
	GoGi
	Yaml

	// Text is some other kind of text file
	AnyText
	PlainText // text/plain mimetype -- used for clipboard
	ICal
	VCal
	VCard

	// Image is an image (jpeg, png, svg, etc) *including* PDF
	AnyImage
	Pdf
	Postscript
	Gimp
	GraphVis
	Gif
	Jpeg
	Png
	Svg
	Tiff
	Pnm
	Pbm
	Pgm
	Ppm
	Xbm
	Xpm

	// Model is a 3D model
	AnyModel
	Vrml
	X3d

	// Audio is an audio file
	AnyAudio
	Aac
	Flac
	Mp3
	Ogg
	Midi
	Wav

	// Video is a video file
	AnyVideo
	Mpeg
	Mp4
	Mov
	Ogv
	Wmv
	Avi

	// Font is a font file
	AnyFont
	TrueType
	WebOpenFont

	// Exe is a binary executable file
	AnyExe

	// Bin is some other unrecognized binary type
	AnyBin

	SupportedN
)

//go:generate stringer -type=Supported

var KiT_Supported = kit.Enums.AddEnumAltLower(SupportedN, false, nil, "")

func (kf Supported) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(kf) }
func (kf *Supported) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(kf, b) }

// map keys require text marshaling:
func (ev Supported) MarshalText() ([]byte, error)  { return kit.EnumMarshalText(ev) }
func (ev *Supported) UnmarshalText(b []byte) error { return kit.EnumUnmarshalText(ev, b) }
