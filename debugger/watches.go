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

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
)

type watcher struct {
	ai addressInfo

	// whether to watch for a specific value. a matchValue of false means the
	// watcher will match regardless of the value
	matchValue bool
	value      uint8

	// wether to compare the address as used or whether to consider mirrored
	// addresses too
	mirrors bool
}

func (w watcher) String() string {
	val := ""
	if w.matchValue {
		val = fmt.Sprintf(" (value=%#02x)", w.value)
	}
	event := "write"
	if w.ai.read {
		event = "read"
	}
	return fmt.Sprintf("%s %s%s", w.ai, event, val)
}

// the list of currently defined watches in the system.
type watches struct {
	dbg                 *Debugger
	watches             []watcher
	lastAddressAccessed uint16
}

// newWatches is the preferred method of initialisation for the watches type.
func newWatches(dbg *Debugger) *watches {
	wtc := &watches{
		dbg: dbg,
	}
	wtc.clear()
	return wtc
}

// clear all watches.
func (wtc *watches) clear() {
	wtc.watches = make([]watcher, 0, 10)
}

// drop a specific watcher by a position in the list.
func (wtc *watches) drop(num int) error {
	if len(wtc.watches)-1 < num {
		return curated.Errorf("watch #%d is not defined", num)
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
// by \n).
func (wtc *watches) check(previousResult string) string {
	if len(wtc.watches) == 0 {
		return previousResult
	}

	checkString := strings.Builder{}
	checkString.WriteString(previousResult)

	for i := range wtc.watches {
		// continue loop if we're not matching last address accessed
		if wtc.watches[i].mirrors {
			if wtc.watches[i].ai.mappedAddress != wtc.dbg.VCS.Mem.LastAccessAddressMapped {
				continue
			}
		} else {
			if wtc.watches[i].ai.address != wtc.dbg.VCS.Mem.LastAccessAddress {
				continue
			}
		}

		// continue if this is a repeat of the last address accessed
		if wtc.lastAddressAccessed == wtc.dbg.VCS.Mem.LastAccessAddress {
			continue
		}

		// match watch event to the type of memory access
		if (!wtc.watches[i].ai.read && wtc.dbg.VCS.Mem.LastAccessWrite) ||
			(wtc.watches[i].ai.read && !wtc.dbg.VCS.Mem.LastAccessWrite) {
			// match watched-for value to the value that was read/written to the
			// watched address
			if !wtc.watches[i].matchValue {
				// prepare string according to event
				if wtc.dbg.VCS.Mem.LastAccessWrite {
					checkString.WriteString(fmt.Sprintf("watch (write) at %s\n", wtc.watches[i]))
				} else {
					checkString.WriteString(fmt.Sprintf("watch (read) at %s\n", wtc.watches[i]))
				}
			} else if wtc.watches[i].matchValue && (wtc.watches[i].value == wtc.dbg.VCS.Mem.LastAccessValue) {
				// prepare string according to event
				if wtc.dbg.VCS.Mem.LastAccessWrite {
					checkString.WriteString(fmt.Sprintf("watch (write) at %s %#02x\n", wtc.watches[i], wtc.dbg.VCS.Mem.LastAccessValue))
				} else {
					checkString.WriteString(fmt.Sprintf("watch (read) at %s %#02x\n", wtc.watches[i], wtc.dbg.VCS.Mem.LastAccessValue))
				}
			}
		}
	}

	// note what the last address accessed was
	wtc.lastAddressAccessed = wtc.dbg.VCS.Mem.LastAccessAddress

	return checkString.String()
}

// list currently defined watches.
func (wtc *watches) list() {
	if len(wtc.watches) == 0 {
		wtc.dbg.printLine(terminal.StyleFeedback, "no watches")
	} else {
		wtc.dbg.printLine(terminal.StyleFeedback, "watches:")
		for i := range wtc.watches {
			wtc.dbg.printLine(terminal.StyleFeedback, "% 2d: %s", i, wtc.watches[i])
		}
	}
}

// parse tokens and add new watch. unlike breakpoints and traps, only one watch
// at a time can be specified on the command line.
func (wtc *watches) parseCommand(tokens *commandline.Tokens) error {
	var event int
	var mirrors bool

	const (
		either int = iota
		read
		write
	)

	// event type
	arg, _ := tokens.Get()
	arg = strings.ToUpper(arg)
	switch arg {
	case "READ":
		event = read
	case "WRITE":
		event = write
	default:
		event = either
		tokens.Unget()
	}

	// mirror address or not
	arg, _ = tokens.Get()
	arg = strings.ToUpper(arg)
	switch arg {
	case "MIRRORS":
		fallthrough
	case "ANY":
		mirrors = true
	default:
		mirrors = false
		tokens.Unget()
	}

	// get address. required.
	a, _ := tokens.Get()

	// convert address
	var ai *addressInfo

	switch event {
	default:
		fallthrough // default to read case
	case read:
		ai = wtc.dbg.dbgmem.mapAddress(a, true)
	case write:
		ai = wtc.dbg.dbgmem.mapAddress(a, false)
	}

	// mapping of the address was unsuccessful
	if ai == nil {
		return curated.Errorf("invalid watch address: %s", a)
	}

	// get value if possible
	var val uint64
	var err error
	v, useVal := tokens.Get()
	if useVal {
		val, err = strconv.ParseUint(v, 0, 8)
		if err != nil {
			return curated.Errorf("invalid watch value (%s)", a)
		}
	}

	nw := watcher{
		ai:         *ai,
		matchValue: useVal,
		value:      uint8(val),
		mirrors:    mirrors,
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
			return curated.Errorf("already being watched (%s)", w)
		}
	}

	// add watch
	wtc.watches = append(wtc.watches, nw)

	return nil
}
