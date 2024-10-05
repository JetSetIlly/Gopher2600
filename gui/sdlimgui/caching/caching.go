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

package caching

import (
	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/rewind"
)

// cachedVCS contains the cached components of the emulated VCS
type cachedVCS struct {
	CPU  *cpu.CPU
	Mem  *memory.Memory
	RIOT *riot.RIOT
	TIA  *tia.TIA
}

// plumbing is necessary to make sure that pointers to memory etc. are correct
func (vcs cachedVCS) plumb(env *environment.Environment) {
	vcs.CPU.Plumb(vcs.Mem)
	vcs.Mem.Plumb(env, true)
	vcs.RIOT.Plumb(env, vcs.Mem.RIOT, vcs.Mem.TIA)
	vcs.TIA.Plumb(env, nil, vcs.Mem.TIA, vcs.RIOT.Ports, vcs.CPU)
}

// GetSaveKey returns nil if no savekey is present
func (vcs cachedVCS) GetSaveKey() *savekey.SaveKey {
	sk, savekeyActive := vcs.RIOT.Ports.RightPlayer.(*savekey.SaveKey)
	if savekeyActive {
		return sk
	}
	vox, savekeyActive := vcs.RIOT.Ports.RightPlayer.(*atarivox.AtariVox)
	if savekeyActive {
		return vox.SaveKey.(*savekey.SaveKey)
	}
	return nil
}

// cachedVCS contains the cached components of the rewind system
type cachedRewind struct {
	Timeline   rewind.Timeline
	Comparison rewind.ComparisonState
}

// cachedDebugger contains the cached components of the debugger
type cachedDebugger struct {
	LiveDisasmEntry disassembly.Entry
	Breakpoints     debugger.CheckBreakpoints
	HaltReason      string
}

// cache is embedded in the Cache type and also used as the type carried by the
// queue channel
type cache struct {
	TV     *television.State
	VCS    cachedVCS
	env    *environment.Environment
	Rewind cachedRewind
	Dbg    cachedDebugger
}

// Cache contains the copied/snapshotting compenents of the emulated system
type Cache struct {
	queue chan cache
	cache
}

// NewCache is the preferred method of initialisation of the Cache type
func NewCache() Cache {
	return Cache{
		queue: make(chan cache, 1),
	}
}

// Update the cache. The update will complete on a future call to Resolve()
func (c *Cache) Update(vcs *hardware.VCS, rewind *rewind.Rewind, dbg *debugger.Debugger) {
	select {
	case c.queue <- cache{
		TV: vcs.TV.Snapshot(),
		VCS: cachedVCS{
			CPU:  vcs.CPU.Snapshot(),
			Mem:  vcs.Mem.Snapshot(),
			RIOT: vcs.RIOT.Snapshot(),
			TIA:  vcs.TIA.Snapshot(),
		},
		env: &environment.Environment{
			Label: "cache",
			Prefs: vcs.Env.Prefs,
			// no television required for gui purposes
		},
		Rewind: cachedRewind{
			Timeline:   rewind.GetTimeline(),
			Comparison: rewind.GetComparisonState(),
		},
		Dbg: cachedDebugger{
			LiveDisasmEntry: dbg.GetLiveDisasmEntry(),
			Breakpoints:     dbg.GetBreakpoints(),
			HaltReason:      dbg.GetHaltReason(),
		},
	}:
	default:
	}
}

// Resolve a previous call to Update(). Fine to call if there has been on
// previous call to Update()
func (c *Cache) Resolve() bool {
	select {
	case cache := <-c.queue:
		c.cache = cache
		c.cache.VCS.plumb(c.env)
	default:
	}
	return c.VCS.CPU != nil
}
