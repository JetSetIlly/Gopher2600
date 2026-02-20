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

package script

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Line struct {
	Entry string
	Batch bool
}

// Queue normalises input into commands and dishes out those commands one at a time. Used by
// interactive terminals and scripts.
type Queue struct {
	lines []Line
}

// More returns true if there are more commands in the queue
func (q *Queue) More() bool {
	return len(q.lines) > 0
}

// Next command in the queue
func (q *Queue) Next() (Line, bool) {
	if len(q.lines) > 0 {
		ln := q.lines[0]
		q.lines = q.lines[1:]
		return ln, true
	}
	return Line{}, false
}

// Push input line into queue. Input is normalised before the first command is returned
func (q *Queue) Push(input string) (Line, error) {
	q.push(input, false)
	if ln, ok := q.Next(); ok {
		return ln, nil
	}
	return Line{}, io.EOF
}

func (q *Queue) push(input string, batch bool) {
	// replace windows and mac line endings with unix line endings
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")

	// commands can be separated by semi-colons as well as newlines.
	// normalise semi-colons with newlines
	input = strings.ReplaceAll(input, ";", "\n")

	// loop through lines
	for s := range strings.SplitSeq(input, "\n") {
		if len(s) > 0 && !strings.HasPrefix(s, "#") {
			ln := Line{Entry: s, Batch: batch}
			q.lines = append(q.lines, ln)
		}
	}
}

// Load script into queue
func (q *Queue) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("script: no such file: %s", filename)
		}
		return fmt.Errorf("script: %w", err)
	}
	defer f.Close()

	s, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("script: %w", err)
	}

	q.push(string(s), true)

	return nil
}
