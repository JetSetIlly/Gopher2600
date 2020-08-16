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
	"os"
	"strings"
)

type Entry struct {
	tag      string
	detail   string
	repeated int
}

func (e *Entry) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s: %s", e.tag, e.detail))
	if e.repeated > 0 {
		s.WriteString(fmt.Sprintf(" (repeat x%d)", e.repeated+1))
	}
	s.WriteString("\n")
	return s.String()
}

type logger struct {
	entries []Entry
	echo    bool
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
	e := Entry{tag: tag, detail: detail}
	if central.echo {
		io.WriteString(os.Stdout, e.String())
	}

	if len(central.entries) == 0 ||
		(e.detail != central.entries[len(central.entries)-1].detail ||
			e.tag != central.entries[len(central.entries)-1].tag) {
		central.entries = append(central.entries, e)
	} else {
		central.entries[len(central.entries)-1].repeated++
	}
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
		io.WriteString(output, e.String())
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
		io.WriteString(output, e.String())
	}
}

// SetEcho to print new entries to os.Stdout
func SetEcho(echo bool) {
	central.echo = echo
}
