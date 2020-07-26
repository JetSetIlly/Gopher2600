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

// Package paths contains functions to prepare paths for gopher2600 resources.
//
// The ResourcePath() function returns the correct path to the resource
// directory/file specified in the arguments. The result of ResourcePath
// depends on the build tag used to compile the program.
//
// For "release" tagged builds, the correct path is one rooted in the user's
// configuration directory. On modern Linux systems the full path would be
// something like:
//
//	/home/user/.config/gopher2600/
//
// For "non-release" tagged builds, the correct path is rooted in the current
// working directory:
//
//	.gopher2600
//
// The reason for this is simple. During development, it is more convenient to
// have the config directory close to hand. For release binaries meanwhile, the
// config directory should be somewhere the user expects.
package paths
