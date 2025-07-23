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

package debugger

// CommandLineOptions holds all the values that can be specified on the command line
// when launching the application. Some arguments are used by both modes while
// some are mode specific.
//
// The reason why we maintain pointers to the values is because we are using
// the modalflag package and by induction the flag package in the standard
// library, which is where this requirement originates.
type CommandLineOptions struct {
	// common to debugger and play modes
	Log       bool
	Spec      string
	FpsCap    bool
	Multiload int
	Mapping   string
	Bank      string
	Left      string
	Right     string
	SwapPorts bool
	Profile   string
	ELF       string

	// playmode only
	ComparisonROM        string
	ComparisonPrefs      string
	Record               bool
	RecordFilename       string
	PlaybackCheckROM     bool
	PlaybackIgnoreDigest bool
	PatchFile            string
	Wav                  bool
	Video                bool
	NoEject              bool
	Macro                string

	// debugger only
	InitScript string
	TermType   string
}
