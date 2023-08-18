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

// Package caching is used for copying gopher2600 data so it can be used by the
// GUI goroutine safely.
//
// To update the cache the following form should be used:
//
//	PushFunction(func() {
//	    cache.Update(img.vcs)
//	})
//
//	if !cache.Resolve() {
//	    return
//	}
//
// The PushFunction() function indicates that the cache.Snapshot() function is
// run in the gopher2600 emulation goroutine. The PushFunction() function itself
// is run in the GUI routine.
//
// The cache.Resolve() function is run in the GUI goroutine. A return value of
// false indicates that the cache is not ready to be used.
//
// # History
//
// The caching package is a development of the now removed lazy package. The
// lazy package worked on a similar principle but was inflexible. It predated
// the development of the rewind package and performed much of the functionality
// of the rewind "Snapshot" concept in an adhoc manner. Now that the rewind
// package exists the caching pacakge repurposes the Snapshot() implementations
// for its own needs
package caching
