//go:build imguifreetype

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

package fonts

import _ "embed"

//go:embed "JetBrainsMono-Regular.ttf"
var JetBrainsMono []byte

// The first and last unicode points used in the application. We use this to
// make sure we're using as small a font texture as possible.
const (
	JetBrainsMonoMin = '\u0003'
	JetBrainsMonoMax = '\u00ff'
)
