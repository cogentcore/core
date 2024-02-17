// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fi

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cogentcore.org/core/glop/datasize"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/vci"
	"github.com/Bios-Marcel/wastebasket"
)

// FileInfo represents the information about a given file / directory,
// including icon, mimetype, etc
type FileInfo struct { //gti:add

	// icon for file
	Ic icons.Icon `tableview:"no-header"`

	// name of the file, without any path
	Name string `width:"40"`

	// size of the file
	Size datasize.Size

	// type of file / directory; shorter, more user-friendly
	// version of mime type, based on category
	Kind string `width:"20" max-width:"20"`

	// full official mime type of the contents
	Mime string `tableview:"-"`

	// functional category of the file, based on mime data etc
	Cat Cat `tableview:"-"`

	// known file type
	Known Known `tableview:"-"`

	// file mode bits
	Mode os.FileMode `tableview:"-"`

	// time that contents (only) were last modified
	ModTime FileTime `label:"Last modified"`

	// version control system status, when enabled
	Vcs vci.FileStatus `tableview:"-"`

	// full path to file, including name; for file functions
	Path string `tableview:"-"`
}

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
	fi.Size = datasize.Size(info.Size())
	fi.Mode = info.Mode()
	fi.ModTime = FileTime(info.ModTime())
	if info.IsDir() {
		fi.Kind = "Folder"
		fi.Cat = Folder
		fi.Known = AnyFolder
	} else {
		fi.Cat = UnknownCat
		fi.Known = Unknown
		fi.Kind = ""
		mtyp, _, err := MimeFromFile(fi.Path)
		if err == nil {
			fi.Mime = mtyp
			fi.Cat = CatFromMime(fi.Mime)
			fi.Known = MimeKnown(fi.Mime)
			if fi.Cat != UnknownCat {
				fi.Kind = fi.Cat.String() + ": "
			}
			if fi.Known != Unknown {
				fi.Kind += fi.Known.String()
			} else {
				fi.Kind += MimeSub(fi.Mime)
			}
		}
		if fi.Cat == UnknownCat {
			if fi.IsExec() {
				fi.Cat = Exe
			}
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
func (fi *FileInfo) Duplicate() (string, error) { //gti:add
	if fi.IsDir() {
		err := fmt.Errorf("giv.Duplicate: cannot copy directory: %v", fi.Path)
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
	return dst, CopyFile(dst, fi.Path, fi.Mode)
}

// Delete moves the file to the trash / recycling bin.
// On mobile and web, it deletes it directly.
func (fi *FileInfo) Delete() error { //gti:add
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
		err = fmt.Errorf("giv.Rename: new name is empty")
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
func (fi *FileInfo) Rename(path string) (newpath string, err error) { //gti:add
	orgpath := fi.Path
	newpath, err = fi.RenamePath(path)
	if err != nil {
		return
	}
	err = os.Rename(orgpath, newpath)
	if err == nil {
		fi.Path = newpath
		_, fi.Name = filepath.Split(newpath)
	}
	return
}

// FindIcon uses file info to find an appropriate icon for this file -- uses
// Kind string first to find a correspondingly-named icon, and then tries the
// extension.  Returns true on success.
func (fi *FileInfo) FindIcon() (icons.Icon, bool) {
	if fi.IsDir() {
		return icons.Folder, true
	}
	if fi.Known != Unknown {
		snm := strings.ToLower(fi.Known.String())
		if icn := icons.Icon(snm); icn.IsValid() {
			return icn, true
		}
		if icn := icons.Icon("file-" + snm); icn.IsValid() {
			return icn, true
		}
		if icn := icons.Icon(snm + "_file"); icn.IsValid() {
			return icn, true
		}
	}
	subt := strings.ToLower(MimeSub(fi.Mime))
	if subt != "" {
		if icn := icons.Icon(subt); icn.IsValid() {
			return icn, true
		}
	}
	if fi.Cat != UnknownCat {
		cat := strings.ToLower(fi.Cat.String())
		if icn := icons.Icon(cat); icn.IsValid() {
			return icn, true
		}
		if icn := icons.Icon("file-" + cat); icn.IsValid() {
			return icn, true
		}
		if icn := icons.Icon(cat + "_file"); icn.IsValid() {
			return icn, true
		}
	}
	ext := filepath.Ext(fi.Name)
	if ext != "" {
		if icn := icons.Icon(ext[1:]); icn.IsValid() {
			return icn, true
		}
	}
	if fi.IsExec() {
		return icons.PlayArrow, true
	}

	icn := icons.None
	return icn, false
}

//////////////////////////////////////////////////////////////////////////////
//    FileTime, FileSize

// Note: can get all the detailed birth, access, change times from this package
// 	"github.com/djherbis/times"

// FileTime provides a default String format for file modification times, and
// other useful methods -- will plug into Value with date / time editor.
type FileTime time.Time

// Int satisfies the ints.Inter interface for sorting etc
func (ft FileTime) Int() int64 {
	return (time.Time(ft)).Unix()
}

// FromInt satisfies the ints.Inter interface
func (ft *FileTime) FromInt(val int64) {
	*ft = FileTime(time.Unix(val, 0))
}

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
	tmp, err := os.CreateTemp(filepath.Dir(dst), "")
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
