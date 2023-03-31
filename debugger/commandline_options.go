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
	Log       *bool
	Spec      *string
	FpsCap    *string
	Multiload *int
	Mapping   *string
	Left      *string
	Right     *string
	Profile   *string
	ELF       *string

	// playmode only
	ComparisonROM    *string
	ComparisonPrefs  *string
	Record           *bool
	PlaybackCheckROM *bool
	PatchFile        *string
	Wav              *bool
	NoEject          *bool
	Macro            *string

	// debugger only
	InitScript *string
	TermType   *string
}

// NewCommandLineOptions creates a minimum instance of CommandLineOptions such
// that it is safe to dereference the fields in all situations.
//
// The values of these fields are shared by type and will be the default values
// for that type. ie. a bool is false, an int is zero, etc. Care should be
// taken therefore to replace the instance with the result from the modalflag
// (or flag) package.
func NewCommandLineOptions() CommandLineOptions {
	var b bool
	var s string
	var i int
	return CommandLineOptions{
		Log:             &b,
		Spec:            &s,
		FpsCap:          &s,
		Multiload:       &i,
		Mapping:         &s,
		Left:            &s,
		Right:           &s,
		Profile:         &s,
		ELF:             &s,
		ComparisonROM:   &s,
		ComparisonPrefs: &s,
		Record:          &b,
		PatchFile:       &s,
		Wav:             &b,
		InitScript:      &s,
		TermType:        &s,
		NoEject:         &b,
	}
}
