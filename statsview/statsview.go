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

//go:build statsview
// +build statsview

package statsview

import (
	"fmt"
	"io"

	"github.com/go-echarts/statsview"
	"github.com/go-echarts/statsview/viewer"
)

const Address = "localhost:12600"
const url = "/debug/statsview"

// Launch a new goroutine running the statsview.
func Launch(output io.Writer) {
	go func() {
		viewer.SetConfiguration(viewer.WithAddr(Address))
		mgr := statsview.New()
		mgr.Start()
	}()

	output.Write([]byte(fmt.Sprintf("stats server available at %s%s\n", Address, url)))
}

// Available returns true if a statsview is available to launch.
func Available() bool {
	return true
}
