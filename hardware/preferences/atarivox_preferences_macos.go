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

//go:build darwin

package preferences

import "os"

func (p *AtariVoxPreferences) binary() string {
	// same as regular UNIX build for now
	candidates := []string{"/usr/local/bin/festival", "/usr/bin/festival"}

	for _, n := range candidates {
		_, err := os.Stat(n)
		if err == nil {
			return n
		}
	}

	return ""
}
