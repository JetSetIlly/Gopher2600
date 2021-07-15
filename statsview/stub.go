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

//go:build !statsview
// +build !statsview

package statsview

// https://github.com/go-echarts/statsview

import (
	"io"
)

const Address = "no statsview"

// Launch a new goroutine running the statsview.
func Launch(output io.Writer) {
	output.Write([]byte(Address))
}

// Available returns true if a statsview is available to launch.
func Available() bool {
	return false
}
