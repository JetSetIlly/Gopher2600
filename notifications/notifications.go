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
	NotifyInitialising  Notice = "NotifyInitialising"
	NotifyPause         Notice = "NotifyPause"
	NotifyRun           Notice = "NotifyRun"
	NotifyRewindBack    Notice = "NotifyRewindBack"
	NotifyRewindFoward  Notice = "NotifyRewindFoward"
	NotifyRewindAtStart Notice = "NotifyRewindAtStart"
	NotifyRewindAtEnd   Notice = "NotifyRewindAtEnd"
	NotifyScreenshot    Notice = "NotifyScreenshot"
	NotifyMute          Notice = "NotifyMute"
	NotifyUnmute        Notice = "NotifyUnmute"

	// LoadStarted is raised for Supercharger mapper whenever a new tape read
	// sequence if started.
	NotifySuperchargerLoadStarted Notice = "NotifySuperchargerLoadStarted"

	// If Supercharger is loading from a fastload binary then this event is
	// raised when the loading has been completed.
	NotifySuperchargerFastloadEnded Notice = "NotifySuperchargerFastloadEnded"

	// If Supercharger is loading from a sound file (eg. mp3 file) then these
	// events area raised when the loading has started/completed.
	NotifySuperchargerSoundloadStarted Notice = "NotifySuperchargerSoundloadStarted"
	NotifySuperchargerSoundloadEnded   Notice = "NotifySuperchargerSoundloadEnded"

	// tape is rewinding.
	NotifySuperchargerSoundloadRewind Notice = "NotifySuperchargerSoundloadRewind"

	// PlusROM cartridge has been inserted.
	NotifyPlusROMInserted Notice = "NotifyPlusROMInserted"

	// PlusROM network activity.
	NotifyPlusROMNetwork Notice = "NotifyPlusROMNetwork"

	// PlusROM new installation
	NotifyPlusROMNewInstallation Notice = "NotifyPlusROMNewInstallation"

	// Moviecart started
	NotifyMovieCartStarted Notice = "NotifyMoveCartStarted"

	// unsupported DWARF data
	NotifyUnsupportedDWARF Notice = "NotifyUnsupportedDWARF"

	// coprocessor development information has been loaded
	NotifyCoprocDevStarted Notice = "NotifyCoprocDevStarted"
	NotifyCoprocDevEnded   Notice = "NotifyCoprocDevEnded"
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
