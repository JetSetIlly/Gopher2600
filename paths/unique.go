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

package paths

import (
	"fmt"
	"strings"
	"time"
)

// UniqueFilename creates a filename that (assuming a functioning clock) should
// not collide with any existing file. Note that the function does not test for
// this.
//
// Used to generate filenames for:
//	- playback recordings
//	- regression scripts
//	- terminal output (sdlimgui GUI)
//
// Format of returned string is:
//
//     prepend_cartname_YYYYMMDD_HHMMSS
//
// Where cartname is the string returned by cartload.ShortName(). If there is
// no cartridge name the returned string will be of the format:
//
//     prepend_YYYYMMDD_HHMMSS
func UniqueFilename(prepend string, shortCartName string) string {
	n := time.Now()
	timestamp := fmt.Sprintf("%04d%02d%02d_%02d%02d%02d", n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second())

	var fn string

	c := strings.TrimSpace(shortCartName)
	if len(c) > 0 {
		fn = fmt.Sprintf("%s_%s_%s", prepend, c, timestamp)
	} else {
		fn = fmt.Sprintf("%s_%s", prepend, timestamp)
	}

	return fn
}
