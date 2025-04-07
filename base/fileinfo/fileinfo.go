// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fileinfo manages file information and categorizes file types;
// it is the single, consolidated place where file info, mimetypes, and
// filetypes are managed in Cogent Core.
//
// This whole space is a bit of a heterogenous mess; most file types
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
// As fallback, we are this Go package:
// github.com/h2non/filetype
package fileinfo

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/datasize"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/icons"
	"github.com/Bios-Marcel/wastebasket/v2"
)

// FileInfo represents the information about a given file / directory,
// including icon, mimetype, etc
type FileInfo struct { //types:add

	// icon for file
	Ic icons.Icon `table:"no-header"`

	// name of the file, without any path
	Name string `width:"40"`

	// size of the file
	Size datasize.Size

	// type of file / directory; shorter, more user-friendly
	// version of mime type, based on category
	Kind string `width:"20" max-width:"20"`

	// full official mime type of the contents
	Mime string `table:"-"`

	// functional category of the file, based on mime data etc
	Cat Categories `table:"-"`

	// known file type
	Known Known `table:"-"`

	// file mode bits
	Mode fs.FileMode `table:"-"`

	// time that contents (only) were last modified
	ModTime time.Time `label:"Last modified"`

	// version control system status, when enabled
	VCS vcs.FileStatus `table:"-"`

	// Generated indicates that the file is generated and should not be edited.
	// For Go files, this regex: `^// Code generated .* DO NOT EDIT\.$` is used.
	Generated bool `table:"-"`

	// full path to file, including name; for file functions
	Path string `table:"-"`
}

// NewFileInfo returns a new FileInfo for given file.
func NewFileInfo(fname string) (*FileInfo, error) {
	fi := &FileInfo{}
	err := fi.InitFile(fname)
	return fi, err
}

// NewFileInfoType returns a new FileInfo representing the given file type.
func NewFileInfoType(ftyp Known) *FileInfo {
	fi := &FileInfo{}
	fi.SetType(ftyp)
	return fi
}

// InitFile initializes a FileInfo for os file based on a filename,
// which is updated to full path using filepath.Abs.
// Returns error from filepath.Abs and / or fs.Stat error on the given file,
// but file info will be updated based on the filename even if
// the file does not exist.
func (fi *FileInfo) InitFile(fname string) error {
	fi.Cat = UnknownCategory
	fi.Known = Unknown
	fi.Generated = false
	fi.Kind = ""
	var errs []error
	path, err := filepath.Abs(fname)
	if err == nil {
		fi.Path = path
	} else {
		fi.Path = fname
	}
	_, fi.Name = filepath.Split(path)
	info, err := os.Stat(fi.Path)
	if err != nil {
		errs = append(errs, err)
		fi.MimeFromFilename()
	} else {
		fi.SetFileInfo(info)
	}
	return errors.Join(errs...)
}

// InitFileFS initializes a FileInfo based on filename in given fs.FS.
// Returns error from fs.Stat error on the given file,
// but file info will be updated based on the filename even if
// the file does not exist.
func (fi *FileInfo) InitFileFS(fsys fs.FS, fname string) error {
	fi.Cat = UnknownCategory
	fi.Known = Unknown
	fi.Generated = false
	fi.Kind = ""
	var errs []error
	fi.Path = fname
	_, fi.Name = path.Split(fname)
	info, err := fs.Stat(fsys, fi.Path)
	if err != nil {
		errs = append(errs, err)
		fi.MimeFromFilename()
	} else {
		fi.SetFileInfo(info)
	}
	return errors.Join(errs...)
}

// MimeFromFilename sets the mime data based only on the filename
// without attempting to open the file.
func (fi *FileInfo) MimeFromFilename() error {
	ext := strings.ToLower(filepath.Ext(fi.Path))
	if mtype, has := ExtMimeMap[ext]; has { // only use our filename ext map
		fi.SetMimeFromType(mtype)
		return nil
	}
	return errors.New("FileInfo MimeFromFilename: Filename extension not known: " + ext)
}

// MimeFromFile sets the mime data for a valid file (i.e., os.Stat works).
// Use MimeFromFilename to only examine the filename.
func (fi *FileInfo) MimeFromFile() error {
	if fi.Path == "" || fi.Path == "." || fi.IsDir() {
		return nil
	}
	fi.Generated = IsGeneratedFile(fi.Path)
	mtype, _, err := MimeFromFile(fi.Path)
	if err != nil {
		return err
	}
	fi.SetMimeFromType(mtype)
	return nil
}

// SetMimeType sets file info fields from given mime type string.
func (fi *FileInfo) SetMimeFromType(mtype string) {
	fi.Mime = mtype
	fi.Cat = CategoryFromMime(mtype)
	fi.Known = MimeKnown(mtype)
	if fi.Cat != UnknownCategory {
		fi.Kind = fi.Cat.String() + ": "
	}
	if fi.Known != Unknown {
		fi.Kind += fi.Known.String()
	} else {
		fi.Kind += MimeSub(fi.Mime)
	}
}

// SetFileInfo updates from given [fs.FileInfo]. It uses a canonical
// [FileInfo.ModTime] when testing to ensure consistent results.
func (fi *FileInfo) SetFileInfo(info fs.FileInfo) {
	fi.Size = datasize.Size(info.Size())
	fi.Mode = info.Mode()
	if testing.Testing() {
		// We use a canonical time when testing to ensure consistent results.
		fi.ModTime = time.Unix(1500000000, 0)
	} else {
		fi.ModTime = info.ModTime()
	}
	if info.IsDir() {
		fi.Kind = "Folder"
		fi.Cat = Folder
		fi.Known = AnyFolder
	} else {
		if fi.Mode.IsRegular() {
			fi.MimeFromFile()
		}
		if fi.Cat == UnknownCategory {
			if fi.IsExec() {
				fi.Cat = Exe
				fi.Known = AnyExe
			}
		}
	}
	icn, _ := fi.FindIcon()
	fi.Ic = icn
}

// SetType sets file type information for given Known file type
func (fi *FileInfo) SetType(ftyp Known) {
	mt := MimeFromKnown(ftyp)
	fi.Mime = mt.Mime
	fi.Cat = mt.Cat
	fi.Known = mt.Known
	if fi.Name == "" && len(mt.Exts) > 0 {
		fi.Name = "_fake" + mt.Exts[0]
		fi.Path = fi.Name
	}
	fi.Kind = fi.Cat.String() + ": "
	if fi.Known != Unknown {
		fi.Kind += fi.Known.String()
	}
}

// IsDir returns true if file is a directory (folder)
func (fi *FileInfo) IsDir() bool {
	return fi.Mode.IsDir()
}

// IsExec returns true if file is an executable file
func (fi *FileInfo) IsExec() bool {
	if fi.Mode&0111 != 0 {
		return true
	}
	ext := filepath.Ext(fi.Path)
	return ext == ".exe"
}

// IsSymLink returns true if file is a symbolic link
func (fi *FileInfo) IsSymlink() bool {
	return fi.Mode&os.ModeSymlink != 0
}

// IsHidden returns true if file name starts with . or _ which are typically hidden
func (fi *FileInfo) IsHidden() bool {
	return fi.Name == "" || fi.Name[0] == '.' || fi.Name[0] == '_'
}

//////////////////////////////////////////////////////////////////////////////
//    File ops

// Duplicate creates a copy of given file -- only works for regular files, not
// directories.
func (fi *FileInfo) Duplicate() (string, error) { //types:add
	if fi.IsDir() {
		err := fmt.Errorf("core.Duplicate: cannot copy directory: %v", fi.Path)
		log.Println(err)
		return "", err
	}
	ext := filepath.Ext(fi.Path)
	noext := strings.TrimSuffix(fi.Path, ext)
	dst := noext + "_Copy" + ext
	cpcnt := 0
	for {
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			cpcnt++
			dst = noext + fmt.Sprintf("_Copy%d", cpcnt) + ext
		} else {
			break
		}
	}
	return dst, fsx.CopyFile(dst, fi.Path, fi.Mode)
}

// Delete moves the file to the trash / recycling bin.
// On mobile and web, it deletes it directly.
func (fi *FileInfo) Delete() error { //types:add
	err := wastebasket.Trash(fi.Path)
	if errors.Is(err, wastebasket.ErrPlatformNotSupported) {
		return os.RemoveAll(fi.Path)
	}
	return err
}

// Filenames recursively adds fullpath filenames within the starting directory to the "names" slice.
// Directory names within the starting directory are not added.
func Filenames(d os.File, names *[]string) (err error) {
	nms, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, n := range nms {
		fp := filepath.Join(d.Name(), n)
		ffi, ferr := os.Stat(fp)
		if ferr != nil {
			return ferr
		}
		if ffi.IsDir() {
			dd, err := os.Open(fp)
			if err != nil {
				return err
			}
			defer dd.Close()
			Filenames(*dd, names)
		} else {
			*names = append(*names, fp)
		}
	}
	return nil
}

// Filenames returns a slice of file names from the starting directory and its subdirectories
func (fi *FileInfo) Filenames(names *[]string) (err error) {
	if !fi.IsDir() {
		err = errors.New("not a directory: Filenames returns a list of files within a directory")
		return err
	}
	path := fi.Path
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()

	err = Filenames(*d, names)
	return err
}

// RenamePath returns the proposed path or the new full path.
// Does not actually do the renaming -- see Rename method.
func (fi *FileInfo) RenamePath(path string) (newpath string, err error) {
	if path == "" {
		err = errors.New("core.Rename: new name is empty")
		log.Println(err)
		return path, err
	}
	if path == fi.Path {
		return "", nil
	}
	ndir, np := filepath.Split(path)
	if ndir == "" {
		if np == fi.Name {
			return path, nil
		}
		dir, _ := filepath.Split(fi.Path)
		newpath = filepath.Join(dir, np)
	}
	return newpath, nil
}

// Rename renames (moves) this file to given new path name.
// Updates the FileInfo setting to the new name, although it might
// be out of scope if it moved into a new path
func (fi *FileInfo) Rename(path string) (newpath string, err error) { //types:add
	orgpath := fi.Path
	newpath, err = fi.RenamePath(path)
	if err != nil {
		return
	}
	err = os.Rename(string(orgpath), newpath)
	if err == nil {
		fi.Path = newpath
		_, fi.Name = filepath.Split(newpath)
	}
	return
}

// FindIcon uses file info to find an appropriate icon for this file -- uses
// Kind string first to find a correspondingly named icon, and then tries the
// extension.  Returns true on success.
func (fi *FileInfo) FindIcon() (icons.Icon, bool) {
	if fi.IsDir() {
		return icons.Folder, true
	}
	return Icons[fi.Known], true
}

// Note: can get all the detailed birth, access, change times from this package
// 	"github.com/djherbis/times"
