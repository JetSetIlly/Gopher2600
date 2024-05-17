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

package archivefs_test

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/jetsetilly/gopher2600/archivefs"
	"github.com/jetsetilly/gopher2600/test"
)

func TestArchivefsPath(t *testing.T) {
	var afs archivefs.Path
	var path string
	var entries []archivefs.Entry
	var err error

	// non-existant file
	path = "foo"
	err = afs.Set(path, false)
	test.ExpectFailure(t, err)
	test.ExpectEquality(t, afs.String(), "")

	// a real directory
	path = "testdir"
	err = afs.Set(path, false)
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, afs.String(), path)
	test.ExpectSuccess(t, afs.IsDir())
	test.ExpectSuccess(t, !afs.InArchive())

	// entries in a directory
	entries, err = afs.List()
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, len(entries), 2)
	test.ExpectEquality(t, fmt.Sprintf("%s", entries), "[testarchive.zip testfile]")

	// non-existant file in directory
	path = filepath.Join("testdir", "foo")
	err = afs.Set(path, false)
	test.ExpectFailure(t, err)
	test.ExpectEquality(t, afs.String(), "")

	// a real file in directory
	path = filepath.Join("testdir", "testfile")
	err = afs.Set(path, false)
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, afs.String(), path)
	test.ExpectSuccess(t, !afs.IsDir())
	test.ExpectSuccess(t, !afs.InArchive())

	// calling List() when path is set to a file type (ie not a direcotry) the
	// list returned should be of the containing directory
	entries, err = afs.List()
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, len(entries), 2)
	test.ExpectEquality(t, fmt.Sprintf("%s", entries), "[testarchive.zip testfile]")

	// a real archive
	path = filepath.Join("testdir", "testarchive.zip")
	err = afs.Set(path, false)
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, afs.String(), path)
	test.ExpectSuccess(t, afs.IsDir())
	test.ExpectSuccess(t, afs.InArchive())

	// entries in an archive
	entries, err = afs.List()
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, len(entries), 3)
	test.ExpectEquality(t, fmt.Sprintf("%s", entries), "[archivedir archivefile1 archivefile2]")

	// file in a real archive
	path = filepath.Join("testdir", "testarchive.zip", "archivefile1")
	err = afs.Set(path, false)
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, afs.String(), path)
	test.ExpectSuccess(t, !afs.IsDir())
	test.ExpectSuccess(t, afs.InArchive())

	// directory a real archive
	path = filepath.Join("testdir", "testarchive.zip", "archivedir")
	err = afs.Set(path, false)
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, afs.String(), path)
	test.ExpectSuccess(t, afs.IsDir())
	test.ExpectSuccess(t, afs.InArchive())

	// entries in an archive
	entries, err = afs.List()
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, len(entries), 2)
	test.ExpectEquality(t, fmt.Sprintf("%s", entries), "[archivedir2 archivefile3]")

	// file in a real archive
	path = filepath.Join("testdir", "testarchive.zip", "archivedir", "archivefile3")
	err = afs.Set(path, false)
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, afs.String(), path)
	test.ExpectSuccess(t, !afs.IsDir())
	test.ExpectSuccess(t, afs.InArchive())
}

func TestArchivefsOpen(t *testing.T) {
	r, sz, err := archivefs.Open(filepath.Join("testdir", "testarchive.zip", "archivefile1"))
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, sz, 22)
	d, err := io.ReadAll(r)
	test.ExpectSuccess(t, err)
	test.ExpectEquality(t, string(d), "archivefile1 contents\n")
}
