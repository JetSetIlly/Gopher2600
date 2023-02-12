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

package unique

import (
	"fmt"
	"strings"
	"time"
)

// Filename creates a filename that (assuming a functioning clock) should not
// collide with any existing file. Note that the function does not test for
// existing files.
//
// Format of returned string is:
//
//	filetype_cartname_YYYYMMDD_HHMMSS
//
// Where cartname is the string returned by cartload.ShortName(). If the
// cartname argument is empty the returned string will be of the format:
//
//	filetype_YYYYMMDD_HHMMSS
//
// The filetype argument is simply another way of identifying the file
// uniquely. For example, if saving a screenshot the filetype might simply be
// "screenshot" or "photo".
//
// If the filetype argument is empty the returned string will be of the format:
//
//	cartname_YYYYMMDD_HHMMSS
//
// If both filetype and cartname arguments are empty then the returned string
// will be the timestamp only.
//
// Note that there is no provision for adding a file extension. If you need one
// you must append that manually.
func Filename(filetype string, cartname string) string {
	n := time.Now()
	timestamp := fmt.Sprintf("%04d%02d%02d_%02d%02d%02d", n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second())

	filetype = strings.TrimSpace(filetype)
	cartname = strings.TrimSpace(cartname)

	s := strings.Builder{}
	if len(filetype) > 0 {
		s.WriteString(filetype)
		s.WriteString("_")
	}
	if len(cartname) > 0 {
		s.WriteString(cartname)
		s.WriteString("_")
	}
	s.WriteString(timestamp)

	return s.String()
}
