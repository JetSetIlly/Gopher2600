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

	"github.com/jetsetilly/gopher2600/debugger/dbgmem"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
)

type watcherCmp int

const (
	watchCmpAny watcherCmp = iota
	watchCmpChanged
	watchCmpMatch
)

type watcher struct {
	ai dbgmem.AddressInfo

	// the compare method
	cmp watcherCmp

	// the cmpValue being compared
	cmpValue uint8

	// whether the address should be interpreted strictly or whether mirrors
	// should be considered too
	strict bool

	// whether the watcher should match phantom accesses too
	phantom bool
}

func (w watcher) String() string {
	val := ""
	switch w.cmp {
	case watchCmpMatch:
		val = fmt.Sprintf(" (value=%#02x)", w.cmpValue)
	case watchCmpChanged:
		val = " (value=changed)"
	}
	event := "write"
	if w.ai.Read {
		event = "read"
	}
	strict := ""
	if w.strict {
		strict = " (strict)"
	}
	return fmt.Sprintf("%s %s%s%s", w.ai, event, val, strict)
}

// the list of currently defined watches in the system.
type watches struct {
	dbg                 *Debugger
	watches             []watcher
	lastAddressAccessed uint16
	lastAddressWrite    bool
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
		return fmt.Errorf("watch #%d is not defined", num)
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
func (wtc *watches) check() string {
	if len(wtc.watches) == 0 {
		return ""
	}

	// no check if access address & write flag haven't changed
	//
	// note that the write flag comparison is required otherwise RMW
	// instructions will not be caught on the write signal (which would mean
	// that a WRITE watch will never match a RMW instruction)
	if wtc.lastAddressAccessed == wtc.dbg.vcs.Mem.LastCPUAddressLiteral && wtc.lastAddressWrite == wtc.dbg.vcs.Mem.LastCPUWrite {
		return ""
	}

	checkString := strings.Builder{}

	for i, w := range wtc.watches {
		// filter phantom accesses
		if !w.phantom && wtc.dbg.vcs.CPU.PhantomMemAccess {
			return ""
		}

		// pick which addresses to compare depending on whether watch is strict
		if w.strict {
			if wtc.dbg.vcs.Mem.LastCPUAddressLiteral != w.ai.Address {
				continue
			}
		} else {
			if wtc.dbg.vcs.Mem.LastCPUAddressMapped != w.ai.MappedAddress {
				continue
			}
		}

		switch w.cmp {
		case watchCmpMatch:
			if w.cmpValue != wtc.dbg.vcs.Mem.LastCPUData {
				continue
			}
		case watchCmpChanged:
			if w.cmpValue == wtc.dbg.vcs.Mem.LastCPUData {
				continue
			}
			wtc.watches[i].cmpValue = wtc.dbg.vcs.Mem.LastCPUData
		}

		lai := wtc.dbg.dbgmem.GetAddressInfo(wtc.dbg.vcs.Mem.LastCPUAddressLiteral, !wtc.dbg.vcs.Mem.LastCPUWrite)

		if w.ai.Read {
			if !wtc.dbg.vcs.Mem.LastCPUWrite {
				fmt.Fprintf(&checkString, "watch at %s (read value %#02x)", lai, wtc.dbg.vcs.Mem.LastCPUData)
			}
		} else {
			if wtc.dbg.vcs.Mem.LastCPUWrite {
				fmt.Fprintf(&checkString, "watch at %s (written value %#02x)", lai, wtc.dbg.vcs.Mem.LastCPUData)
			}
		}

		if wtc.dbg.vcs.CPU.PhantomMemAccess {
			checkString.WriteString(" phantom")
		}
		if checkString.Len() > 0 {
			checkString.WriteRune('\n')
		}
	}

	// note what the last address accessed was
	wtc.lastAddressAccessed = wtc.dbg.vcs.Mem.LastCPUAddressLiteral
	wtc.lastAddressWrite = wtc.dbg.vcs.Mem.LastCPUWrite

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
	var write bool
	var strict bool
	var phantom bool
	var cmp watcherCmp

	tokens.ForRemaining(func() {
		// event type
		arg, _ := tokens.Get()
		arg = strings.ToUpper(arg)
		switch arg {
		case "READ":
			write = false
		case "WRITE":
			write = true
		default:
			tokens.Unget()
		}

		// strict addressing or not
		arg, _ = tokens.Get()
		arg = strings.ToUpper(arg)
		switch arg {
		case "STRICT":
			strict = true
		default:
			tokens.Unget()
		}

		// phantom addressing
		arg, _ = tokens.Get()
		arg = strings.ToUpper(arg)
		switch arg {
		case "PHANTOM":
			fallthrough
		case "GHOST":
			phantom = true
		default:
			tokens.Unget()
		}

		// changed values
		arg, _ = tokens.Get()
		arg = strings.ToUpper(arg)
		switch arg {
		case "CHANGED":
			cmp = watchCmpChanged
		default:
			tokens.Unget()
		}
	})

	// get address. required.
	a, _ := tokens.Get()

	// convert address
	var ai *dbgmem.AddressInfo

	if write {
		ai = wtc.dbg.dbgmem.GetAddressInfo(a, false)
	} else {
		ai = wtc.dbg.dbgmem.GetAddressInfo(a, true)
	}

	// mapping of the address was unsuccessful
	if ai == nil {
		if write {
			return fmt.Errorf("invalid watch address (%s) expecting 16-bit address or a write symbol", a)
		}
		return fmt.Errorf("invalid watch address (%s) expecting 16-bit address or a read symbol", a)
	}

	// get value if possible
	var val uint64
	var err error
	if v, ok := tokens.Get(); ok {
		if cmp != watchCmpAny {
			return fmt.Errorf("match value given with CHANGED argument")
		}
		cmp = watchCmpMatch
		val, err = strconv.ParseUint(v, 0, 8)
		if err != nil {
			return fmt.Errorf("invalid watch value (%s) expecting 8-bit value", a)
		}
	}

	nw := watcher{
		ai:       *ai,
		cmp:      cmp,
		cmpValue: uint8(val),
		strict:   strict,
		phantom:  phantom,
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
		if w.ai.Address == nw.ai.Address &&
			w.ai.Read == nw.ai.Read &&
			w.cmp == nw.cmp {
			return fmt.Errorf("already being watched (%s)", w)
		}
	}

	// add watch
	wtc.watches = append(wtc.watches, nw)

	return nil
}
