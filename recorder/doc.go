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

// Package recorder handles recording and playback of user input. The Recorder
// type implements the riot.input.EventRecorder() interface. Once added as a
// transcriber to the VCS port, it will record all user input to the specified
// file.
//
// To keep things simple, recording gameplay will use the VCS in it's default
// state. Future versions of the recorder fileformat will support localised
// preferences.
package recorder
