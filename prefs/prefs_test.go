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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package prefs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/test"
)

const tempFile = "gopher2600_prefs_test"

func getTmpPrefFile(t *testing.T) string {
	t.Helper()

	// get info about the temp directory
	td := os.TempDir()
	info, err := os.Stat(td)
	if err != nil {
		t.Errorf("error accessing tmp file: %v", err)
	}

	// check read/write permission for user
	p := info.Mode().Perm()
	if p&0o0700 != 0o0700 {
		t.Errorf("error accessing tmp file: %v", err)
	}

	// return temporary file
	fn := path.Join(td, tempFile)
	return fn
}

func delTmpPrefFile(t *testing.T, fn string) {
	t.Helper()

	if err := os.Remove(fn); err != nil {
		// not worrying about path errors; these are returned if
		// file doesn't exist
		if _, ok := err.(*os.PathError); !ok {
			t.Errorf("error removing tmp pref file: %v", err)
		}
	}
}

func cmpTmpFile(t *testing.T, fn string, expected string) {
	t.Helper()

	f, err := os.Open(fn)
	if err != nil {
		t.Errorf("error opening tmp file: %v", err)
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Errorf("error reading tmp file: %v", err)
		return
	}

	expected = fmt.Sprintf("%s\n%s", prefs.WarningBoilerPlate, expected)

	if expected != string(data) {
		t.Errorf("expected data and data in prefs file do not match")
		fmt.Println("expected:")
		fmt.Println(expected)
		fmt.Println("\nin file:")
		fmt.Println(string(data))
	}
}

func TestBool(t *testing.T) {
	fn := getTmpPrefFile(t)
	defer delTmpPrefFile(t, fn)

	dsk, err := prefs.NewDisk(fn)
	if err != nil {
		t.Errorf("error preparing disk: %v", err)
		return
	}

	var v prefs.Bool
	var w prefs.Bool
	var x prefs.Bool
	dsk.Add("test", &v)
	dsk.Add("testB", &w)
	dsk.Add("testC", &x)

	v.Set(true)
	w.Set("foo")
	x.Set("true")

	err = dsk.Save()
	if err != nil {
		t.Errorf("error saving disk: %v", err)
		return
	}

	cmpTmpFile(t, fn, "test :: true\ntestB :: false\ntestC :: true\n")
}

func TestString(t *testing.T) {
	fn := getTmpPrefFile(t)
	defer delTmpPrefFile(t, fn)

	dsk, err := prefs.NewDisk(fn)
	if err != nil {
		t.Errorf("error preparing disk: %v", err)
		return
	}

	var v prefs.String
	dsk.Add("foo", &v)

	v.Set("bar")

	err = dsk.Save()
	if err != nil {
		t.Errorf("error saving disk: %v", err)
	}

	cmpTmpFile(t, fn, "foo :: bar\n")
}

func TestInt(t *testing.T) {
	fn := getTmpPrefFile(t)
	defer delTmpPrefFile(t, fn)

	dsk, err := prefs.NewDisk(fn)
	if err != nil {
		t.Errorf("error preparing disk: %v", err)
		return
	}

	var v prefs.Int
	var w prefs.Int
	dsk.Add("number", &v)
	dsk.Add("numberB", &w)

	err = v.Set(10)
	test.ExpectedSuccess(t, err)

	// test string conversion to int
	err = w.Set("99")
	test.ExpectedSuccess(t, err)

	err = dsk.Save()
	if err != nil {
		t.Errorf("error saving disk: %v", err)
	}

	cmpTmpFile(t, fn, "number :: 10\nnumberB :: 99\n")

	// while we have a prefs.Int instance set up we'll test some
	// failure conditions
	err = v.Set("---")
	test.ExpectedFailure(t, err)

	err = v.Set(1.0)
	test.ExpectedFailure(t, err)
}

func TestGeneric(t *testing.T) {
	fn := getTmpPrefFile(t)
	defer delTmpPrefFile(t, fn)

	dsk, err := prefs.NewDisk(fn)
	if err != nil {
		t.Errorf("error preparing disk: %v", err)
		return
	}

	var w, h int

	v := prefs.NewGeneric(
		func(s string) error {
			_, err := fmt.Sscanf(s, "%d,%d", &w, &h)
			if err != nil {
				return err
			}
			return nil
		},
		func() string {
			return fmt.Sprintf("%d,%d", w, h)
		},
	)

	dsk.Add("generic", v)

	// change values
	w = 1
	h = 2

	// save to disk
	err = dsk.Save()
	if err != nil {
		t.Errorf("error saving disk: %v", err)
	}

	cmpTmpFile(t, fn, "generic :: 1,2\n")

	// reset values
	w = 0
	h = 0

	// reload them from disk
	err = dsk.Load()
	if err != nil {
		t.Errorf("error saving disk: %v", err)
	}

	// check that the values have been restoed
	test.Equate(t, w, 1)
	test.Equate(t, h, 2)
}

// write bool and then a string from a different prefs.Disk instance. tests
// that the second writing doesn't clobber the results of the first write.
func TestBoolAndString(t *testing.T) {
	fn := getTmpPrefFile(t)
	defer delTmpPrefFile(t, fn)

	dsk, err := prefs.NewDisk(fn)
	if err != nil {
		t.Errorf("error preparing disk: %v", err)
		return
	}

	var v prefs.Bool
	dsk.Add("test", &v)

	v.Set(true)

	err = dsk.Save()
	if err != nil {
		t.Errorf("error saving disk: %v", err)
		return
	}

	// start a new disk instance using the same file. (we haven't deleted it yet)
	dsk, err = prefs.NewDisk(fn)
	if err != nil {
		t.Errorf("error preparing disk: %v", err)
		return
	}

	var s prefs.String
	dsk.Add("foo", &s)

	s.Set("bar")

	err = dsk.Save()
	if err != nil {
		t.Errorf("error saving disk: %v", err)
		return
	}

	// compate file. the file should contain contents set by both disk
	// instances
	cmpTmpFile(t, fn, "foo :: bar\ntest :: true\n")
}
