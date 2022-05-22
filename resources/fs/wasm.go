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

//go:build wasm
// +build wasm

package fs

import (
	"os"
)

// MkdirAll is a stub function.
func MkdirAll(pth string, perm os.FileMode) error {
	return nil
}

// File is a stub type.
type File struct{}

// Close is a stub function.
func (f *File) Close() error {
	return nil
}

// Read is a stub function.
func (f *File) Read(p []byte) (n int, err error) {
	return 0, nil
}

// Write is a stub function.
func (f *File) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// Open is a stub function.
func Open(pth string) (*File, error) {
	return &File{}, nil
}

// Create is a stub function.
func Create(pth string) (*File, error) {
	return &File{}, nil
}

// Abs is an abstraction of filepath.Abs().
func Abs(pth string) (string, error) {
	return pth, nil
}
