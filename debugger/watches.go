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

// the watch system can watch for read, write events specifically or for either
// event
type watchEvent int

const (
	watchEventAny watchEvent = iota
	watchEventRead
	watchEventWrite
)

func (ev watchEvent) String() string {
	switch ev {
	case watchEventRead:
		return "read"
	case watchEventWrite:
		return "write"
	case watchEventAny:
		return "read/write"
	default:
		return ""
	}
}

type watcher struct {
	address uint16
	event   watchEvent

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
	return fmt.Sprintf("%#04x %s%s", wtr.address, wtr.event, val)
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
		if wtc.watches[i].address != wtc.vcsmem.LastAccessAddress {
			continue
		}

		// continue if this is a repeat of the last address accessed
		if wtc.lastAddressAccessed == wtc.vcsmem.LastAccessAddress {
			continue
		}

		// match watch event to the type of memory access
		if wtc.watches[i].event == watchEventAny ||
			(wtc.watches[i].event == watchEventWrite && wtc.vcsmem.LastAccessWrite) ||
			(wtc.watches[i].event == watchEventRead && !wtc.vcsmem.LastAccessWrite) {

			// match watched-for value to the value that was read/written to the
			// watched address
			if !wtc.watches[i].matchValue ||
				(wtc.watches[i].matchValue && (wtc.watches[i].value == wtc.vcsmem.LastAccessValue)) {

				// prepare string according to event
				if wtc.vcsmem.LastAccessWrite {
					checkString.WriteString(fmt.Sprintf("watch at %s -> %#02x\n", wtc.watches[i], wtc.vcsmem.LastAccessValue))
				} else {
					checkString.WriteString(fmt.Sprintf("watch at %s\n", wtc.watches[i]))
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
func (wtc *watches) parseWatch(tokens *commandline.Tokens, dbgmem *memoryDebug) error {
	var event watchEvent

	// read mode
	mode, _ := tokens.Get()
	mode = strings.ToUpper(mode)
	switch mode {
	case "READ":
		event = watchEventRead
	case "WRITE":
		event = watchEventWrite
	default:
		event = watchEventAny
		tokens.Unget()
	}

	// get address. required.
	a, _ := tokens.Get()

	// convert address
	var ai *addressInfo
	switch event {
	case watchEventRead:
		ai = dbgmem.mapAddress(a, true)
	case watchEventWrite:
		ai = dbgmem.mapAddress(a, false)
	default:
		// try both perspectives
		ai = dbgmem.mapAddress(a, false)
		if ai == nil {
			ai = dbgmem.mapAddress(a, true)
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
		address:    ai.mappedAddress,
		matchValue: useVal,
		value:      uint8(val),
		event:      event,
	}

	// check to see if watch already exists
	for i, w := range wtc.watches {
		if w.address == nw.address && w.matchValue == nw.matchValue && w.value == nw.value {
			// we've found a matching watcher (address and value if
			// appropriate). the following switch handles how the watcher event
			// matches:
			switch w.event {
			case watchEventRead:
				if nw.event == watchEventRead {
					return errors.New(errors.CommandError, fmt.Sprintf("already being watched (%s)", w))
				}
				wtc.watches[i].event = watchEventAny
				return nil
			case watchEventWrite:
				if nw.event == watchEventWrite {
					return errors.New(errors.CommandError, fmt.Sprintf("already being watched (%s)", w))
				}
				wtc.watches[i].event = watchEventAny
				return nil
			case watchEventAny:
				return errors.New(errors.CommandError, fmt.Sprintf("already being watched (%s)", w))
			}
		}
	}

	// add watch
	wtc.watches = append(wtc.watches, nw)

	return nil
}
