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

	// whether to watch for a specific value
	matchValue bool
	value      uint8

	event watchEvent
}

func (wtr watcher) String() string {
	val := ""
	if wtr.matchValue {
		val = fmt.Sprintf(" (value=%#02x)", wtr.value)
	}
	return fmt.Sprintf("%#04x %s%s", wtr.address, wtr.event, val)
}

type watches struct {
	dbg    *Debugger
	vcsmem *memory.VCSMemory

	watches             []watcher
	lastAddressAccessed uint16
}

// newBreakpoints is the preferred method of initialisation for breakpoins
func newWatches(dbg *Debugger) *watches {
	wtc := &watches{
		dbg:    dbg,
		vcsmem: dbg.vcs.Mem,
	}
	wtc.clear()
	return wtc
}

func (wtc *watches) clear() {
	wtc.watches = make([]watcher, 0, 10)
}

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

// breakpoints.check compares the current state of the emulation with every
// break condition. it returns a string listing every condition that applies
func (wtc *watches) check(previousResult string) string {
	checkString := strings.Builder{}
	checkString.WriteString(previousResult)

	for i := range wtc.watches {
		// match addresses if memory has been accessed recently (LastAddressFlag)
		if wtc.watches[i].address == wtc.vcsmem.LastAccessAddress {
			if wtc.lastAddressAccessed != wtc.vcsmem.LastAccessAddress {
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
		}
	}

	// note what the last address accessed was
	wtc.lastAddressAccessed = wtc.vcsmem.LastAccessAddress

	return checkString.String()
}

func (wtc *watches) list() {
	if len(wtc.watches) == 0 {
		wtc.dbg.print(terminal.StyleFeedback, "no watches")
	} else {
		wtc.dbg.print(terminal.StyleFeedback, "watches")
		for i := range wtc.watches {
			wtc.dbg.print(terminal.StyleFeedback, "% 2d: %s", i, wtc.watches[i])
		}
	}
}

func (wtc *watches) parseWatch(tokens *commandline.Tokens, dbgmem *memoryDebug) error {
	var event watchEvent

	// read mode
	mode, present := tokens.Get()
	if !present {
		return errors.New(errors.CommandError, "watch address required")
	}
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
	a, present := tokens.Get()
	if !present {
		return errors.New(errors.CommandError, "watch address required")
	}

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
			//  o if the existing entry is looking for read/write events then
			//		this is duplicate watcher
			//  o if the existing entry is looking for read events and new
			//		watcher is not then update exising entry
			//  o ditto for write events
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
