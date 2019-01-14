package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/debugger/ui"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"strconv"
	"strings"
)

type watchEvent int

func (ev watchEvent) String() string {
	switch ev {
	case watchEventRead:
		return "read-only"
	case watchEventWrite:
		return "write-only"
	case watchEventAny:
		fallthrough
	default:
		return ""
	}
}

const (
	watchEventAny watchEvent = iota
	watchEventRead
	watchEventWrite
)

type watcher struct {
	address    uint16
	matchValue bool
	value      uint8
	event      watchEvent
}

func (wtr watcher) String() string {
	val := ""
	if wtr.matchValue {
		val = fmt.Sprintf("(value=%#02x)", wtr.value)
	}
	return fmt.Sprintf("%#04x %s %s", wtr.address, wtr.event, val)
}

type watches struct {
	dbg    *Debugger
	vcsmem *memory.VCSMemory

	watches []watcher
}

// newBreakpoints is the preferred method of initialisation for breakpoins
func newWatches(dbg *Debugger) *watches {
	wtc := new(watches)
	wtc.dbg = dbg
	wtc.vcsmem = dbg.vcs.Mem
	wtc.clear()
	return wtc
}

func (wtc *watches) clear() {
	wtc.watches = make([]watcher, 0, 10)
}

func (wtc *watches) drop(num int) error {
	if len(wtc.watches)-1 < num {
		return errors.NewGopherError(errors.CommandError, fmt.Errorf("watch #%d is not defined", num))
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
		if wtc.vcsmem.LastAddressAccessFlag && wtc.watches[i].address == wtc.vcsmem.LastAddressAccessed {

			// match events
			if wtc.watches[i].event == watchEventAny || (wtc.watches[i].event == watchEventWrite && wtc.vcsmem.LastAddressAccessWrite) || (wtc.watches[i].event == watchEventRead && !wtc.vcsmem.LastAddressAccessWrite) {
				// match value
				if !wtc.watches[i].matchValue || (wtc.watches[i].matchValue && (wtc.watches[i].value == wtc.vcsmem.LastAddressAccessValue)) {
					// prepare string according to event
					if wtc.vcsmem.LastAddressAccessWrite {
						checkString.WriteString(fmt.Sprintf("watch at %s -> %#02x\n", wtc.watches[i], wtc.vcsmem.LastAddressAccessValue))
					} else {
						checkString.WriteString(fmt.Sprintf("watch at %s\n", wtc.watches[i]))
					}

					wtc.vcsmem.LastAddressAccessFlag = false
				}
			}
		}
	}
	return checkString.String()
}

func (wtc *watches) list() {
	if len(wtc.watches) == 0 {
		wtc.dbg.print(ui.Feedback, "no watches")
	} else {
		for i := range wtc.watches {
			wtc.dbg.print(ui.Feedback, "% 2d: %s", i, wtc.watches[i])
		}
	}
}

func (wtc *watches) parseWatch(tokens *input.Tokens) error {
	// TODO: move this into parseWatch() function, in they style
	// parseBreakpoint() and parseTrap()
	var event watchEvent

	// read mode
	mode, present := tokens.Get()
	if !present {
		return nil
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

	// read address. required.
	a, present := tokens.Get()
	if !present {
		return fmt.Errorf("address required")
	}
	addr, err := strconv.ParseUint(a, 0, 16)
	if err != nil {
		return fmt.Errorf("16bit numeric address required (%s)", a)
	}

	// read value if possible
	var val uint64
	a, useVal := tokens.Get()
	if useVal {
		val, err = strconv.ParseUint(a, 0, 8)
		if err != nil {
			return fmt.Errorf("8bit value required (%s)", a)
		}
	}

	// add watch
	wtc.watches = append(wtc.watches, watcher{address: uint16(addr), matchValue: useVal, value: uint8(val), event: event})

	a, present = tokens.Get()

	return nil
}
