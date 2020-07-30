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
	"fmt"
	"io"
)

type Entry struct {
	tag    string
	detail string
}

type logger struct {
	entries []Entry
}

func newLogger() *logger {
	return &logger{
		entries: make([]Entry, 0),
	}
}

// only allowing one central log for the entire application. there's no need to
// allow more than one log
var central *logger

func init() {
	central = newLogger()
}

// Log adds an entry to the central logger
func Log(tag, detail string) {
	central.entries = append(central.entries, Entry{tag: tag, detail: detail})
}

// Clear all entries from central logger
func Clear() {
	central.entries = central.entries[:0]
}

// Write contents of central logger to io.Writer
func Write(output io.Writer) bool {
	if len(central.entries) == 0 {
		return false
	}
	for _, e := range central.entries {
		io.WriteString(output,
			fmt.Sprintf("%s: %s\n", e.tag, e.detail))
	}
	return true
}

// Write the last N entries to io.Writer
func Tail(output io.Writer, number int) {
	// cap number to the number of entries
	if number > len(central.entries) {
		number = len(central.entries)
	}

	for _, e := range central.entries[len(central.entries)-number:] {
		io.WriteString(output,
			fmt.Sprintf("%s: %s\n", e.tag, e.detail))
	}
}
