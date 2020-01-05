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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// Package paths contains functions to prepare paths to gopher2600 resources.
//
// The ResourcePath() function modifies the supplied resource string such that
// it is prepended with the appropriate config directory. For example, the
// following will return the path to a cartridge patch.
//
//	d := paths.ResourcePath("patches", "ET")
//
// The policy of ResourcePath() is simple: if the base resource path, currently
// defined to be ".gopher2600", is present in the program's current directory
// then that is the base path that will used. If it is not preseent not, then
// the user's config directory is used. The package uses os.UserConfigDir()
// from go standard library for this.
//
// In the example above, on a modern Linux system, the path returned will be:
//
//	/home/user/.config/gopher2600/patches/ET
package paths
