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

//go:build !wasm

package fs

import (
	"os"
)

// MkdirAll is an abstraction of os.MkdirAll().
func MkdirAll(pth string, perm os.FileMode) error {
	if err := os.MkdirAll(pth, perm); err != nil {
		return err
	}
	return nil
}

// File is an abstraction of os.File.
type File struct {
	f *os.File
}

// Close is an abstraction of os.File.Close().
func (f *File) Close() error {
	return f.f.Close()
}

// Close is an abstraction of os.File.Read().
func (f *File) Read(p []byte) (n int, err error) {
	return f.f.Read(p)
}

// Close is an abstraction of os.File.Write().
func (f *File) Write(p []byte) (n int, err error) {
	return f.f.Write(p)
}

// File is an abstraction of os.Open().
func Open(pth string) (*File, error) {
	var err error
	f := &File{}
	f.f, err = os.Open(pth)
	return f, err
}

// File is an abstraction of os.Create().
func Create(pth string) (*File, error) {
	var err error
	f := &File{}
	f.f, err = os.Create(pth)
	return f, err
}
