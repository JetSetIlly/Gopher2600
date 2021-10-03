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

// Package resources contains functions to prepare paths for gopher2600
// resources.
//
// The JoinPath() function returns the correct path to the resource
// directory/file specified in the arguments. It handles the creation of
// directories as required but does not otherwise touch or create files.
//
// JoinPath() handles the inclusion of the correct base path. The base path
// depends on how the binary was built.
//
// For builds with the "releas" build tag, the path returned by JoinPath() is
// rooted in the user's configuration directory. On modern Linux systems the
// full path would be something like:
//
//	/home/user/.config/gopher2600/
//
// For non-"release" builds, the correct path is rooted in the current working
// directory:
//
//	.gopher2600
//
// The package does this because during development it is more convenient to
// have the config directory close to hand. For release binaries however, the
// config directory should be somewhere the end-user expects.
package resources
