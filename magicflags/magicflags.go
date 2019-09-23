package magicflags

import (
	"flag"
	"fmt"
	"strings"
)

// MagicFlags provides an easy way of handling command line arguments
type MagicFlags struct {
	// should be initialised on declaration
	ProgModes   []string
	DefaultMode string

	Mode string

	ValidSubModes  []string
	DefaultSubMode string
	SubMode        string
	SubModeFlags   *flag.FlagSet

	progFlags *flag.FlagSet
	args      []string
	argsIdx   int
}

// ParseResult is returned from the Parse() function
type ParseResult int

// a list of valid ParseResult values
const (
	ParseContinue ParseResult = iota
	ParseNoArgs
	ParseHelp
)

// nopWriter is used to remove the default output from the flag package
type nopWriter struct{}

func (*nopWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

// Next argument. returns empty string if there is no next argument
func (mf *MagicFlags) Next() string {
	if mf.argsIdx >= len(mf.args) {
		return ""
	}
	return mf.args[mf.argsIdx]
}

// Parse the top level layer of arguments
func (mf *MagicFlags) Parse(args []string) ParseResult {
	// initialise flags
	mf.progFlags = flag.NewFlagSet("", flag.ContinueOnError)
	mf.progFlags.SetOutput(&nopWriter{})

	// make sure everything is initialised
	mf.Mode = ""
	mf.args = args
	mf.argsIdx = 0

	// parse arguments
	if err := mf.progFlags.Parse(mf.args); err != nil {
		if err == flag.ErrHelp {
			if len(mf.ProgModes) > 0 {
				fmt.Printf("available modes: %s\n", strings.Join(mf.ProgModes, ", "))
				if mf.DefaultMode != "" {
					fmt.Printf("default: %s\n", mf.DefaultMode)
				}
			}
			return ParseHelp
		}

		// flags have been set that are not recognised. default to the RUN mode
		mf.Mode = mf.DefaultMode
	} else {
		switch mf.progFlags.NArg() {
		case 0:
			return ParseNoArgs
		case 1:
			// check to see if the single argument is in the list of modes
			arg := strings.ToUpper(mf.progFlags.Arg(0))
			for i := range mf.ProgModes {
				if mf.ProgModes[i] == arg {
					mf.Mode = arg
					mf.argsIdx++
					break
				}
			}

			// argument isn't in list of modes so assume the default mode
			if mf.Mode == "" {
				mf.Mode = mf.DefaultMode
				// not adjusting argsIdx becasue we haven't consumed
				// anything from the args yet
			}

		default:
			// many arguments have been supplied. in this case, the first
			// argument
			mf.Mode = strings.ToUpper(mf.progFlags.Arg(0))
			mf.argsIdx++
		}
	}

	// setup SubModeFlags now so we can add flags once we've decided what
	// submode we're in.
	mf.SubModeFlags = flag.NewFlagSet("", flag.ContinueOnError)

	return ParseContinue
}

// SubParse parses arguments for submodes
func (mf *MagicFlags) SubParse() ParseResult {
	// return immediately if there are no more flags to parse
	if len(mf.args) < 1 || mf.argsIdx > len(mf.args) {
		if mf.DefaultSubMode == "" {
			return ParseNoArgs
		}
		mf.SubMode = mf.DefaultSubMode
		return ParseContinue
	}

	if len(mf.ValidSubModes) > 0 {
		mf.SubModeFlags.SetOutput(&nopWriter{})
	}

	if err := mf.SubModeFlags.Parse(mf.args[mf.argsIdx:]); err != nil {
		if err == flag.ErrHelp {
			if len(mf.ValidSubModes) > 0 {
				fmt.Printf("available sub-modes for %s: %s\n", mf.Mode, strings.Join(mf.ValidSubModes, ", "))
				if mf.DefaultSubMode != "" {
					fmt.Printf("default: %s\n", mf.DefaultSubMode)
				}
			}
			return ParseHelp
		}

		mf.SubMode = mf.DefaultSubMode
	}

	return ParseContinue
}

// TryDefault indicates that the mode is going to look for a default sub-mode
func (mf *MagicFlags) TryDefault() {
	mf.argsIdx++
}

// DefaultFound indicates that the default sub-mode is being used
func (mf *MagicFlags) DefaultFound() {
	mf.argsIdx--
}
