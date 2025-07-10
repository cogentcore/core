// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileinfo

//go:generate core generate

import (
	"fmt"
)

// Known is an enumerated list of known file types, for which
// appropriate actions can be taken etc.
type Known int32 //enums:enum

// KnownMimes maps from the known type into the MimeType info for each
// known file type; the known MimeType may be just one of
// multiple that correspond to the known type; it should be first in list
// and have extensions defined
var KnownMimes map[Known]MimeType

// MimeString gives the string representation of the canonical mime type
// associated with given known mime type.
func MimeString(kn Known) string {
	mt, has := KnownMimes[kn]
	if !has {
		// log.Printf("fileinfo.MimeString called with unrecognized 'Known' type: %v\n", sup)
		return ""
	}
	return mt.Mime
}

// Cat returns the Cat category for given known file type
func (kn Known) Cat() Categories {
	if kn == Unknown {
		return UnknownCategory
	}
	mt, has := KnownMimes[kn]
	if !has {
		// log.Printf("fileinfo.KnownCat called with unrecognized 'Known' type: %v\n", sup)
		return UnknownCategory
	}
	return mt.Cat
}

// IsMatch returns true if given file type matches target type,
// which could be any of the Any options
func IsMatch(targ, typ Known) bool {
	if targ == Any {
		return true
	}
	if targ == AnyKnown {
		return typ != Unknown
	}
	if targ == typ {
		return true
	}
	cat := typ.Cat()
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
func IsMatchList(targs []Known, typ Known) bool {
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

// KnownByName looks up known file type by caps or lowercase name
func KnownByName(name string) (Known, error) {
	var kn Known
	err := kn.SetString(name)
	if err != nil {
		err = fmt.Errorf("fileinfo.KnownByName: doesn't look like that is a known file type: %v", name)
		return kn, err
	}
	return kn, nil
}

// These are the super high-frequency used mime types, to have very quick const level support
const (
	TextPlain = "text/plain"
	DataJson  = "application/json"
	DataXml   = "application/xml"
	DataCsv   = "text/csv"
)

// These are the known file types, organized by category
const (
	// Unknown = a non-known file type
	Unknown Known = iota

	// Any is used when selecting a file type, if any type is OK (including Unknown)
	// see also AnyKnown and the Any options for each category
	Any

	// AnyKnown is used when selecting a file type, if any Known
	// file type is OK (excludes Unknown) -- see Any and Any options for each category
	AnyKnown

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
	Cosh
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
	Goal
	Haskell
	Java
	JavaScript
	TypeScript
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
	SQL
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
	Yaml
	Toml
	// special support for data fs
	Number
	String
	Tensor
	Table

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
	Bmp
	Heic
	Heif

	// Model is a 3D model
	AnyModel
	Vrml
	X3d
	Obj

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
)
