// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filecat

import "github.com/goki/ki/kit"

// filecat.Support are file types that are specifically supported by GoGi
// and can be processed in one way or another, plus various others
// that we SHOULD be able to process at some point
type Support int

// SupportMimes maps from the support type into the MimeType info for each
// supported file type -- the supported MimeType may be just one of
// multiple that correspond to the supported type -- it should be first in list
// and have extensions defined
var SupportMimes map[Support]MimeType

// These are the supported file types, organized by category
const (
	// NoSupport = a non-supported file type
	NoSupport Support = iota

	// Archive is a collection of files, e.g., zip tar
	Tar
	Zip
	GZip
	SevenZ
	Xz
	Dmg
	Shar

	// Program is a programming language file
	Ada
	Bash
	C // includes C++
	Csh
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
	Scala
	Tcl

	// Document is an editable word processing file including latex, markdown, html, css, etc
	Bibtex
	Tex
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

	// Spreadsheet is a spreadsheet file (.xls etc)
	MSExcel
	OpenSheet

	// Data is some kind of data format (csv, json, database, etc)
	Csv
	Json
	Xml
	Protobuf
	Ini
	Tsv

	// Text is some other kind of text file
	PlainText // text/plain mimetype -- used for clipboard
	ICal
	VCal
	VCard

	// Image is an image (jpeg, png, svg, etc) *including* PDF
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
	Vrml
	X3d

	// Audio is an audio file
	Flac
	Mp3
	Ogg
	Midi
	Wav

	// Video is a video file
	Mpeg
	Mp4
	Mov
	Ogv
	Wmv
	Avi

	// Font is a font file

	// Exe is a binary executable file

	// Binary is some other unrecognized binary type

	SupportN
)

//go:generate stringer -type=Support

var KiT_Support = kit.Enums.AddEnum(SupportN, false, nil)

func (kf Support) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(kf) }
func (kf *Support) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(kf, b) }

// map keys require text marshaling:
func (ev Support) MarshalText() ([]byte, error)  { return kit.EnumMarshalText(ev) }
func (ev *Support) UnmarshalText(b []byte) error { return kit.EnumUnmarshalText(ev, b) }
