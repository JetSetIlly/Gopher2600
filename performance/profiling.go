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

package performance

import (
	"gopher2600/errors"
	"os"
	"runtime"
	"runtime/pprof"
)

// ProfileCPU runs supplied function "through" the pprof CPU profiler
func ProfileCPU(outFile string, run func() error) error {
	// write cpu profile
	f, err := os.Create(outFile)
	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}
	defer pprof.StopCPUProfile()

	return run()
}

// ProfileMem takes a snapshot of memory and writes to outFile
func ProfileMem(outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}
	runtime.GC()
	err = pprof.WriteHeapProfile(f)
	if err != nil {
		return errors.New(errors.PerformanceError, err)
	}
	f.Close()

	return nil
}
