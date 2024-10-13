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

package sdlimgui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
	"github.com/jetsetilly/gopher2600/resources/fs"
)

const managerHotkeysFile = "managerHotkeys"
const managerHotkeysNumFields = 2

// save hotkeys to disk
//
// uses a similar method to the prefs package and in fact references the prefs
// package for consistency
//
// called once on destroy
func (wm *manager) saveManagerHotkeys() (rerr error) {
	pth, err := resources.JoinPath(managerHotkeysFile)
	if err != nil {
		return fmt.Errorf("manager hotkeys: %w", err)
	}

	// create a new hotkeys file
	f, err := fs.Create(pth)
	if err != nil {
		return fmt.Errorf("manager hotkeys: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			rerr = fmt.Errorf("manager hotkeys: %w", err)
		}
	}()

	// write boiler plate warning to manager hotkeys file
	s := fmt.Sprintf("%s\n", prefs.WarningBoilerPlate)
	n, err := fmt.Fprint(f, s)
	if err != nil {
		return fmt.Errorf("manager hotkeys: %w", err)
	}
	if n != len(s) {
		return fmt.Errorf("manager hotkeys: incorrect number of characters written to file")
	}

	// write hotkeys and window name to file
	for key, win := range wm.hotkeys {
		s := fmt.Sprintf("%c%s%s\n", key, prefs.KeySep, win.id())
		n, err := fmt.Fprint(f, s)
		if err != nil {
			return fmt.Errorf("manager hotkeys: %w", err)
		}
		if n != len(s) {
			return fmt.Errorf("manager hotkeys: incorrect number of characters written to file")
		}
	}

	return nil
}

// load hotkeys from disk
//
// uses a similar method to the prefs package and in fact references the prefs
// package for consistency
//
// called once on startup
func (wm *manager) loadManagerHotkeys() (rerr error) {
	pth, err := resources.JoinPath(managerHotkeysFile)
	if err != nil {
		return fmt.Errorf("manager hotkeys: %w", err)
	}

	// open an existing hotkeys file
	f, err := fs.Open(pth)
	if err != nil {
		var pathError *os.PathError
		if errors.As(err, &pathError) {
			return nil
		}
		return fmt.Errorf("manager hotkeys: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			rerr = fmt.Errorf("manager hotkeys: %w", err)
		}
	}()

	// new scanner - splitting on newlines
	scanner := bufio.NewScanner(f)

	// check validity of file by checking the first line for the boiler plate warning
	scanner.Scan()
	if len(scanner.Text()) > 0 && scanner.Text() != prefs.WarningBoilerPlate {
		return fmt.Errorf("manager hotkeys: not a valid manager hotkeys file (%s)", pth)
	}

	// loop through file until EOF
	for scanner.Scan() {
		// split line into key/value pair
		spt := strings.SplitN(scanner.Text(), prefs.KeySep, managerHotkeysNumFields)

		// ignore lines that haven't been split successfully
		if len(spt) != managerHotkeysNumFields {
			continue
		}

		// assign keys to windows
		key := spt[0]
		win := wm.debuggerWindows[spt[1]]
		if win != nil {
			if len(key) > 0 {
				wm.hotkeys[rune(key[0])] = win
			}
		}
	}

	return nil
}
