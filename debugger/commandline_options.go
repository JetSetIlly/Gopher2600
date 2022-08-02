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
type CommandLineOptions struct {
	// common to debugger and play modes
	Log       *bool
	Spec      *string
	FpsCap    *bool
	Multiload *int
	Mapping   *string
	Left      *string
	Right     *string
	Profile   *string

	// playmode only
	ComparisonROM   *string
	ComparisonPrefs *string
	Record          *bool
	PatchFile       *string
	Wav             *bool

	// debugger only
	InitScript *string
	TermType   *string
}
