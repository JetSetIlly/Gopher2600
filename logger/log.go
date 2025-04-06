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
	"time"
)

// Entry represents a single line/entry in the log
type Entry struct {
	Time     time.Time
	Tag      string
	Detail   string
	Repeated int
}

func (e *Entry) String() string {
	s := strings.Builder{}
	if len(e.Tag) == 0 {
		s.WriteString(e.Detail)
	} else {
		s.WriteString(fmt.Sprintf("%s: %s", e.Tag, e.Detail))
	}
	if e.Repeated > 0 {
		s.WriteString(fmt.Sprintf(" (repeat x%d)", e.Repeated+1))
	}
	return s.String()
}

// not exposing Logger to outside of the package. the package level functions
// can be used to log to the central Logger
type Logger struct {
	crit sync.Mutex

	entries    []Entry
	maxEntries int

	// the index of newest entry not seen by the writeRecent() function
	recentStart int

	// if echo is not nil than write new entry to the io.Writer
	echo io.Writer
}

// NewLogger is the preferred method of initialisation for the Logger type
func NewLogger(maxEntries int) *Logger {
	return &Logger{
		entries:    make([]Entry, 0, maxEntries),
		maxEntries: maxEntries,
	}
}

// the boolean value indicates that the detail type is 'supported'. false is
// returned if the type is not supported and the string value has been formatted
// with the %v verb from the fmt package
func detailConversion(detail any) (string, bool) {
	switch d := detail.(type) {
	case string:
		return d, true
	case error:
		return d.Error(), true
	case fmt.Stringer:
		return d.String(), true
	}
	return fmt.Sprintf("%v\n", detail), false
}

// Log adds a new entry to the logger. The detail string will be interpreted as
// either a string, an error type or a fmt.Stringer type
//
// In the case of being an error type, the detail string will be taken from the
// Error() function
//
// Detail arguments of an unsupported type will be formatted using the %v verb
// from the fmt package. ie. fmt.Sprintf("%v", detail)
func (l *Logger) Log(perm Permission, tag string, detail any) {
	if !(perm == Allow || perm.AllowLogging()) {
		return
	}

	detailConverted, supported := detailConversion(detail)
	if supported {
		p := strings.SplitN(detailConverted, ": ", 3)
		// remove first part of the details string if it's the same as the tag
		if len(p) > 1 && p[0] == tag {
			detailConverted = strings.Join(p[1:], ": ")
		}
	}

	tag = strings.TrimSpace(tag)

	l.crit.Lock()
	defer l.crit.Unlock()

	last := Entry{}
	if len(l.entries) > 0 {
		last = l.entries[len(l.entries)-1]
	}

	// time of logging
	now := time.Now()

	// split multi-line log entries and log each separetely
	for _, d := range strings.Split(detailConverted, "\n") {
		d = strings.TrimSpace(d)
		if len(d) == 0 {
			continue
		}

		e := Entry{
			Time:   now,
			Tag:    tag,
			Detail: d,
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
}

// Logf adds a new entry to the logger. The detail string is interpreted as a
// formatting string as described by the fmt package
func (l *Logger) Logf(perm Permission, tag, detail string, args ...any) {
	l.Log(perm, tag, fmt.Sprintf(detail, args...))
}

// Clear all entries from logger
func (l *Logger) Clear() {
	l.crit.Lock()
	defer l.crit.Unlock()

	l.entries = l.entries[:0]
	l.recentStart = 0
}

// Write contents of central logger to io.Writer
func (l *Logger) Write(output io.Writer) {
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

// WriteRecent returns only the entries added since the last call to CopyRecent
func (l *Logger) WriteRecent(output io.Writer) {
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

// Tail writes the last N entries in the central logger to io.Writer. A number
// parameter of <0 will output the entire log.
func (l *Logger) Tail(output io.Writer, number int) {
	if output == nil {
		return
	}

	l.crit.Lock()
	defer l.crit.Unlock()

	var n int
	if number < 0 {
		n = 0
	} else {
		n = max(len(l.entries)-number, 0)
	}

	for _, e := range l.entries[n:] {
		io.WriteString(output, e.String())
		io.WriteString(output, "\n")
	}
}

// SetEcho prints entries to io.Writer as and when they are added
func (l *Logger) SetEcho(output io.Writer, writeRecent bool) {
	l.crit.Lock()
	l.echo = output
	l.crit.Unlock()

	if writeRecent {
		l.WriteRecent(output)
	}
}

// BorrowLog gives the provided function the critial section and access to the
// list of log entries
func (l *Logger) BorrowLog(f func([]Entry)) {
	l.crit.Lock()
	defer l.crit.Unlock()

	f(l.entries)
}
