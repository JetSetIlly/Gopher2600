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

package reflection

// ReflectedInfo identifies specific information that has been reflected in a
// ReflectedVideoStep instance.
//
// We don't use this inside the reflection package itself but the type and
// associated values are useful and it makes sense to define them in this
// package.
type ReflectedInfo int

// List of valid ReflectedInfo values.
const (
	Hz ReflectedInfo = iota
	WSYNC
	Collision
	CXCLR
	HMOVEdelay
	HMOVEripple
	HMOVElatched
	RSYNCalign
	RSYNCreset
	AudioPhase0
	AudioPhase1
	AudioChanged

	// for graphical purposes we only distinguish between active and inactive
	// coprocessor states. the underlying states as defined in the mapper
	// package (mapper.CoProcSynchronisation) are used to decided whether the
	// coproc is active or inactive
	CoProcInactive
	CoProcActive
)

// Overlay is used to define the list of possible overlays that can be used to
// illustrate/visualise information in a ReflectedVideoStep instance.
//
// We don't use this inside the reflection package itself but the type and
// associated values are useful and it makes sense to define them in this
// package.
type Overlay int

// List of valid Overlay values.
const (
	OverlayNone Overlay = iota
	OverlayWSYNC
	OverlayCollision
	OverlayHMOVE
	OverlayRSYNC
	OverlayAudio
	OverlayCoproc
)

// OverlayLabels are names/labels for the the Policy type values.
var OverlayLabels = []string{"No overlay", "WSYNC", "Collisions", "HMOVE", "RSYNC", "Audio", "Coprocessor"}
