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

// Package rewind coordinates the periodic snapshotting of the emulation state.
// A previously seen frame can be returned to with the GotoFrame() function. If the
// requested frame is not in the history then the nearest frame that is will be
// used. The GotoLast() function will move the emulation to the last frame,
// whatever that might be.
//
// The frequency at which snapshots are made is definable. Frames that are in
// between the snapshots can still be plumbed in, the rewind package handling
// the interleaving by running the emulation for the missing period.
//
// In fact, the emulation is run even when the rewind package is set to
// snapshot every frame (a frequency of one) in order to generate the image
// data (storing the image data is memory wasteful). For this to work, the
// Rewind type is initialised with a reference to to the Runner interface.
// Implementations of the Runner interface should loop until the
// continueCheck() returns false.
//
// Regular emulation loops (ie. not catch-up loop of the Runner interface) must
// call Check() after every CPU instruction to catch frame boundaries as early
// as possible. The rewind package will take the snapshot at the appropriate
// time.
//
// The ExecutionState() function can be called to force a snapshot to be taken
// at any time. This should probably only ever be used when the emulation is
// paused. The rewind package will delete an execution snapshot when the next
// snapshot is taken (meaning that there is only ever one execution state in
// the history at any one time and that it will be at the end).
//
// Snapshots are stored in frame order from the splice point. The splice point
// will be wherever the snapshot history has been rewound to. For example, in a
// history of length 100 frames: the emulation has rewound back to frame 50.
// When the emulation is resumed, snapshots will be added at this point. The
// previous history of frames 51 to 100 will be lost.
//
// Reset and Boundary snapshots are special splice points that only ever occur
// once and at the beginning of the history. Reset occurs only when the Reset()
// function is called and is intended to be called whenver the VCS is power
// cycled (not the reset switch).
//
// The Boundrary snapshot occurs when history has been cleared for some other
// reason. This was added to better support PlusROM cartridges and to ensure
// that network events are not replayed. The rewind package will add the
// boundary snapshot (to an empty history) automatically whenever the
// RewindBoundary() function from the attached cartridge returns true.
//
// One of the weaknesses of the rewind package currently is the absence of any
// input replay. This might be particularly noticeable with large snapshot
// frequencies. Future versions of the package will record and replay input.
package rewind
