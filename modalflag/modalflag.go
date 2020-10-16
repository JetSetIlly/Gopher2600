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

package modalflag

import (
	"flag"
	"io"
	"strings"
	"time"
)

const modeSeparator = "/"

// Modes provides an easy way of handling command line arguments. The Output
// field should be specified before calling Parse() or you will not see any
// help messages.
type Modes struct {
	// where to print output (help messages etc). defaults to os.Stdout
	Output io.Writer

	// whether Parse() has been called recently
	parsed bool

	// the underlying flag structure. this can be used directly as described by
	// the flag.FlagSet documentation. the only thing you shouldn't do is call
	// Parse() directly. Use the Parse() function of the parent Modes struct
	// instead.
	//
	// a new flagset is created on every call to NewArgs() and NewMode()
	flags *flag.FlagSet

	// the argument list as specified by the NewArgs() function
	args    []string
	argsIdx int

	// the most recent list of sub-modes specified with the NewMode() function
	subModes []string

	// path is the series of sub-modes that have been found during subsequent
	// calls to Parse()
	//
	// we never reset this variable
	path []string

	// some modes will benefit from a verbose explanation. use
	additionalHelp string
}

func (md *Modes) String() string {
	return md.Path()
}

// Mode returns the last mode to be encountered.
func (md *Modes) Mode() string {
	if len(md.path) == 0 {
		return ""
	}
	return md.path[len(md.path)-1]
}

// Path returns a string all the modes encountered during parsing.
func (md *Modes) Path() string {
	return strings.Join(md.path, modeSeparator)
}

// NewArgs with a string of arguments (from the command line for example).
func (md *Modes) NewArgs(args []string) {
	// initialise args
	md.args = args
	md.argsIdx = 0

	// by definition, a newly initialised Modes struct begins with a new mode
	md.NewMode()
}

// NewMode indicates that further arguments should be considered part of a new
// mode.
func (md *Modes) NewMode() {
	md.subModes = []string{}
	md.flags = flag.NewFlagSet("", flag.ContinueOnError)
	md.parsed = false
}

// AdditionalHelp allows you to add extensive help text to be displayed in
// addition to the regular help on available flags.
func (md *Modes) AdditionalHelp(help string) {
	md.additionalHelp = help
}

// Parsed returns false if Parse() has not yet been called since either a call
// to NewArgs() or NewMode(). Note that, a Modes struct is considered to be
// Parsed() even if Parse() results in an error.
func (md *Modes) Parsed() bool {
	return md.parsed
}

// ParseResult is returned from the Parse() function.
type ParseResult int

// a list of valid ParseResult values.
const (
	// Continue with command line processing. How this result should be
	// interpreted depends on the context, which the caller of the Parse()
	// function knows best. However, generally we can say that if sub-modes
	// were specified in the preceding call to NewMode() then the Mode field
	// of the Modes struct should be checked.
	ParseContinue ParseResult = iota

	// Help was requested and has been printed.
	ParseHelp

	// an error has occurred and is returned as the second return value.
	ParseError
)

// Parse the top level layer of arguments. Returns a value of ParseResult.
// The idiomatic usage is as follows:
//
//		r, err := md.Parse(".....") }
//		case ParseHelp:
//			// help message has already been printed
//			return
//		case ParseError:
//			printError(err)
//			return
//		}
//
// Help messages are handled automatically by the function. The return value
// ParseHelp is to help you guide your program appropriately. The above pattern
// suggests it should be treated similarly to an error and without the need to
// display anything further to the user.
//
// Note that the Output field of the Modes struct *must* be specified in order
// for any help messages to be visible. The most common and useful value of the
// field is os.Stdout.
func (md *Modes) Parse() (ParseResult, error) {
	// flag the parsed flag in all instances, even if we eventually return an
	// error
	md.parsed = true

	// set output of flags.Parse() to an instance of helpWriter
	hw := &helpWriter{}
	md.flags.SetOutput(hw)

	// parse arguments
	err := md.flags.Parse(md.args[md.argsIdx:])
	if err != nil {
		if err == flag.ErrHelp {
			hw.Help(md.Output, md.Path(), md.subModes, md.additionalHelp)
			hw.Clear()
			return ParseHelp, nil
		}

		// flags have been set that are not recognised. if sub-modes and a
		// default mode have been defined, set selected mode to default mode
		// and continue. otherwise return error
		if len(md.subModes) > 0 {
			md.path = append(md.path, md.subModes[0])
		} else {
			return ParseError, err
		}
	} else if len(md.subModes) > 0 {
		arg := strings.ToUpper(md.flags.Arg(0))

		// check to see if the single argument is in the list of modes,
		// starting off assuming it isn't
		mode := md.subModes[0]
		for i := range md.subModes {
			if md.subModes[i] == arg {
				// found matching sub-mode
				mode = arg
				md.argsIdx++
				break // for loop
			}
		}

		// add mode (either one we've found or the default) and add it to
		// the path
		md.path = append(md.path, mode)
	}

	return ParseContinue, nil
}

// RemainingArgs after a call to Parse() ie. arguments that aren't flags or a
// listed sub-mode.
func (md *Modes) RemainingArgs() []string {
	return md.flags.Args()
}

// GetArg returns the numbered argument that isn't a flag or listed sub-mode.
func (md *Modes) GetArg(i int) string {
	return md.flags.Arg(i)
}

// AddSubModes to list of submodes for next parse. The first sub-mode in the
// list is considered to be the default sub-mode. If you need more control over
// this, AddDefaultSubMode() can be used.
//
// Note that sub-mode comparisons are case insensitive.
func (md *Modes) AddSubModes(submodes ...string) {
	md.subModes = append(md.subModes, submodes...)
	for i := range md.subModes {
		md.subModes[i] = strings.ToUpper(md.subModes[i])
	}
}

// AddDefaultSubMode to list of sub-modes.
func (md *Modes) AddDefaultSubMode(defSubMode string) {
	md.subModes = append([]string{defSubMode}, md.subModes...)
}

// AddBool flag for next call to Parse().
func (md *Modes) AddBool(name string, value bool, usage string) *bool {
	return md.flags.Bool(name, value, usage)
}

// AddDuration flag for next call to Parse().
func (md *Modes) AddDuration(name string, value time.Duration, usage string) *time.Duration {
	return md.flags.Duration(name, value, usage)
}

// AddFloat64 flag for next call to Parse().
func (md *Modes) AddFloat64(name string, value float64, usage string) *float64 {
	return md.flags.Float64(name, value, usage)
}

// AddInt flag for next call to Parse().
func (md *Modes) AddInt(name string, value int, usage string) *int {
	return md.flags.Int(name, value, usage)
}

// AddInt64 flag for next call to Parse().
func (md *Modes) AddInt64(name string, value int64, usage string) *int64 {
	return md.flags.Int64(name, value, usage)
}

// AddString flag for next call to Parse().
func (md *Modes) AddString(name string, value string, usage string) *string {
	return md.flags.String(name, value, usage)
}

// AddUint flag for next call to Parse().
func (md *Modes) AddUint(name string, value uint, usage string) *uint {
	return md.flags.Uint(name, value, usage)
}

// AddUint64 flag for next call to Parse().
func (md *Modes) AddUint64(name string, value uint64, usage string) *uint64 {
	return md.flags.Uint64(name, value, usage)
}

// Visit visits the flags in lexicographical order, calling fn for each. It
// visits only those flags that have been set.
func (md *Modes) Visit(fn func(flag string)) {
	md.flags.Visit(func(f *flag.Flag) {
		fn(f.Name)
	})
}
