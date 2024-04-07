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

package notifications

// Notice describes events that somehow change the presentation of the
// emulation. These notifications can be used to present additional information
// to the user
type Notice string

// List of defined notifications.
const (
	// a screen shot is taking place
	NotifyScreenshot Notice = "NotifyScreenshot"

	// notifications sent when supercharger is loading from a sound file (eg. mp3 file)
	NotifySuperchargerSoundloadStarted Notice = "NotifySuperchargerSoundloadStarted"
	NotifySuperchargerSoundloadEnded   Notice = "NotifySuperchargerSoundloadEnded"
	NotifySuperchargerSoundloadRewind  Notice = "NotifySuperchargerSoundloadRewind"

	// if Supercharger is loading from a fastload binary then this event is
	// raised when the ROM requests the next block be loaded from the "tape
	NotifySuperchargerFastload Notice = "NotifySuperchargerFastload"

	// notifications sent by plusrom
	NotifyPlusROMNewInstall Notice = "NotifyPlusROMNewInstall"
	NotifyPlusROMNetwork    Notice = "NotifyPlusROMNetwork"

	// moviecart has started
	NotifyMovieCartStarted Notice = "NotifyMoveCartStarted"

	// unsupported DWARF data
	NotifyUnsupportedDWARF Notice = "NotifyUnsupportedDWARF"
)

// Notify is used for direct communication between a the hardware and the
// emulation package. Not often used but necessary for correct operation of:
//
// Supercharger 'fastload' binaries require a post-load step that initiates the
// hardware based on information in the binary file.
//
// PlusROM cartridges need information about network connectivity from the user
// on first use of the PlusROM system.
type Notify interface {
	Notify(notice Notice) error
}
