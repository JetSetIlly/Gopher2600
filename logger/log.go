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
)

// Entry represents a single line/entry in the log.
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

const allEntries = -1

// not exposing logger to outside of the package. the package level functions
// can be used to log to the central logger.
type logger struct {
	// add a new Entry
	add chan Entry

	// get and del return the requested number of entries counting from the
	// most recent. to specify all entries use the allEntries constant
	get    chan int
	del    chan int
	recent chan bool

	// array of Entries from the service goroutine
	entries chan []Entry

	// if echo is not nil than write new entry to the io.Writer
	echo io.Writer

	// the index of the last entry sent over the recent channel
	lastRecent int
}

func newLogger(maxEntries int) *logger {
	l := &logger{
		add:     make(chan Entry),
		get:     make(chan int),
		del:     make(chan int),
		recent:  make(chan bool),
		entries: make(chan []Entry),
	}

	// the loggger service gorountine is simple enough to inline and still
	// retain clarity
	go func() {
		entries := make([]Entry, 0, maxEntries)

		for {
			select {
			case e := <-l.add:
				last := Entry{}
				if len(entries) > 0 {
					last = entries[len(entries)-1]
				}

				if last.tag == e.tag && last.detail == e.detail {
					entries[len(entries)-1].repeated++
				} else {
					entries = append(entries, e)
				}

				if len(entries) > maxEntries {
					l.lastRecent -= maxEntries - len(entries)
					if l.lastRecent < 0 {
						l.lastRecent = 0
					}
					entries = entries[len(entries)-maxEntries:]
				}

			case n := <-l.get:
				if n < 0 || n > len(entries) {
					n = len(entries)
				}
				l.entries <- entries[len(entries)-n:]

			case v := <-l.recent:
				if v {
					l.entries <- entries[l.lastRecent:]
					l.lastRecent = len(entries)
				}

			case n := <-l.del:
				if n < 0 || n > len(entries) {
					n = len(entries)
				}
				entries = entries[:len(entries)-n]
				l.lastRecent = 0
			}
		}
	}()

	return l
}

func (l *logger) log(tag, detail string) {
	// remove first part of the details string if it's the same as the tag
	p := strings.SplitN(detail, ": ", 3)
	if len(p) > 1 && p[0] == tag {
		detail = strings.Join(p[1:], ": ")
	}

	e := Entry{tag: tag, detail: detail}
	l.add <- e
	if l.echo != nil {
		l.echo.Write([]byte(e.String()))
	}
}

func (l *logger) clear() {
	l.del <- allEntries
}

func (l *logger) write(output io.Writer) {
	l.get <- allEntries
	entries := <-l.entries

	for _, e := range entries {
		io.WriteString(output, e.String())
	}
}

func (l *logger) writeRecent(output io.Writer) {
	l.recent <- true
	entries := <-l.entries

	for _, e := range entries {
		io.WriteString(output, e.String())
	}
}

func (l *logger) tail(output io.Writer, number int) {
	l.get <- number
	entries := <-l.entries

	for _, e := range entries {
		io.WriteString(output, e.String())
	}
}

func (l *logger) copy() []Entry {
	l.get <- allEntries
	return <-l.entries
}

func (l *logger) setEcho(output io.Writer) {
	l.echo = output
}
