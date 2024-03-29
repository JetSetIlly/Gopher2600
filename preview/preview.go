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

package preview

import (
	"fmt"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

type Emulation struct {
	vcs *hardware.VCS
}

// NewEmulation is the preferred method of initialisation for the Emulation type
func NewEmulation(prefs *preferences.Preferences) (*Emulation, error) {
	em := &Emulation{}

	// the VCS and not referred to directly again
	tv, err := television.NewTelevision("AUTO")
	if err != nil {
		return nil, fmt.Errorf("preview: %w", err)
	}
	tv.SetFPSCap(false)

	// create a new VCS emulation
	em.vcs, err = hardware.NewVCS(tv, prefs)
	if err != nil {
		return nil, fmt.Errorf("preview: %w", err)
	}
	em.vcs.Env.Label = environment.Label("preview")

	return em, nil
}

// RunN runs the preview emulation for N frames
func (em *Emulation) RunN(filename string, N int) error {
	loader, err := cartridgeloader.NewLoader(filename, "")
	if err != nil {
		return fmt.Errorf("preview: %w", err)
	}

	// we don't want the preview emulation to run for too long
	timeout := time.After(1 * time.Second)

	em.vcs.AttachCartridge(loader, true)
	err = em.vcs.RunForFrameCount(N, func(_ int) (govern.State, error) {
		select {
		case <-timeout:
			return govern.Ending, nil
		default:
		}
		return govern.Running, nil
	})
	if err != nil {
		return fmt.Errorf("preview: %w", err)
	}

	return nil
}

// Run the preview emulation for 30 frames
func (em *Emulation) Run(filename string) error {
	return em.RunN(filename, 30)
}
