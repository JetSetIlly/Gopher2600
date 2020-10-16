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

package controllers

// ControllerList is the list of controllers. These are the values that can be
// returned by the ID() function of the ports.Peripheral implementations in
// this package.
var ControllerList = []string{"Stick", "Paddle", "Keyboard"}

// Sentinal error returned if controller implementation does not understand
// event sent to HandleEvent().
const (
	UnhandledEvent = "unhandled event: %s does not support event %v"
)
