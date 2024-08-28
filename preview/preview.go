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

const previewLabel = environment.Label("preview")

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
	em.vcs, err = hardware.NewVCS(previewLabel, tv, nil, prefs)
	if err != nil {
		return nil, fmt.Errorf("preview: %w", err)
	}

	return em, nil
}

// RunN runs the preview emulation for N frames
func (em *Emulation) RunN(loader cartridgeloader.Loader, N int) error {
	// we don't want the preview emulation to run for too long
	timeout := time.After(1 * time.Second)

	err := em.vcs.AttachCartridge(loader, true)
	if err != nil {
		return fmt.Errorf("preview: %w", err)
	}

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

// Run the preview emulation for 60 frames
func (em *Emulation) Run(loader cartridgeloader.Loader) error {
	// the number of frames tried so far:
	// 30 => too few for Spike's Peak
	return em.RunN(loader, 60)
}
