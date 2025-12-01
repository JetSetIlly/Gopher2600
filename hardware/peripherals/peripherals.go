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

package peripherals

// AvailableLeftPlayer is the list of peripherals that can feasibly be plugged
// into the left player port.
//
// These are the values that can be returned by the ID() function of the
// ports.Peripheral implementations in this package.
//
// Note that SaveKey and AtariVox can both technically be inserted into the
// left player but to keep things simple (we don't want multiple savekeys) we
// don't encourage it.
var AvailableLeftPlayer = []string{"Stick", "Paddle", "Keypad", "Gamepad"}

// AvailableRightPlayer is the list of peripherals that can feasibly be plugged
// into the right player port.
//
// These are the values that can be returned by the ID() function of the
// ports.Peripheral implementations in this package.
var AvailableRightPlayer = []string{"Stick", "Paddle", "Keypad", "Gamepad", "SaveKey", "AtariVox"}

// AvailableKeyportari is the list of protocols implemented for keyportari
var AvailableKeyportari = []string{"None", "24char", "ASCII"}
