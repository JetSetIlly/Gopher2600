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

package colorterm

// this part of the colorterm package facilitates the reading of runes from a
// reader interface (eg, os.Stdin). a straight call to reader.ReadRune() will
// block until input is available. this causes problems if we want the program
// to wait for something else from another reader or channel. with the mechanism
// here, an input loop can use a channel-select to prevent the blocking.

import (
	"bufio"
	"io"
)

type readRune struct {
	r   rune
	n   int
	err error
}

type runeReader chan readRune

func initRuneReader(reader io.Reader) runeReader {
	bufReader := bufio.NewReader(reader)
	ch := make(runeReader)
	go func() {
		var readRune readRune
		for {
			readRune.r, readRune.n, readRune.err = bufReader.ReadRune()
			ch <- readRune
		}
	}()

	return ch
}
