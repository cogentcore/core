// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/chroma/lexers"
	"github.com/c2h5oh/datasize"
	"github.com/gabriel-vasile/mimetype"
	"github.com/goki/gi"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// FileInfo represents the information about a given file / directory,
// including icon, mimetype, etc
type FileInfo struct {
	Ic      gi.IconName `tableview:"no-header" desc:"icon for file"`
	Name    string      `width:"40" desc:"name of the file, without any path"`
	Size    FileSize    `desc:"size of the file in bytes"`
	Kind    string      `width:"20" max-width:"20" desc:"type of file / directory -- shorter, more user-friendly version of mime type"`
	Mime    string      `tableview:"-" desc:"full official mime type of the contents"`
	Mode    os.FileMode `desc:"file mode bits"`
	ModTime FileTime    `desc:"time that contents (only) were last modified"`
	Path    string      `view:"-" tableview:"-" desc:"full path to file, including name -- for file functions"`
}

var KiT_FileInfo = kit.Types.AddType(&FileInfo{}, FileInfoProps)

// NewFileInfo returns a new FileInfo based on a filename -- directly returns
// filepath.Abs or os.Stat error on the given file.  filename can be anything
// that works given current directory -- Path will contain the full
// filepath.Abs path, and Name will be just the filename.
func NewFileInfo(fname string) (*FileInfo, error) {
	fi := &FileInfo{}
	err := fi.InitFile(fname)
	return fi, err
}

// InitFile initializes a FileInfo based on a filename -- directly returns
// filepath.Abs or os.Stat error on the given file.  filename can be anything
// that works given current directory -- Path will contain the full
// filepath.Abs path, and Name will be just the filename.
func (fi *FileInfo) InitFile(fname string) error {
	path, err := filepath.Abs(fname)
	if err != nil {
		return err
	}
	fi.Path = path
	_, fi.Name = filepath.Split(path)
	return fi.Stat()
}

// Stat runs os.Stat on file, returns any error directly but otherwise updates
// file info, including mime type, which then drives Kind and Icon -- this is
// the main function to call to update state.
func (fi *FileInfo) Stat() error {
	info, err := os.Stat(fi.Path)
	if err != nil {
		return err
	}
	fi.Size = FileSize(info.Size())
	fi.Mode = info.Mode()
	fi.ModTime = FileTime(info.ModTime())
	if info.IsDir() {
		fi.Kind = "Folder"
	} else {
		mtyp, _, err := MimeFromFile(fi.Path)
		if err == nil {
			fi.Mime = mtyp
			fi.Kind = FileKindFromMime(fi.Mime)
		}
	}
	icn, _ := fi.FindIcon()
	fi.Ic = icn
	return nil
}

// IsDir returns true if file is a directory (folder)
func (fi *FileInfo) IsDir() bool {
	return fi.Mode.IsDir()
}

// IsExec returns true if file is an executable file
func (fi *FileInfo) IsExec() bool {
	return fi.Mode&0111 != 0
}

//////////////////////////////////////////////////////////////////////////////
//    File ops

// Duplicate creates a copy of given file -- only works for regular files, not
// directories.
func (fi *FileInfo) Duplicate() error {
	if fi.IsDir() {
		err := fmt.Errorf("giv.Duplicate: cannot copy directory: %v", fi.Path)
		log.Println(err)
		return err
	}
	ext := filepath.Ext(fi.Path)
	noext := strings.TrimSuffix(fi.Path, ext)
	dst := noext + "_Copy" + ext
	return CopyFile(dst, fi.Path, fi.Mode)
}

// Delete deletes this file -- does not work on directories (todo: fix)
func (fi *FileInfo) Delete() error {
	if fi.IsDir() {
		err := fmt.Errorf("giv.Delete: cannot deleted directory: %v", fi.Path)
		log.Println(err)
		return err
	}
	return os.Remove(fi.Path)
	// note: we should be deleted now!
}

// Rename renames file to new name
func (fi *FileInfo) Rename(newpath string) error {
	if newpath == "" {
		err := fmt.Errorf("giv.Rename: new name is empty")
		log.Println(err)
		return err
	}
	if newpath == fi.Path {
		return nil
	}
	ndir, np := filepath.Split(newpath)
	if ndir == "" {
		if np == fi.Name {
			return nil
		}
		dir, _ := filepath.Split(fi.Path)
		newpath = filepath.Join(dir, newpath)
	}
	err := os.Rename(fi.Path, newpath)
	if err == nil {
		fi.InitFile(newpath)
	}
	return err
}

// MimeFromFile gets mime type from file, using Gabriel Vasile's mimetype
// package, mime.TypeByExtension, the chroma syntax highlighter,
// CustomExtMimeMap, and finally FileExtMimeMap.  Use the mimetype package's
// extension mechanism to add further content-based matchers as needed, and
// set CustomExtMimeMap to your own map or call AddCustomExtMime for
// extension-based ones.
func MimeFromFile(fname string) (mtype, ext string, err error) {
	mtyp, ext, err := mimetype.DetectFile(fname)
	if err == nil {
		// todo: may have custom overrides / specializations here
		return mtyp, ext, err
	}
	ext = filepath.Ext(fname)
	mtyp = mime.TypeByExtension(ext)
	if mtyp != "" {
		return mtyp, strings.ToLower(ext), nil
	}
	lexer := lexers.Match(fname) // todo: could get start of file and pass to
	// Analyze, but might be too slow..
	if lexer != nil {
		if len(lexer.Config().MimeTypes) > 0 {
			mtyp = lexer.Config().MimeTypes[0]
			return mtyp, ext, nil
		}
		mtyp := "application/" + strings.ToLower(lexer.Config().Name)
		return mtyp, ext, nil
	}
	ext = strings.ToLower(ext)
	if CustomExtMimeMap != nil {
		if mtyp, ok := CustomExtMimeMap[ext]; ok {
			return mtyp, ext, nil
		}
	}
	if mtyp, ok := FileExtMimeMap[ext]; ok {
		return mtyp, ext, nil
	}
	return "", ext, fmt.Errorf("giv.MimeFromFile could not find mime type for ext: %v file: %v", ext, fname)
}

// FileKindFromMime returns simplfied Kind description based on the given full
// mime type string.  Strips out application/ prefix for example.
func FileKindFromMime(mime string) string {
	switch {
	case strings.HasPrefix(mime, "application/"):
		return strings.TrimPrefix(mime, "application/")
	}
	return mime
}

// FindIcon uses file info to find an appropriate icon for this file -- uses
// Kind string first to find a correspondingly-named icon, and then tries the
// extension.  Returns true on success.
func (fi *FileInfo) FindIcon() (gi.IconName, bool) {
	kind := fi.Kind
	icn := gi.IconName(kind)
	if icn.IsValid() {
		return icn, true
	}
	kind = strings.ToLower(kind)
	icn = gi.IconName(kind)
	if icn.IsValid() {
		return icn, true
	}
	if fi.IsDir() {
		return gi.IconName("folder"), true
	}
	if strings.Contains(kind, "/") {
		si := strings.IndexByte(kind, '/')
		typ := kind[:si]
		subtyp := kind[si+1:]
		if icn = "file-" + gi.IconName(subtyp); icn.IsValid() {
			return icn, true
		}
		if icn = gi.IconName(subtyp); icn.IsValid() {
			return icn, true
		}
		if ms, ok := KindToIconMap[string(subtyp)]; ok {
			if icn = gi.IconName(ms); icn.IsValid() {
				return icn, true
			}
		}
		if icn = "file-" + gi.IconName(typ); icn.IsValid() {
			return icn, true
		}
		if icn = gi.IconName(typ); icn.IsValid() {
			return icn, true
		}
		if ms, ok := KindToIconMap[string(typ)]; ok {
			if icn = gi.IconName(ms); icn.IsValid() {
				return icn, true
			}
		}
	}
	ext := filepath.Ext(fi.Name)
	if ext != "" {
		if icn = gi.IconName(ext[1:]); icn.IsValid() {
			return icn, true
		}
	}

	icn = gi.IconName("none")
	return icn, false
}

var FileInfoProps = ki.Props{
	"CtxtMenu": ki.PropSlice{
		{"Duplicate", ki.Props{
			"updtfunc": func(fii interface{}, act *gi.Action) {
				fi := fii.(*FileInfo)
				act.SetInactiveStateUpdt(fi.IsDir())
			},
		}},
		{"Delete", ki.Props{
			"desc":    "Ok to delete this file?  This is not undoable and is not moving to trash / recycle bin",
			"confirm": true,
			"updtfunc": func(fii interface{}, act *gi.Action) {
				fi := fii.(*FileInfo)
				act.SetInactiveStateUpdt(fi.IsDir())
			},
		}},
		{"Rename", ki.Props{
			"desc": "Rename file to new file name",
			"Args": ki.PropSlice{
				{"New Name", ki.Props{
					"default-field": "Name",
				}},
			},
		}},
	},
}

//////////////////////////////////////////////////////////////////////////////
//    FileTime, FileSize

// Note: can get all the detailed birth, access, change times from this package
// 	"github.com/djherbis/times"

// FileTime provides a default String format for file modification times, and
// other useful methods -- will plug into ValueView with date / time editor.
type FileTime time.Time

func (ft FileTime) String() string {
	return (time.Time)(ft).Format("Mon Jan  2 15:04:05 MST 2006")
}

func (ft FileTime) MarshalBinary() ([]byte, error) {
	return time.Time(ft).MarshalBinary()
}

func (ft FileTime) MarshalJSON() ([]byte, error) {
	return time.Time(ft).MarshalJSON()
}

func (ft FileTime) MarshalText() ([]byte, error) {
	return time.Time(ft).MarshalText()
}

func (ft *FileTime) UnmarshalBinary(data []byte) error {
	return (*time.Time)(ft).UnmarshalBinary(data)
}

func (ft *FileTime) UnmarshalJSON(data []byte) error {
	return (*time.Time)(ft).UnmarshalJSON(data)
}

func (ft *FileTime) UnmarshalText(data []byte) error {
	return (*time.Time)(ft).UnmarshalText(data)
}

type FileSize datasize.ByteSize

func (fs FileSize) String() string {
	return (datasize.ByteSize)(fs).HumanReadable()
}

//////////////////////////////////////////////////////////////////////////////
//    CopyFile

// here's all the discussion about why CopyFile is not in std lib:
// https://old.reddit.com/r/golang/comments/3lfqoh/why_golang_does_not_provide_a_copy_file_func/
// https://github.com/golang/go/issues/8868

// CopyFile copies the contents from src to dst atomically.
// If dst does not exist, CopyFile creates it with permissions perm.
// If the copy fails, CopyFile aborts and dst is preserved.
func CopyFile(dst, src string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := ioutil.TempFile(filepath.Dir(dst), "")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, in)
	if err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err = tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err = os.Chmod(tmp.Name(), perm); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

//////////////////////////////////////////////////////////////////////////////
//    Mime type / Icon name maps

// FileExtMimeMap is the builtin map of extensions (lowercase) to mime types
// -- used as a last resort when everything else fails!
var FileExtMimeMap = map[string]string{
	".gide": "application/gide",
	".go":   "application/go",
	".py":   "application/python",
	".cpp":  "application/python",
}

// CustomExtMimeMap can be set to your own map of extensions (lowercase) to
// mime types to cover any special cases needed for your app, not otherwise
// covered
var CustomExtMimeMap map[string]string

// AddCustomExtMime adds given extension (lowercase), mime to the
// FileExtMimeMap -- see also CustomExtMimeMap to install a full map.
func AddCustomExtMime(ext, mime string) {
	FileExtMimeMap[ext] = mime
}

// KindToIconMap has special cases for mapping mime type to icon, for those
// that basic string doesn't work
var KindToIconMap = map[string]string{
	"svg+xml": "svg",
}
