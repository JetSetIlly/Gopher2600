package debugger

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/commandline"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"strconv"
	"strings"
)

type watcher struct {
	ai addressInfo

	// whether to watch for a specific value. a matchValue of false means the
	// watcher will match regardless of the value
	matchValue bool
	value      uint8
}

func (wtr watcher) String() string {
	val := ""
	if wtr.matchValue {
		val = fmt.Sprintf(" (value=%#02x)", wtr.value)
	}
	event := "write"
	if wtr.ai.read {
		event = "read"
	}
	return fmt.Sprintf("%s %s%s", wtr.ai, event, val)
}

// the list of currently defined watches in the system
type watches struct {
	dbg    *Debugger
	vcsmem *memory.VCSMemory

	watches             []watcher
	lastAddressAccessed uint16
}

// newWatches is the preferred method of initialisation for the watches type
func newWatches(dbg *Debugger) *watches {
	wtc := &watches{
		dbg:    dbg,
		vcsmem: dbg.vcs.Mem,
	}
	wtc.clear()
	return wtc
}

// clear all watches
func (wtc *watches) clear() {
	wtc.watches = make([]watcher, 0, 10)
}

// drop a specific watcher by a position in the list
func (wtc *watches) drop(num int) error {
	if len(wtc.watches)-1 < num {
		return errors.New(errors.CommandError, fmt.Sprintf("watch #%d is not defined", num))
	}

	h := wtc.watches[:num]
	t := wtc.watches[num+1:]
	wtc.watches = make([]watcher, len(h)+len(t), cap(wtc.watches))
	copy(wtc.watches, h)
	copy(wtc.watches[len(h):], t)

	return nil
}

// check compares the current state of the emulation with every watch
// condition. returns a string listing every condition that matches (separated
// by \n)
func (wtc *watches) check(previousResult string) string {
	checkString := strings.Builder{}
	checkString.WriteString(previousResult)

	for i := range wtc.watches {
		// continue loop if we're not matching last address accessed
		if wtc.watches[i].ai.address != wtc.vcsmem.LastAccessAddress {
			continue
		}

		// continue if this is a repeat of the last address accessed
		if wtc.lastAddressAccessed == wtc.vcsmem.LastAccessAddress {
			continue
		}

		// match watch event to the type of memory access
		if (wtc.watches[i].ai.read == false && wtc.vcsmem.LastAccessWrite) ||
			(wtc.watches[i].ai.read == true && !wtc.vcsmem.LastAccessWrite) {

			// match watched-for value to the value that was read/written to the
			// watched address
			if !wtc.watches[i].matchValue {
				// prepare string according to event
				if wtc.vcsmem.LastAccessWrite {
					checkString.WriteString(fmt.Sprintf("watch at %s (write)\n", wtc.watches[i]))
				} else {
					checkString.WriteString(fmt.Sprintf("watch at %s (read)\n", wtc.watches[i]))
				}
			} else if wtc.watches[i].matchValue && (wtc.watches[i].value == wtc.vcsmem.LastAccessValue) {
				// prepare string according to event
				if wtc.vcsmem.LastAccessWrite {
					checkString.WriteString(fmt.Sprintf("watch at %s (write) %#02x\n", wtc.watches[i], wtc.vcsmem.LastAccessValue))
				} else {
					checkString.WriteString(fmt.Sprintf("watch at %s (read) %#02x\n", wtc.watches[i], wtc.vcsmem.LastAccessValue))
				}
			}
		}
	}

	// note what the last address accessed was
	wtc.lastAddressAccessed = wtc.vcsmem.LastAccessAddress

	return checkString.String()
}

// list currently defined watches
func (wtc *watches) list() {
	if len(wtc.watches) == 0 {
		wtc.dbg.print(terminal.StyleFeedback, "no watches")
	} else {
		wtc.dbg.print(terminal.StyleFeedback, "watches:")
		for i := range wtc.watches {
			wtc.dbg.print(terminal.StyleFeedback, "% 2d: %s", i, wtc.watches[i])
		}
	}
}

// parse tokens and add new watch. unlike breakpoints and traps, only one watch
// at a time can be specified on the command line.
func (wtc *watches) parseWatch(tokens *commandline.Tokens) error {
	var event int

	const (
		either int = iota
		read
		write
	)

	// read mode
	mode, _ := tokens.Get()
	mode = strings.ToUpper(mode)
	switch mode {
	case "READ":
		event = read
	case "WRITE":
		event = write
	default:
		event = either
		tokens.Unget()
	}

	// get address. required.
	a, _ := tokens.Get()

	// convert address
	var ai *addressInfo

	switch event {
	case read:
		ai = wtc.dbg.dbgmem.mapAddress(a, true)
	case write:
		ai = wtc.dbg.dbgmem.mapAddress(a, false)
	default:
		// default to write address and then read address if that's not
		// possible
		ai = wtc.dbg.dbgmem.mapAddress(a, false)
		if ai == nil {
			ai = wtc.dbg.dbgmem.mapAddress(a, true)
		}
	}

	// mapping of the address was unsucessful
	if ai == nil {
		return errors.New(errors.CommandError, fmt.Sprintf("invalid watch address: %s", a))
	}

	// get value if possible
	var val uint64
	var err error
	v, useVal := tokens.Get()
	if useVal {
		val, err = strconv.ParseUint(v, 0, 8)
		if err != nil {
			return errors.New(errors.CommandError, fmt.Sprintf("invalid watch value (%s)", a))
		}
	}

	nw := watcher{
		ai:         *ai,
		matchValue: useVal,
		value:      uint8(val),
	}

	// check to see if watch already exists
	for _, w := range wtc.watches {

		// the conditions for a watch matching are very specific: both must
		// have the same address, be the same /type/ of address (read or
		// write), and the same watch value (if applicable)
		//
		// note that this method means we can add a watch that is a subset of
		// an existing watch (or vice-versa) but that's okay, the check()
		// function will list all matches. plus, if we combine two watches such
		// that only the larger set remains, it may confuse the user
		if w.ai.address == nw.ai.address &&
			w.ai.read == nw.ai.read &&
			w.matchValue == nw.matchValue && w.value == nw.value {

			return errors.New(errors.CommandError, fmt.Sprintf("already being watched (%s)", w))
		}
	}

	// add watch
	wtc.watches = append(wtc.watches, nw)

	return nil
}
