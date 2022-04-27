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
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
	"github.com/jetsetilly/gopher2600/resources/fs"
)

const managerStateFile = "managerState"

// save window open/close state to disk.
//
// uses a similar method to the prefs package and in fact references the prefs
// package for consistency
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

	for key, def := range wm.windows {
		s := fmt.Sprintf("%s%s%v\n", key, prefs.KeySep, def.isOpen())
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
func (wm *manager) loadManagerState() (rerr error) {
	pth, err := resources.JoinPath(managerStateFile)
	if err != nil {
		return curated.Errorf("manager state: %v", err)
	}

	// open an existing state file
	f, err := fs.Open(pth)
	if err != nil {
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
		spt := strings.SplitN(scanner.Text(), prefs.KeySep, 2)

		// ignore lines that haven't been split successfully
		if len(spt) != 2 {
			continue
		}

		// open/close window according to the state file
		k := spt[0]
		v := spt[1]
		wm.windows[k].setOpen(strings.ToUpper(v) == "TRUE")
	}

	return nil
}
