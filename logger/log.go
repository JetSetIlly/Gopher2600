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
	"strings"
	"sync"
)

// Entry represents a single line/entry in the log.
type Entry struct {
	Tag      string
	Detail   string
	Repeated int
}

func (e *Entry) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s: %s", e.Tag, e.Detail))
	if e.Repeated > 0 {
		s.WriteString(fmt.Sprintf(" (repeat x%d)", e.Repeated+1))
	}
	return s.String()
}

// not exposing logger to outside of the package. the package level functions
// can be used to log to the central logger.
type logger struct {
	crit sync.Mutex

	entries    []Entry
	maxEntries int

	// the index of newest entry not seen by the writeRecent() function
	recentStart int

	// if echo is not nil than write new entry to the io.Writer
	echo io.Writer
}

func newLogger(maxEntries int) *logger {
	return &logger{
		entries:    make([]Entry, 0, maxEntries),
		maxEntries: maxEntries,
	}
}

func (l *logger) log(tag, detail string) {
	l.crit.Lock()
	defer l.crit.Unlock()

	// remove first part of the details string if it's the same as the tag
	p := strings.SplitN(detail, ": ", 3)
	if len(p) > 1 && p[0] == tag {
		detail = strings.Join(p[1:], ": ")
	}

	last := Entry{}
	if len(l.entries) > 0 {
		last = l.entries[len(l.entries)-1]
	}

	e := Entry{
		Tag:    tag,
		Detail: detail,
	}

	if last.Tag == e.Tag && last.Detail == e.Detail {
		l.entries[len(l.entries)-1].Repeated++
		return
	} else {
		l.entries = append(l.entries, e)
	}

	if len(l.entries) > l.maxEntries {
		l.recentStart -= l.maxEntries - len(l.entries)
		if l.recentStart < 0 {
			l.recentStart = 0
		}
	}

	if l.echo != nil {
		l.echo.Write([]byte(e.String()))
		l.echo.Write([]byte("\n"))
	}
}

func (l *logger) logf(tag, detail string, args ...interface{}) {
	l.log(tag, fmt.Sprintf(detail, args...))
}

func (l *logger) clear() {
	l.crit.Lock()
	defer l.crit.Unlock()

	l.entries = l.entries[:0]
	l.recentStart = 0
}

func (l *logger) write(output io.Writer) {
	if output == nil {
		return
	}

	l.crit.Lock()
	defer l.crit.Unlock()

	for _, e := range l.entries {
		io.WriteString(output, e.String())
		io.WriteString(output, "\n")
	}
}

func (l *logger) writeRecent(output io.Writer) {
	l.crit.Lock()
	defer l.crit.Unlock()

	if output != nil {
		for _, e := range l.entries[l.recentStart:] {
			io.WriteString(output, e.String())
			io.WriteString(output, "\n")
		}
	}

	l.recentStart = len(l.entries)
}

func (l *logger) tail(output io.Writer, number int) {
	if output == nil {
		return
	}

	l.crit.Lock()
	defer l.crit.Unlock()

	var n int

	n = len(l.entries) - number
	if n < 0 {
		n = 0
	}

	for _, e := range l.entries[n:] {
		io.WriteString(output, e.String())
		io.WriteString(output, "\n")
	}
}

func (l *logger) setEcho(output io.Writer, writeRecent bool) {
	l.crit.Lock()
	l.echo = output
	l.crit.Unlock()

	if writeRecent {
		l.writeRecent(output)
	}
}

func (l *logger) borrowLog(f func([]Entry)) {
	l.crit.Lock()
	defer l.crit.Unlock()

	f(l.entries)
}
