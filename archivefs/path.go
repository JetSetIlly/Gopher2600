// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
package archivefs

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Node represents a single part of a full path
type Node struct {
	Name string

	// a directory has the the field of IsDir set to true
	IsDir bool

	// a recognised archive file has InArchive set to true. note that an archive
	// file is also considered to be directory
	IsArchive bool
}

func (e Node) String() string {
	return e.Name
}

// Path represents a single destination in the file system
type Path struct {
	current string
	isDir   bool

	zf *zip.ReadCloser

	// if the path is inside a zip file, we split the in-zip path into the path
	// to a file and the file itself
	inZipPath string
	inZipFile string
}

// String returns the current path
func (afs Path) String() string {
	return afs.current
}

// Base returns the last element of the current path
func (afs Path) Base() string {
	return filepath.Base(afs.current)
}

// Dir returns all but the last element of path
func (afs Path) Dir() string {
	if afs.isDir {
		return afs.current
	}
	return filepath.Dir(afs.current)
}

// IsDir returns true if Path is currently set to a directory. For the purposes
// of archivefs, the root of an archive is treated as a directory
func (afs Path) IsDir() bool {
	return afs.isDir
}

// InArchive returns true if path is currently inside an archive
func (afs Path) InArchive() bool {
	return afs.zf != nil
}

// Open and return an io.ReadSeeker for the filename previously set by the Set()
// function.
//
// Returns the io.ReadSeeker, the size of the data behind the ReadSeeker and any
// errors.
func (afs Path) Open() (io.ReadSeeker, int, error) {
	if afs.zf != nil {
		f, err := afs.zf.Open(filepath.Join(afs.inZipPath, afs.inZipFile))
		if err != nil {
			return nil, 0, err
		}
		defer f.Close()

		b, err := io.ReadAll(f)
		if err != nil {
			return nil, 0, err
		}

		return bytes.NewReader(b), len(b), nil
	}

	f, err := os.Open(afs.current)
	if err != nil {
		return nil, 0, err
	}

	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	return f, int(info.Size()), nil
}

// Close any open zip files and reset path
func (afs *Path) Close() {
	afs.current = ""
	afs.isDir = false
	afs.inZipPath = ""
	afs.inZipFile = ""
	if afs.zf != nil {
		afs.zf.Close()
		afs.zf = nil
	}
}

// List returns the child entries for the current path location. If the current
// path is a file then the list will be the contents of the containing directory
// of that file
func (afs *Path) List() ([]Node, error) {
	var ent []Node

	if afs.zf != nil {
		for _, f := range afs.zf.File {
			// split file name into parts. the list is joined together again
			// below to create the path to the file. this is better than
			// filepath.Dir() because that will add path components that make it
			// awkward to compare with afs.inZipPath
			flst := strings.Split(filepath.Clean(f.Name), string(filepath.Separator))
			fdir := filepath.Join(flst[:len(flst)-1]...)

			// if path to the file is not the same as inZipPath then continue
			// with the next file
			if fdir != afs.inZipPath {
				continue
			}

			fi := f.FileInfo()
			if fi.IsDir() {
				ent = append(ent, Node{
					Name:  fi.Name(),
					IsDir: true,
				})
			} else {
				ent = append(ent, Node{
					Name: fi.Name(),
				})
			}
		}
	} else {
		path := afs.current
		if !afs.isDir {
			path = filepath.Dir(path)
		}

		dir, err := os.ReadDir(path)
		if err != nil {
			return []Node{}, fmt.Errorf("archivefs: entries: %w", err)
		}

		for _, d := range dir {
			// using os.Stat() to get file information otherwise links to
			// directories do not have the IsDir() property
			fi, err := os.Stat(filepath.Join(path, d.Name()))
			if err != nil {
				continue
			}

			if fi.IsDir() {
				ent = append(ent, Node{
					Name:  d.Name(),
					IsDir: true,
				})
			} else {
				p := filepath.Join(path, d.Name())
				_, err := zip.OpenReader(p)
				if err == nil {
					ent = append(ent, Node{
						Name:      d.Name(),
						IsDir:     true,
						IsArchive: true,
					})
				} else {
					ent = append(ent, Node{
						Name: d.Name(),
					})
				}
			}
		}
	}

	// sort so that directories are at the start of the list
	sort.Slice(ent, func(i int, j int) bool {
		return ent[i].IsDir
	})

	// sort alphabetically (case insensitive)
	sort.SliceStable(ent, func(i int, j int) bool {
		return strings.ToLower(ent[i].Name) < strings.ToLower(ent[j].Name)
	})

	return ent, nil
}

func (afs *Path) Set(path string) error {
	afs.Close()

	// clean path and split into parts
	path = filepath.Clean(path)
	lst := strings.Split(path, string(filepath.Separator))

	// strings.Split will remove a leading filepath.Separator. we need to add
	// one back so that filepath.Join() works as expected
	if lst[0] == "" {
		lst[0] = string(filepath.Separator)
	}

	// reuse path string
	path = ""

	for _, l := range lst {
		path = filepath.Join(path, l)

		if afs.zf != nil {
			p := filepath.Join(afs.inZipPath, l)

			zf, err := afs.zf.Open(p)
			if err != nil {
				return fmt.Errorf("archivefs: set: %v", err)
			}

			zfi, err := zf.Stat()
			if err != nil {
				return fmt.Errorf("archivefs: set: %v", err)
			}

			afs.isDir = zfi.IsDir()
			if afs.isDir {
				afs.inZipPath = p
				afs.inZipFile = ""
			} else {
				afs.inZipFile = l
			}

		} else {
			fi, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("archivefs: set: %v", err)
			}

			afs.isDir = fi.IsDir()
			if afs.isDir {
				continue
			}

			afs.zf, err = zip.OpenReader(path)
			if err == nil {
				// the root of an archive file is considered to be a directory
				afs.isDir = true
				continue
			}

			if !errors.Is(err, zip.ErrFormat) {
				return fmt.Errorf("archivefs: set: %v", err)
			}
		}
	}

	// make sure path is clean
	afs.current = filepath.Clean(path)

	return nil
}
