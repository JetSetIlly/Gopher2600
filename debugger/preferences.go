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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package debugger

import (
	"strings"

	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

// Preferences defines and collates all the preference values used by the debugger
type Preferences struct {
	dbg         *Debugger
	dsk         *prefs.Disk
	RandomStart *prefs.Bool
	RandomPins  *prefs.Bool
}

func (p Preferences) String() string {
	return p.dsk.String()
}

func loadPreferences(dbg *Debugger) (*Preferences, error) {
	p := &Preferences{
		dbg:         dbg,
		RandomStart: &dbg.vcs.RandomStart,
		RandomPins:  &dbg.vcs.Mem.RandomPins,
	}

	// setup preferences and load from disk
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}
	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}
	err = p.dsk.Add("debugger.randstart", p.RandomStart)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}
	err = p.dsk.Add("debugger.randpins", p.RandomPins)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}
	err = p.dsk.Load()
	if err != nil {
		// ignore missing prefs file errors
		if !errors.Is(err, errors.PrefsNoFile) {
			return nil, err
		}
	}

	return p, nil
}

func (p *Preferences) load() error {
	return p.dsk.Load()
}

func (p *Preferences) save() error {
	return p.dsk.Save()
}

func (p *Preferences) parseCommand(tokens *commandline.Tokens) error {
	action, ok := tokens.Get()

	if !ok {
		p.dbg.printLine(terminal.StyleFeedback, p.String())
		return nil
	}

	switch action {
	case "LOAD":
		return p.load()
	case "SAVE":
		return p.save()
	}

	option, _ := tokens.Get()

	option = strings.ToUpper(option)
	switch option {
	case "RANDSTART":
		switch action {
		case "SET":
			p.RandomStart.Set(true)
		case "NO":
			p.RandomStart.Set(false)
		case "TOGGLE":
			v := p.RandomStart.Get().(bool)
			p.RandomStart.Set(!v)
		}
	case "RANDPINS":
		switch action {
		case "SET":
			p.RandomPins.Set(true)
		case "NO":
			p.RandomPins.Set(true)
		case "TOGGLE":
			v := p.RandomPins.Get().(bool)
			p.RandomPins.Set(!v)
		}
	}

	return nil
}
