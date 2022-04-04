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

// Package thumbnailer can be used to create either a series of thumbnail
// images or a single thumbnail image with the CreateFromLoader() and
// SingleFrameFromRewindState() functions respsectively.
//
// The CreateFromLodaer() function will run asynchronously and is good for
// generating just the images from a new emulation.
//
// The SingleFrameFromRewindState() function meanwhile, is more limited and is
// used to generate a single TV frame starting from the supplied rewind state.
package thumbnailer
