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

// Package notifications defines the Notify type and the possible values of
// that type. These values represent the different notifications that be sent
// to the GUI.
//
//	hardware  ---->  emulation  ---->  GUI
//	(eg. cartridge)  (eg. debugger)
//
// Notifications flow in one direction only and can be generated and terminate
// at any of the points in the chart above.
//
// For example, a pause PlusROM network activitiy notification will be
// generated in the hardware, passed to the "emulation" package and forwarded
// to the GUI.
//
// Another example, is the rewind notification. This will be generated in the
// "emulation" package and sent to the GUI.
//
// Finally, a mute notification will be generated and consumed entirely inside
// the GUI.
//
// In some instances, the emulation may choose not to forward the notification
// to the GUI or to transform it into some other notification but these
// instances should be rare.
//
// Loosely related to the Notify type is the gui.FeatureRequest. The GUI
// FeatureRequest system is the mechanism by which notifications are forwarded
// to the GUI.
//
// Communication between hardware and the emulation is meanwhile is handled by
// the NotificationHook mechanism defined in this package.
package notifications
