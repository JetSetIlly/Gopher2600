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
// allow more than one log.
var central *logger

// maximum number of entries in the central logger.
const maxCentral = 256

func init() {
	central = newLogger(maxCentral)
}

// Log adds an entry to the central logger.
func Log(tag, detail string) {
	central.log(tag, detail)
}

// Clear all entries from central logger.
func Clear() {
	central.clear()
}

// Write contents of central logger to io.Writer.
func Write(output io.Writer) {
	central.write(output)
}

// WriteRecent returns only the entries added since the last call to CopyRecent.
func WriteRecent(output io.Writer) {
	central.writeRecent(output)
}

// Tail writes the last N entries to io.Writer.
func Tail(output io.Writer, number int) {
	central.tail(output, number)
}

// Slice returns a copy all log entries.
func Copy() []Entry {
	return central.copy()
}

// SetEcho to print new entries to os.Stdout.
func SetEcho(output io.Writer) {
	central.setEcho(output)
}
