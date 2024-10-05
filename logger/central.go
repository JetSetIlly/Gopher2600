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

package logger

import (
	"io"
)

// only allowing one central log for the entire application. there's no need to
// allow more than one log
var central *Logger

// maximum number of entries in the central logger
const maxCentral = 256

func init() {
	central = NewLogger(maxCentral)
}

// Log adds an entry to the central logger
func Log(perm Permission, tag string, detail any) {
	central.Log(perm, tag, detail)
}

// Logf adds a new entry to the central logger. The detail string is interpreted
// as a formatting string as described by the fmt package
func Logf(perm Permission, tag, detail string, args ...any) {
	central.Logf(perm, tag, detail, args...)
}

// Clear all entries from central logger
func Clear() {
	central.Clear()
}

// Write contents of central logger to io.Writer
func Write(output io.Writer) {
	central.Write(output)
}

// WriteRecent returns only the entries in the central logger added since the
// last call to WriteRecent
func WriteRecent(output io.Writer) {
	central.WriteRecent(output)
}

// Tail writes the last N entries in the central logger to io.Writer
func Tail(output io.Writer, number int) {
	central.Tail(output, number)
}

// SetEcho prints entries in the central logger to io.Writer as they are created
func SetEcho(output io.Writer, writeRecent bool) {
	central.SetEcho(output, writeRecent)
}

// BorrowLog gives the provided function the critial section and access to the
// list of log entries
func BorrowLog(f func([]Entry)) {
	central.BorrowLog(f)
}
