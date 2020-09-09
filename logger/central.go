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
	"time"
)

// only allowing one central log for the entire application. there's no need to
// allow more than one log
var central *logger

// maximum number of entries in the central logger
const maxCentral = 256

func init() {
	central = newLogger(maxCentral)
}

// Log adds an entry to the central logger
func Log(tag, detail string) {
	central.log(tag, detail)
}

// Clear all entries from central logger
func Clear() {
	central.clear()
}

// Write contents of central logger to io.Writer
func Write(output io.Writer) bool {
	return central.write(output)
}

// Tail writes the last N entries to io.Writer
func Tail(output io.Writer, number int) {
	central.tail(output, number)
}

// Slice returns a copy of the last n entries. the ref argument is the
// timestamp of the last entry of the last copy. A new copy will only be made
// if the timestamp of the current last entry is different to the ref value.
//
// The function will return nil if no new copy has been made. Callers should
// continue to use a previous copy of the log
func Copy(ref time.Time) []Entry {
	return central.copy(ref)
}

// SetEcho to print new entries to os.Stdout
func SetEcho(echo bool) {
	central.echo = echo
}
