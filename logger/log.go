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
	"sync/atomic"
	"time"
)

// Entry represents a single line/entry in the log
type Entry struct {
	Timestamp time.Time
	tag       string
	detail    string
	repeated  int
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

// not exposing logger to outside of the package. the package level functions
// can be used to log to the central logger.
type logger struct {
	maxEntries int
	entries    []Entry
	echo       bool

	// timestamp of most recent log() event
	atomicTimestamp atomic.Value // time.Time
}

func newLogger(maxEntries int) *logger {
	return &logger{
		maxEntries: maxEntries,
		entries:    make([]Entry, 0),
	}
}

func (l *logger) log(tag, detail string) {
	e := &Entry{}
	if len(l.entries) > 0 {
		e = &l.entries[len(l.entries)-1]
	}

	// remove all newline characters from tag and detail string
	tag = strings.ReplaceAll(tag, "\n", "")
	detail = strings.ReplaceAll(detail, "\n", "")

	if detail != e.detail || tag != e.tag {
		l.entries = append(l.entries, Entry{Timestamp: time.Now(), tag: tag, detail: detail})
	} else {
		e.repeated++
		e.Timestamp = time.Now()
	}

	// store atomic timestamp
	l.atomicTimestamp.Store(e.Timestamp)

	// mainain maximum length
	if len(l.entries) > l.maxEntries {
		l.entries = l.entries[len(l.entries)-maxCentral:]
	}

	if l.echo {
		io.WriteString(os.Stdout, e.String())
	}
}

func (l *logger) clear() {
	l.entries = l.entries[:0]
}

func (l *logger) write(output io.Writer) bool {
	if len(l.entries) == 0 {
		return false
	}
	for _, e := range l.entries {
		io.WriteString(output, e.String())
	}
	return true
}

func (l *logger) tail(output io.Writer, number int) {
	// cap number to the number of entries
	if number > len(l.entries) {
		number = len(l.entries)
	}

	for _, e := range l.entries[len(l.entries)-number:] {
		io.WriteString(output, e.String())
	}
}

func (l *logger) copy(ref time.Time) []Entry {
	if ref != l.atomicTimestamp.Load().(time.Time) {
		c := make([]Entry, len(l.entries))
		copy(c, l.entries)
		return c
	}
	return nil
}
