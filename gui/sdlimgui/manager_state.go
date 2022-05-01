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
	"fmt"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
	"github.com/jetsetilly/gopher2600/resources/fs"
)

const managerStateFile = "managerState"
const managerStateNumFields = 3

// save window open/close state to disk.
//
// uses a similar method to the prefs package and in fact references the prefs
// package for consistency
//
// called once on destroy
func (wm *manager) saveManagerState() (rerr error) {
	pth, err := resources.JoinPath(managerStateFile)
	if err != nil {
		return curated.Errorf("manager state: %v", err)
	}

	// create a new state file
	f, err := fs.Create(pth)
	if err != nil {
		return curated.Errorf("manager state: %v", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			rerr = curated.Errorf("manager state: %v", err)
		}
	}()

	// write boiler plate warning to manager state file
	s := fmt.Sprintf("%s\n", prefs.WarningBoilerPlate)
	n, err := fmt.Fprint(f, s)
	if err != nil {
		return curated.Errorf("manager state: %v", err)
	}
	if n != len(s) {
		return curated.Errorf("manager state: %v", "incorrect number of characters written to file")
	}

	for key, win := range wm.debuggerWindows {
		s := fmt.Sprintf("%s%s%s%s%v\n", emulation.ModeDebugger.String(), prefs.KeySep, key, prefs.KeySep, win.debuggerIsOpen())
		n, err := fmt.Fprint(f, s)
		if err != nil {
			return curated.Errorf("manager state: %v", err)
		}
		if n != len(s) {
			return curated.Errorf("manager state: %v", "incorrect number of characters written to file")
		}
	}

	for key, win := range wm.playmodeWindows {
		s := fmt.Sprintf("%s%s%s%s%v\n", emulation.ModePlay.String(), prefs.KeySep, key, prefs.KeySep, win.playmodeIsOpen())
		n, err := fmt.Fprint(f, s)
		if err != nil {
			return curated.Errorf("manager state: %v", err)
		}
		if n != len(s) {
			return curated.Errorf("manager state: %v", "incorrect number of characters written to file")
		}
	}

	return nil
}

// load window open/close state from disk.
//
// uses a similar method to the prefs package and in fact references the prefs
// package for consistency
//
// called once on startup
func (wm *manager) loadManagerState() (rerr error) {
	pth, err := resources.JoinPath(managerStateFile)
	if err != nil {
		return curated.Errorf("manager state: %v", err)
	}

	// open an existing state file
	f, err := fs.Open(pth)
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			return nil
		}
		return curated.Errorf("manager state: %v", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			rerr = curated.Errorf("manager state: %v", err)
		}
	}()

	// new scanner - splitting on newlines
	scanner := bufio.NewScanner(f)

	// check validity of file by checking the first line for the boiler plate warning
	scanner.Scan()
	if len(scanner.Text()) > 0 && scanner.Text() != prefs.WarningBoilerPlate {
		return curated.Errorf("manager state: %v", fmt.Errorf("not a valid manager state file (%s)", pth))
	}

	// loop through file until EOF
	for scanner.Scan() {
		// split line into key/value pair
		spt := strings.SplitN(scanner.Text(), prefs.KeySep, managerStateNumFields)

		// ignore lines that haven't been split successfully
		if len(spt) != managerStateNumFields {
			continue
		}

		// open/close window according to the state file
		m := spt[0]
		k := spt[1]
		v := spt[2]

		if m == emulation.ModeDebugger.String() {
			if w, ok := wm.debuggerWindows[k]; ok {
				w.debuggerSetOpen(strings.ToUpper(v) == "TRUE")
			}
		}

		if m == emulation.ModePlay.String() {
			if w, ok := wm.playmodeWindows[k]; ok {
				w.playmodeSetOpen(strings.ToUpper(v) == "TRUE")
			}
		}
	}

	// hold arrangeBySize signal for 5 frames
	wm.arrangeBySize = 5

	return nil
}
